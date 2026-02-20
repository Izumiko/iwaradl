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
	"github.com/flytam/filenamify"
)

type ProgressReport struct {
	VID           string
	BytesComplete int64
	BytesTotal    int64
	Done          bool
	Success       bool
}

var (
	progressHookMu sync.RWMutex
	progressHook   func(ProgressReport)
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

func vidFromFilename(filename string) string {
	base := filepath.Base(filename)
	if strings.HasSuffix(base, ".mp4") {
		base = strings.TrimSuffix(base, ".mp4")
	}
	if idx := strings.LastIndex(base, "-"); idx >= 0 && idx < len(base)-1 {
		return base[idx+1:]
	}
	return strings.TrimSpace(base)
}

// DoChanVid get vid from channel then grab info and download
func DoChanVid(c *grab.Client, vidch <-chan string, respch chan<- *grab.Response) {
	for vid := range vidch {
		util.DebugLog("Processing video ID: %s", vid)
		emptyReq, _ := grab.NewRequest(vid, "")
		vi, err := api.GetVideoInfo(vid)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- resp
			continue
		}
		u := api.GetVideoUrl(vi)
		if u == "" {
			util.DebugLog("Failed to get video URL for ID: %s", vid)
			println("Get video url " + vid + " failed")
			resp := c.Do(emptyReq)
			respch <- resp
			continue
		}
		title, path, err := WriteNfo(vi)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- resp
			continue
		}
		titleSafe, _ := filenamify.Filenamify(title, filenamify.Options{Replacement: "_", MaxLength: 64})
		filename := filepath.Join(path, titleSafe+"-"+vid+".mp4")
		util.DebugLog("Starting download: %s", filename)
		req, err := grab.NewRequest(filename, u)
		if err != nil {
			println(vid + ": " + err.Error())
			resp := c.Do(emptyReq)
			respch <- resp
			continue
		}
		resp := c.Do(req)
		respch <- resp
		<-resp.Done
	}
}

// ConcurrentDownload Get video info and download concurrently
func ConcurrentDownload() int {
	util.DebugLog("Starting concurrent download process")
	// Remove downloaded
	newList := make([]string, 0)
	newList = append(newList, VidList...)
	for i := 0; i < len(VidList); i++ {
		if FindHistory(VidList[i]) {
			println("Video " + VidList[i] + " already downloaded")
			newList = RemoveVid(newList, VidList[i])
			continue
		}
	}
	VidList = newList
	SaveVidList()

	vidch := make(chan string, len(VidList))
	respch := make(chan *grab.Response, len(VidList))

	// start client with proxy
	util.DebugLog("Initializing download client with %d threads", config.Cfg.ThreadNum)
	client := grab.NewClient()
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		println(err.Error())
		return 0
	}
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
		}
	}
	client.HTTPClient = &http.Client{Transport: tr}

	//start workers
	wg := sync.WaitGroup{}
	for i := 0; i < config.Cfg.ThreadNum; i++ {
		wg.Add(1)
		go func() {
			//client.DoChannel(reqch, respch)
			DoChanVid(client, vidch, respch)
			wg.Done()
		}()
	}

	go func() {
		//send vids
		for _, v := range VidList {
			vidch <- v
		}
		close(vidch)

		// wait for workers to finish
		wg.Wait()
		close(respch)
	}()

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	fmt.Print("\033[s")

	completed := 0
	succeeded := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)

	for completed < len(VidList) {
		select {
		case resp := <-respch:
			if resp != nil {
				responses = append(responses, resp)
			}
		case <-t.C:
			if inProgress > 0 {
				fmt.Printf("\033[%dA\033[K", inProgress)
			}
			inProgress = 0
			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					vid := vidFromFilename(resp.Filename)
					if resp.Err() == nil {
						fmt.Printf("Download saved to %v \n", resp.Filename)
						util.DebugLog("Download completed successfully: %s", vid)
						SaveHistory(vid)
						emitProgress(ProgressReport{VID: vid, BytesComplete: resp.Size(), BytesTotal: resp.Size(), Done: true, Success: true})
						succeeded++
					} else if resp.Request.HTTPRequest.Host != "" {
						util.DebugLog("Download failed: %s, error: %v", filepath.Base(resp.Filename), resp.Err())
						filename := filepath.Base(resp.Filename)
						_, _ = fmt.Fprintf(os.Stderr, "Download %v failed: %v\n", filename, resp.Err())
						emitProgress(ProgressReport{VID: vid, BytesComplete: resp.BytesComplete(), BytesTotal: resp.Size(), Done: true, Success: false})
					} else {
						emitProgress(ProgressReport{VID: vid, BytesComplete: resp.BytesComplete(), BytesTotal: resp.Size(), Done: true, Success: false})
					}
					responses[i] = nil
					completed++
				}
			}

			for _, resp := range responses {
				if resp != nil && !resp.IsComplete() {
					inProgress++
					filename := filepath.Base(resp.Filename)
					emitProgress(ProgressReport{VID: vidFromFilename(resp.Filename), BytesComplete: resp.BytesComplete(), BytesTotal: resp.Size(), Done: false, Success: false})
					fmt.Printf("Downloading %s %s / %s (%.2f%%)\033[K\n",
						filename, humanize.Bytes(uint64(resp.BytesComplete())), humanize.Bytes(uint64(resp.Size())), 100*resp.Progress())
				}
			}
		}
	}

	for i := 0; i < len(VidList); i++ {
		if FindHistory(VidList[i]) {
			VidList = RemoveVid(VidList, VidList[i])
			continue
		}
	}
	SaveVidList()

	fmt.Printf("%d files completed, %d successed and %d failed.\n", completed, succeeded, completed-succeeded)

	return completed - succeeded
}
