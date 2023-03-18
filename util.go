package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
	"github.com/flytam/filenamify"
	"iwaradl/api"
	"iwaradl/config"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func ParseUrl(u string) (vid string, user string, err error) {
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" || parsed.Path == "" {
		return
	}
	host := parsed.Hostname()
	if !strings.Contains(host, "iwara.tv") {
		err = errors.New("website error")
		return
	}
	path := parsed.Path
	if strings.Contains(path, "/video/") {
		vid = strings.Split(path, "/")[2]
		user = ""
	} else if strings.Contains(path, "/profile/") {
		user = strings.Split(path, "/")[2]
		vid = ""
	} else {
		err = errors.New("URL error")
		return
	}
	return
}

//func DownloadVideo(vi VideoInfo) {
//	if FindHistory(vi.Vid) {
//		println("Video " + vi.Vid + " already downloaded")
//		return
//	}
//	user, title := api.GetVideoInfo(vi.Ecchi, vi.Vid)
//	path := prepareFolder(user)
//	titleSafe, _ := filenamify.Filenamify(title, filenamify.Options{Replacement: "_"})
//	filename := filepath.Join(path, titleSafe+"-"+vi.Vid+".mp4")
//	// check if file exists
//	finfo, err := os.Stat(filename)
//	if err == nil {
//		videoSize := api.GetVideoSize(vi.Ecchi, vi.Vid)
//		fileSize := finfo.Size()
//		if videoSize == fileSize {
//			println("Video " + vi.Vid + " already downloaded")
//			SaveHistory(vi.Vid)
//			return
//		} else {
//			err = DownloadFile(vi.Ecchi, vi.Vid, filename)
//			if err != nil {
//				println(err.Error())
//			} else {
//				SaveHistory(vi.Vid)
//			}
//		}
//	} else {
//		err = DownloadFile(vi.Ecchi, vi.Vid, filename)
//		if err != nil {
//			println(err.Error())
//		} else {
//			SaveHistory(vi.Vid)
//		}
//	}
//}

func prepareFolder(username string) string {
	path := config.Cfg.RootDir
	err := os.Mkdir(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		println(err.Error())
	}
	if config.Cfg.UseSubDir && username != "" {
		subfolder, _ := filenamify.Filenamify(username, filenamify.Options{Replacement: "_"})
		path = filepath.Join(path, subfolder)
		err = os.Mkdir(path, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			println(err.Error())
		}
	}
	return path
}

func processUrlList(urls []string) {
	for _, u := range urls {
		vid, user, err := ParseUrl(u)
		if err != nil {
			println(err.Error())
			continue
		}
		if vid != "" {
			vidList = append(vidList, vid)
		} else if user != "" {
			videos := api.GetVideoList(user)
			for _, vi := range videos {
				vidList = append(vidList, vi.Id)
			}
		}
	}
}

func removeDuplicate(sliceList []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func SaveVidList(uList []string) {
	_ = os.Mkdir(config.Cfg.RootDir, 0755)
	urlFile := filepath.Join(config.Cfg.RootDir, "jobs.list")
	file, err := os.OpenFile(urlFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		println(err.Error())
	}
	defer file.Close()
	vidList = removeDuplicate(uList)
	for _, v := range vidList {
		_, err := file.WriteString(v + "\n")
		if err != nil {
			println(err.Error())
		}
	}
}
func LoadVidList() (vidList []string) {
	vidFile := filepath.Join(config.Cfg.RootDir, "jobs.list")
	_, err := os.Stat(vidFile)
	if err != nil {
		return
	}
	data, err := os.ReadFile(vidFile)
	if err != nil {
		return
	}
	vl := strings.Split(string(data), "\n")
	for _, v := range vl {
		if v != "" {
			vidList = append(vidList, v)
		}
	}
	return
}

func RemoveVid(list []string, vid string) (l []string) {
	l = make([]string, 0)
	for _, vi := range list {
		if vi != vid {
			l = append(l, vi)
		}
	}
	return
}

func SaveHistory(vid string) {
	historyFile := filepath.Join(config.Cfg.RootDir, "history.list")
	file, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		println(err.Error())
	}
	defer file.Close()
	_, err = file.WriteString(vid + "\n")
	if err != nil {
		println(err.Error())
		return
	}
}

