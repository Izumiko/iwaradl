package downloader

import (
	"fmt"
	"iwaradl/api"
	"iwaradl/config"
	"iwaradl/util"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
)

type ProgressReport struct {
	VID           string
	BytesComplete int64
	BytesTotal    int64
	Done          bool
	Success       bool
}

type DownloadOptions struct {
	RootDir          string
	UseSubDir        bool
	UseSubDirSet     bool
	ProxyURL         string
	Cookie           string
	FilenameTemplate string
}

type downloadResult struct {
	VID  string
	Resp *grab.Response
}

var (
	progressHookMu sync.RWMutex
	progressHook   func(ProgressReport)
	runMu          sync.Mutex
)

func SetProgressHook(hook func(ProgressReport)) {
	progressHookMu.Lock()
	defer progressHookMu.Unlock()
	progressHook = hook
}

func emitProgress(report ProgressReport) {
	progressHookMu.RLock()
	hook := progressHook
	progressHookMu.RUnlock()
	if hook != nil {
		hook(report)
	}
}

func ConcurrentDownload() int {
	return ConcurrentDownloadWithOptions(DownloadOptions{})
}

func ConcurrentDownloadWithOptions(opts DownloadOptions) int {
	runMu.Lock()
	defer runMu.Unlock()

	origRootDir := config.Cfg.RootDir
	origUseSubDir := config.Cfg.UseSubDir
	origProxyURL := config.Cfg.ProxyUrl
	origFilenameTemplate := config.Cfg.FilenameTemplate

	if opts.UseSubDirSet {
		config.Cfg.UseSubDir = opts.UseSubDir
	}
	if opts.ProxyURL != "" {
		config.Cfg.ProxyUrl = opts.ProxyURL
	}
	if opts.FilenameTemplate != "" {
		config.Cfg.FilenameTemplate = opts.FilenameTemplate
	}

	result := api.ExecuteWithRuntimeOptions(config.Cfg.ProxyUrl, opts.Cookie, func() int {
		return concurrentDownloadOnce(opts)
	})

	config.Cfg.RootDir = origRootDir
	config.Cfg.UseSubDir = origUseSubDir
	config.Cfg.ProxyUrl = origProxyURL
	config.Cfg.FilenameTemplate = origFilenameTemplate

	return result
}

func DoChanVid(c *grab.Client, vidch <-chan string, respch chan<- downloadResult, opts DownloadOptions) {
	for vid := range vidch {
		util.DebugLog("Processing video ID: %s", vid)
		emptyReq, _ := grab.NewRequest(vid, "")
		vi, err := api.GetVideoInfo(vid)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- downloadResult{VID: vid, Resp: resp}
			continue
		}
		u, quality := api.GetVideoUrl(vi)
		if u == "" {
			util.DebugLog("Failed to get video URL for ID: %s", vid)
			println("Get video url " + vid + " failed")
			resp := c.Do(emptyReq)
			respch <- downloadResult{VID: vid, Resp: resp}
			continue
		}
		out, err := ResolveOutputPath(vi, quality, opts.RootDir, opts.FilenameTemplate)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- downloadResult{VID: vid, Resp: resp}
			continue
		}
		// generate nfo filename
		nfoFile := strings.TrimSuffix(out.FilePath, ".mp4") + ".nfo"
		_, _, err = WriteNfoToPath(vi, nfoFile)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- downloadResult{VID: vid, Resp: resp}
			continue
		}
		filename := out.FilePath
		util.DebugLog("Starting download: %s", filename)
		req, err := grab.NewRequest(filename, u)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- downloadResult{VID: vid, Resp: resp}
			continue
		}
		resp := c.Do(req)
		respch <- downloadResult{VID: vid, Resp: resp}
		<-resp.Done
	}
}

func concurrentDownloadOnce(opts DownloadOptions) int {
	util.DebugLog("Starting concurrent download process")
	newList := make([]string, 0)
	newList = append(newList, VidList...)
	for i := 0; i < len(VidList); i++ {
		if FindHistory(VidList[i]) {
			println("Video " + VidList[i] + " already downloaded")
			newList = RemoveVid(newList, VidList[i])
		}
	}
	VidList = newList
	SaveVidList()

	vidch := make(chan string, len(VidList))
	respch := make(chan downloadResult, len(VidList))

	util.DebugLog("Initializing download client with %d threads", config.Cfg.ThreadNum)
	client := grab.NewClient()
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if config.Cfg.ProxyUrl != "" {
		parsedURL, err := url.Parse(config.Cfg.ProxyUrl)
		if err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https" || parsedURL.Scheme == "socks5") {
			tr.Proxy = http.ProxyURL(parsedURL)
		}
	}
	client.HTTPClient = &http.Client{Transport: tr}

	wg := sync.WaitGroup{}
	for i := 0; i < config.Cfg.ThreadNum; i++ {
		wg.Add(1)
		go func() {
			DoChanVid(client, vidch, respch, opts)
			wg.Done()
		}()
	}

	go func() {
		for _, v := range VidList {
			vidch <- v
		}
		close(vidch)
		wg.Wait()
		close(respch)
	}()

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	fmt.Print("\033[s")

	completed := 0
	succeeded := 0
	inProgress := 0
	responses := make([]downloadResult, 0)

	for completed < len(VidList) {
		select {
		case item := <-respch:
			if item.Resp != nil {
				responses = append(responses, item)
			}
		case <-t.C:
			if inProgress > 0 {
				fmt.Printf("\033[%dA\033[K", inProgress)
			}
			inProgress = 0
			for i, item := range responses {
				resp := item.Resp
				if resp != nil && resp.IsComplete() {
					if resp.Err() == nil {
						fmt.Printf("Download saved to %v \n", resp.Filename)
						util.DebugLog("Download completed successfully: %s", item.VID)
						SaveHistory(item.VID)
						emitProgress(ProgressReport{VID: item.VID, BytesComplete: resp.Size(), BytesTotal: resp.Size(), Done: true, Success: true})
						succeeded++
					} else {
						filename := filepath.Base(resp.Filename)
						if resp.Request != nil && resp.Request.HTTPRequest != nil && resp.Request.HTTPRequest.Host != "" {
							util.DebugLog("Download failed: %s, error: %v", filename, resp.Err())
							_, _ = fmt.Fprintf(os.Stderr, "Download %v failed: %v\n", filename, resp.Err())
						}
						emitProgress(ProgressReport{VID: item.VID, BytesComplete: resp.BytesComplete(), BytesTotal: resp.Size(), Done: true, Success: false})
					}
					responses[i].Resp = nil
					completed++
				}
			}

			for _, item := range responses {
				resp := item.Resp
				if resp != nil && !resp.IsComplete() {
					inProgress++
					filename := filepath.Base(resp.Filename)
					emitProgress(ProgressReport{VID: item.VID, BytesComplete: resp.BytesComplete(), BytesTotal: resp.Size(), Done: false, Success: false})
					fmt.Printf("Downloading %s %s / %s (%.2f%%)\033[K\n",
						filename, humanize.Bytes(uint64(resp.BytesComplete())), humanize.Bytes(uint64(resp.Size())), 100*resp.Progress())
				}
			}
		}
	}

	for i := 0; i < len(VidList); i++ {
		if FindHistory(VidList[i]) {
			VidList = RemoveVid(VidList, VidList[i])
		}
	}
	SaveVidList()

	fmt.Printf("%d files completed, %d successed and %d failed.\n", completed, succeeded, completed-succeeded)
	return completed - succeeded
}
