package cmd

import (
	"encoding/xml"
	"errors"
	"fmt"
	"iwaradl/api"
	"iwaradl/config"
	"iwaradl/util"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
	"github.com/flytam/filenamify"
	"github.com/spf13/cobra"
)

var (
	configFile string
	listFile   string
	resumeJob  bool
	debug      bool
	rootDir    string
	useSubDir  bool
	auth       string
	proxyUrl   string
	threadNum  int
	maxRetry   int
	vidList    []string
)

// ParseUrl parse url to get vid or user
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

// prepareFolder create folder for download
func prepareFolder(username string) string {
	path := config.Cfg.RootDir
	err := os.Mkdir(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		println(err.Error())
	}
	if config.Cfg.UseSubDir && username != "" {
		subfolder, _ := filenamify.Filenamify(username, filenamify.Options{Replacement: "_", MaxLength: 64})
		path = filepath.Join(path, subfolder)
		err = os.Mkdir(path, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			println(err.Error())
		}
	}
	return path
}

// processUrlList get vid from video url or vid list from user url
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

// removeDuplicate remove duplicate items in slice
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

// SaveVidList save current jobs to job list file
func SaveVidList(uList []string) {
	_ = os.Mkdir(config.Cfg.RootDir, 0755)
	urlFile := filepath.Join(config.Cfg.RootDir, "jobs.list")
	file, err := os.OpenFile(urlFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		println(err.Error())
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	vidList = removeDuplicate(uList)
	for _, v := range vidList {
		_, err := file.WriteString(v + "\n")
		if err != nil {
			println(err.Error())
		}
	}
}

// LoadVidList load unfinished jobs from job list file
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

// RemoveVid remove video id from list
func RemoveVid(list []string, vid string) (l []string) {
	l = make([]string, 0)
	for _, vi := range list {
		if vi != vid {
			l = append(l, vi)
		}
	}
	return
}

// SaveHistory save video id to history file
func SaveHistory(vid string) {
	historyFile := filepath.Join(config.Cfg.RootDir, "history.list")
	file, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		println(err.Error())
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_, err = file.WriteString(vid + "\n")
	if err != nil {
		println(err.Error())
		return
	}
}

// FindHistory check if video is downloaded
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

// WriteNfo get video detail info and write to Jellyfin nfo file
func WriteNfo(vi api.VideoInfo) (title string, path string, err error) {
	detailInfo, err := api.GetDetailInfo(vi)
	if err != nil {
		return "", "", err
	}

	// add <br> to description
	detailInfo.Description = strings.ReplaceAll(detailInfo.Description, "\n", "<br/>\n")

	path = prepareFolder(detailInfo.Author)
	titleSafe, _ := filenamify.Filenamify(detailInfo.VideoName, filenamify.Options{Replacement: "_", MaxLength: 64})
	filename := filepath.Join(path, titleSafe+"-"+vi.Id+".nfo")
	f, err := os.Create(filename)
	if err != nil {
		println(err.Error())
		return "", "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
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

// DoChanVid get vid from channel then grab info and download
func DoChanVid(c *grab.Client, vidch <-chan string, respch chan<- *grab.Response) {
	for vid := range vidch {
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

	vidch := make(chan string, len(vidList))
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
			//client.DoChannel(reqch, respch)
			DoChanVid(client, vidch, respch)
			wg.Done()
		}()
	}

	go func() {
		//send vids
		for _, v := range vidList {
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
					if resp.Err() == nil {
						fmt.Printf("Download saved to %v \n", resp.Filename)
						paths := strings.Split(resp.Filename[:len(resp.Filename)-4], "-")
						SaveHistory(paths[len(paths)-1])
						succeeded++
					} else if resp.Request.HTTPRequest.Host != "" {
						filename := filepath.Base(resp.Filename)
						_, _ = fmt.Fprintf(os.Stderr, "Download %v failed: %v\n", filename, resp.Err())
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

	fmt.Printf("%d files completed, %d successed and %d failed.\n", completed, succeeded, completed-succeeded)

	return completed - succeeded
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "iwaradl [flags] [URL...]",
	Short: "A downloader for iwara.tv",
	Long: `A downloader for iwara.tv that supports:
- Multiple URLs download
- URL list file
- Resume unfinished downloads
- Custom download directory
- Proxy support`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !resumeJob && len(args) == 0 && listFile == "" {
			return cmd.Help()
		}

		err := config.LoadConfig(&config.Cfg, configFile)
		if err != nil {
			return err
		}

		// 命令行参数优先级高于配置文件
		if rootDir != "" {
			config.Cfg.RootDir = rootDir
		}
		if useSubDir {
			config.Cfg.UseSubDir = useSubDir
		}
		if auth != "" {
			config.Cfg.Authorization = auth
		}
		if proxyUrl != "" {
			config.Cfg.ProxyUrl = proxyUrl
		}
		if threadNum > 0 {
			config.Cfg.ThreadNum = threadNum
		}
		if maxRetry > 0 {
			config.Cfg.MaxRetry = maxRetry
		}

		if debug {
			util.Debug = true
		}

		// 处理下载任务
		if resumeJob {
			vidList = LoadVidList()
		}
		if len(args) > 0 {
			processUrlList(args)
		}
		if listFile != "" {
			_, err := os.Stat(listFile)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(listFile)
			if err != nil {
				return err
			}
			urls := strings.Split(string(data), "\n")
			for i, v := range urls {
				urls[i] = strings.TrimRight(v, "\r")
			}
			processUrlList(urls)
		}
		SaveVidList(vidList)

		failed := len(vidList)
		for i := 0; i < config.Cfg.MaxRetry && failed > 0; i++ {
			failed = ConcurrentDownload()
			if failed > 0 && i < config.Cfg.MaxRetry-1 {
				time.Sleep(30 * time.Second)
			}
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&listFile, "list-file", "l", "", "URL list file")
	rootCmd.PersistentFlags().BoolVarP(&resumeJob, "resume", "r", false, "resume unfinished job")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&rootDir, "root-dir", "", "root directory for videos")
	rootCmd.PersistentFlags().BoolVar(&useSubDir, "use-sub-dir", false, "use user name as sub directory")
	rootCmd.PersistentFlags().StringVar(&auth, "auth-token", "", "authorization token")
	rootCmd.PersistentFlags().StringVar(&proxyUrl, "proxy-url", "", "proxy url")
	rootCmd.PersistentFlags().IntVar(&threadNum, "thread-num", -1, "concurrent download thread number (default 3)")
	rootCmd.PersistentFlags().IntVar(&maxRetry, "max-retry", -1, "max retry times (default 3)")
}
