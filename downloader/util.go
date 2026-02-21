package downloader

import (
	"errors"
	"iwaradl/api"
	"iwaradl/config"
	"iwaradl/util"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flytam/filenamify"
)

var VidList []string

// ParseUrl parse url to get vid or user
func ParseUrl(u string) (vid string, user string, err error) {
	util.DebugLog("Parsing URL: %s", u)
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" || parsed.Path == "" {
		return
	}
	host := parsed.Hostname()
	if !strings.Contains(host, "iwara.tv") {
		err = errors.New("website error")
		util.DebugLog("Invalid website host: %s", host)
		return
	}
	path := parsed.Path
	if strings.Contains(path, "/video/") {
		vid = strings.Split(path, "/")[2]
		user = ""
		util.DebugLog("Found video ID: %s", vid)
	} else if strings.Contains(path, "/profile/") {
		user = strings.Split(path, "/")[2]
		vid = ""
		util.DebugLog("Found user profile: %s", user)
	} else {
		err = errors.New("URL error")
		return
	}
	return
}

// PrepareFolder create folder for download
func PrepareFolder(username string) string {
	util.DebugLog("Preparing download folder for user: %s", username)
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

// ProcessUrlList get vid from video url or vid list from user url
func ProcessUrlList(urls []string) (vids []string) {
	util.DebugLog("Processing URL list with %d URLs", len(urls))

	for _, u := range urls {
		vid, user, err := ParseUrl(u)
		if err != nil {
			println(err.Error())
			continue
		}
		if vid != "" {
			vids = append(vids, vid)
			util.DebugLog("Added video ID to list: %s", vid)
		} else if user != "" {
			util.DebugLog("Fetching video list for user: %s", user)
			videos := api.GetVideoListByUser(user)
			for _, vi := range videos {
				vids = append(vids, vi.Id)
			}
		}
	}
	return vids
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
func SaveVidList() {
	util.DebugLog("Saving video list with %d items", len(VidList))
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
	VidList = removeDuplicate(VidList)
	for _, v := range VidList {
		_, err := file.WriteString(v + "\n")
		if err != nil {
			println(err.Error())
		}
	}
}

// LoadVidList load unfinished jobs from job list file
func LoadVidList() {
	util.DebugLog("Loading unfinished jobs list")
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
			VidList = append(VidList, v)
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
	util.DebugLog("Adding video to history: %s", vid)
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
	util.DebugLog("Checking history for video: %s", vid)
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