func FindHistory(vid string) bool {
	historyFile := filepath.Join(config.Cfg.RootDir, "history.list")
	_, err := os.Stat(historyFile)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return false
	}
	vids := strings.Split(string(data), "\n")
	sort.Strings(vids)
	i := sort.SearchStrings(vids, vid)
	if i >= 0 && i < len(vids) && vids[i] == vid {
		return true
	}
	return false
}

func DownloadFile(vid string, filename string) error {
	vi, err := api.GetVideoInfo(vid)
	u := api.GetVideoUrl(vi)
	client := grab.NewClient()
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		return err
	}
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
		}
	}
	client.HTTPClient = &http.Client{Transport: tr}
	req, err := grab.NewRequest(filename, u)

	fmt.Printf("Downloading %v...\n", vid)
	resp := client.Do(req)
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	fmt.Print("\033[s")
Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf("\033[u\033[K  transferred %s / %s (%.2f%%)\n",
				humanize.Bytes(uint64(resp.BytesComplete())),
				humanize.Bytes(uint64(resp.Size())),
				100*resp.Progress())
		case <-resp.Done:
			break Loop
		}
	}
	if err := resp.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		return err
	}
	fmt.Printf("Download saved to ./%v \n", resp.Filename)
	return nil
}

// Get all video info, then download concurrently
func ConcurrentDownload() int {
	reqs := make([]*grab.Request, 0)
	newList := make([]string, 0)
	newList = append(newList, vidList...)
	noinfo := 0
	for i := 0; i < len(vidList); i++ {
		if FindHistory(vidList[i]) {
			println("Video " + vidList[i] + " already downloaded")
			newList = RemoveVid(newList, vidList[i])
			continue
		}
		println("Getting video info: " + vidList[i] + " ...")
		vi, err := api.GetVideoInfo(vidList[i])
		if err != nil {
			println(err.Error())
			noinfo++
			continue
		}
		u := api.GetVideoUrl(vi)
		if u == "" {
			println("Get video url " + vidList[i] + " failed")
			noinfo++
			continue
		}
		title, path, err := WriteNfo(vi)
		if err != nil {
			println(err.Error())
			noinfo++
			continue
		}
		titleSafe, _ := filenamify.Filenamify(title, filenamify.Options{Replacement: "_"})
		filename := filepath.Join(path, titleSafe+"-"+vidList[i]+".mp4")
		req, err := grab.NewRequest(filename, u)
		if err != nil {
			println(err.Error())
			continue
		}
		reqs = append(reqs, req)
	}
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
	respch := client.DoBatch(config.Cfg.ThreadNum, reqs...)

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	fmt.Print("\033[s")
	// varibles for progress
	completed := 0
	failed := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)

	for completed < len(reqs) {
		select {
		case resp := <-respch:
			if resp != nil {
				responses = append(responses, resp)
			}
		case <-t.C:
			if inProgress > 0 {
				fmt.Printf("\033[%dA\033[K", inProgress)
			}
			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					if resp.Err() != nil {
						filename := filepath.Base(resp.Filename)
						fmt.Fprintf(os.Stderr, "Download %v failed: %v\n", filename, resp.Err())
						failed++
					} else {
						fmt.Printf("Download saved to %v \n", resp.Filename)
						paths := strings.Split(resp.Filename[:len(resp.Filename)-4], "-")
						SaveHistory(paths[len(paths)-1])
						newList = RemoveVid(newList, paths[len(paths)-1])
						time.Sleep(10 * time.Second)
					}
					responses[i] = nil
					completed++
				}
			}
			inProgress = 0
			for _, resp := range responses {
				if resp != nil && !resp.IsComplete() {
					inProgress++
					filename := filepath.Base(resp.Filename)
					fmt.Printf("Downloading %s %s / %s (%.2f%%)\033[K\n",
						filename, humanize.Bytes(uint64(resp.BytesComplete())), humanize.Bytes(uint64(resp.Size())), 100*resp.Progress())
				}
			}
		}
	}
	fmt.Printf("%d files completed, %d successed and %d failed.\n", completed, completed-failed, failed+noinfo)

	SaveVidList(newList)
	vidList = newList
	return failed + noinfo
}

func WriteNfo(vi api.VideoInfo) (title string, path string, err error) {
	detailInfo, err := api.GetDetailInfo(vi)
	if err != nil {
		return "", "", err
	}
	path = prepareFolder(detailInfo.Author)
	titleSafe, _ := filenamify.Filenamify(detailInfo.VideoName, filenamify.Options{Replacement: "_"})
	filename := filepath.Join(path, titleSafe+"-"+vi.Id+".nfo")
	f, err := os.Create(filename)
	if err != nil {
		println(err.Error())
		return "", "", err
	}
	defer f.Close()
	// write xml header
	_, err = f.WriteString(xml.Header)
	if err != nil {
		return "", "", err
	}
	// marshal
	b, err := xml.MarshalIndent(detailInfo, "", "  ")
	if err != nil {
		println(err.Error())
		return "", "", err
	}
	_, err = f.Write(b)
	if err != nil {
		return "", "", err
	}

	return detailInfo.VideoName, path, nil
}

// Get video info and download concurrently
func ConcurrentDownload2() int {
	// Remove downloaded
	newList := make([]string, 0)
	newList = append(newList, vidList...)
	for i := 0; i < len(vidList); i++ {
		if FindHistory(vidList[i]) {
			println("Video " + vidList[i] + " already downloaded")
			newList = RemoveVid(newList, vidList[i])
			continue
		}
	}
	vidList = newList
	SaveVidList(vidList)

	reqch := make(chan *grab.Request, len(vidList))
	respch := make(chan *grab.Response, len(vidList))

	// start client with proxy
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
			client.DoChannel(reqch, respch)
			wg.Done()
		}()
	}

	failed := 0
	emptyReq, _ := grab.NewRequest(config.Cfg.RootDir, "")

	go func() {
		// send requests
		for i := 0; i < len(vidList); i++ {
			vi, err := api.GetVideoInfo(vidList[i])
			if err != nil {
				println(err.Error())
				failed++
				reqch <- emptyReq
				continue
			}
			u := api.GetVideoUrl(vi)
			if u == "" {
				println("Get video url " + vidList[i] + " failed")
				failed++
				reqch <- emptyReq
				continue
			}
			title, path, err := WriteNfo(vi)
			if err != nil {
				println(err.Error())
				failed++
				reqch <- emptyReq
				continue
			}
			titleSafe, _ := filenamify.Filenamify(title, filenamify.Options{Replacement: "_"})
			filename := filepath.Join(path, titleSafe+"-"+vidList[i]+".mp4")
			req, err := grab.NewRequest(filename, u)
			if err != nil {
				println(err.Error())
				continue
			}

			reqch <- req
		}
		close(reqch)

		// wait for workers to finish
		wg.Wait()
		close(respch)
	}()

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	fmt.Print("\033[s")

	completed := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)

	for completed < len(vidList) {
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
					if resp.Err() != nil {
						filename := filepath.Base(resp.Filename)
						fmt.Fprintf(os.Stderr, "Download %v failed: %v\n", filename, resp.Err())
						failed++
					} else {
						fmt.Printf("Download saved to %v \n", resp.Filename)
						paths := strings.Split(resp.Filename[:len(resp.Filename)-4], "-")
						SaveHistory(paths[len(paths)-1])
					}
					responses[i] = nil
					completed++
				}
			}

			for _, resp := range responses {
				if resp != nil && !resp.IsComplete() {
					inProgress++
					filename := filepath.Base(resp.Filename)
					fmt.Printf("Downloading %s %s / %s (%.2f%%)\033[K\n",
						filename, humanize.Bytes(uint64(resp.BytesComplete())), humanize.Bytes(uint64(resp.Size())), 100*resp.Progress())
				}
			}
		}
	}

	for i := 0; i < len(vidList); i++ {
		if FindHistory(vidList[i]) {
			vidList = RemoveVid(vidList, vidList[i])
			continue
		}
	}
	SaveVidList(vidList)

	return failed
}
