package api

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"iwaradl/config"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// Fetch the url and return the http response
func FetchResp(u string) (resp *http.Response, err error) {
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
		} else {
			return nil, errors.New("proxy URL scheme error")
		}
	}
	client := &http.Client{Transport: tr, Timeout: 6 * time.Second}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	if config.Cfg.Cookie != "" {
		req.Header.Set("Cookie", config.Cfg.Cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return
}

// Fetch the url and return the response body
func Fetch(u string) (data []byte, err error) {
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
		} else {
			return nil, errors.New("proxy URL scheme error")
		}
	}
	client := &http.Client{Transport: tr, Timeout: 6 * time.Second}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	if config.Cfg.Cookie != "" {
		req.Header.Set("Cookie", config.Cfg.Cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("HTTP status code error")
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return
}

// Get the username of the video by vid
func GetUserName(ecchi string, vid string) string {
	u := "https://" + ecchi + ".iwara.tv/videos/" + vid
	resp, err := Fetch(u)
	if err != nil {
		return ""
	}
	reg, _ := regexp.Compile(`class="username">(.+?)</a>`)
	username := reg.FindAllStringSubmatch(string(resp), -1)[0][1]

	return username
}

// Get the title of the video by vid
func GetVideoName(ecchi string, vid string) string {
	u := "https://" + ecchi + ".iwara.tv/videos/" + vid
	resp, err := Fetch(u)
	if err != nil {
		return ""
	}
	reg, _ := regexp.Compile(`class="title">(.+?)</h1>`)
	videoname := reg.FindAllStringSubmatch(string(resp), -1)[0][1]

	return videoname
}

// Get the username and title of the video by vid
func GetVideoInfo(ecchi string, vid string) (string, string) {
	u := "https://" + ecchi + ".iwara.tv/videos/" + vid
	resp, err := Fetch(u)
	if err != nil {
		return "", ""
	}
	reg, _ := regexp.Compile(`class="username">(.+?)</a>`)
	username := reg.FindAllStringSubmatch(string(resp), -1)[0][1]
	reg, _ = regexp.Compile(`class="title">(.+?)</h1>`)
	videoname := reg.FindAllStringSubmatch(string(resp), -1)[0][1]

	return username, videoname
}

type downloadInfo struct {
	Resolution string `json:"resolution"`
	Uri        string `json:"uri"`
	Mime       string `json:"mime"`
}

// Get the mp4 source url of the video by vid
func GetVideoUrl(ecchi string, vid string) string {
	u := "https://" + ecchi + ".iwara.tv/api/video/" + vid
	resp, err := Fetch(u)
	if err != nil {
		return ""
	}
	var dlList []downloadInfo
	err = json.Unmarshal(resp, &dlList)
	if err != nil {
		return ""
	}
	for _, v := range dlList {
		if v.Resolution == "Source" && v.Mime == "video/mp4" {
			return `https:` + v.Uri
		}
	}

	return ""
}

// Get the max page of the user's video list
func GetMaxPage(ecchi string, user string) int {
	u := "https://" + ecchi + ".iwara.tv/users/" + user + "/videos"
	resp, err := Fetch(u)
	if err != nil {
		return -1
	}
	reg, _ := regexp.Compile(`<li class="pager-last last"><a title=".+?" href="/users/.+?/videos\?.*?page=([0-9]{1,3})">`)
	maxPage := reg.FindAllStringSubmatch(string(resp), -1)
	if len(maxPage) == 0 {
		return 0
	} else {
		page, _ := strconv.Atoi(maxPage[0][1])
		return page
	}
}

// Get the video list of the user
func GetVideoList(ecchi string, user string) []string {
	u := "https://" + ecchi + ".iwara.tv/users/" + user
	resp, err := Fetch(u)
	if err != nil {
		return nil
	}
	reg1, _ := regexp.Compile(`class="more-link">.+?<a href="/users/`)
	reg2, _ := regexp.Compile(`class="title"><a href="/videos/(.+?)">.+?</a>`)
	hasMore := len(reg1.FindString(string(resp))) > 0
	var list []string
	if hasMore {
		maxPage := GetMaxPage(ecchi, user)
		for i := 0; i <= maxPage; i++ {
			u := "https://" + ecchi + ".iwara.tv/users/" + user + "/videos?page=" + strconv.Itoa(i)
			resp, err := Fetch(u)
			if err != nil {
				return nil
			}
			vidList := reg2.FindAllStringSubmatch(string(resp), -1)
			for _, v := range vidList {
				list = append(list, v[1])
			}
		}
	} else {
		vidList := reg2.FindAllStringSubmatch(string(resp), -1)
		for _, v := range vidList {
			list = append(list, v[1])
		}
	}

	return list
}

// Get the file size of the video by vid
func GetVideoSize(ecchi string, vid string) int64 {
	u := GetVideoUrl(ecchi, vid)
	resp, err := FetchResp(u)
	if err != nil {
		return -1
	}
	return resp.ContentLength
}

type DetailInfo struct {
	XMLName     xml.Name `xml:"musicvideo"`
	Author      string   `xml:"director"`
	VideoName   string   `xml:"title"`
	Description string   `xml:"plot"`
	ReleaseDate string   `xml:"releasedate"`
	Year        string   `xml:"year"`
	AddedDate   string   `xml:"dateadded"`
	Categories  []string `xml:"genre,omitempty"`
}

// Get the detail information of the video by vid
// including author, title, description, release date, year, categories
func GetDetailInfo(ecchi string, vid string) (DetailInfo, error) {
	u := "https://" + ecchi + ".iwara.tv/videos/" + vid + "?language=en"
	resp, err := Fetch(u)
	if err != nil {
		return DetailInfo{}, err
	}
	html := string(resp)
	// video name
	reg, _ := regexp.Compile(`class="title">(.+?)</h1>`)
	results := reg.FindAllStringSubmatch(html, -1)
	if len(results) == 0 {
		return DetailInfo{}, errors.New("video " + vid + " does not exist")
	}
	videoname := results[0][1]
	// author
	reg, _ = regexp.Compile(`class="username">(.+?)</a>`)
	username := reg.FindAllStringSubmatch(html, -1)[0][1]
	// description
	reg, _ = regexp.Compile(`(?s)<div class="field field-name-body field-type-text-with-summary field-label-hidden"><div class="field-items"><div class="field-item even">(.+?)</div></div></div>`)
	descriptions := reg.FindAllStringSubmatch(html, -1)
	description := ""
	if len(descriptions) > 0 {
		description = descriptions[0][1]
		reg, _ = regexp.Compile(`</p>|</br>|<br />`)
		description = reg.ReplaceAllString(description, "\n")
		reg, _ = regexp.Compile(`<.+?>`)
		description = reg.ReplaceAllString(description, "")
	}
	// date
	reg, _ = regexp.Compile(`class="username">.+?</a>.*?on.*?([0-9]{4}-[0-9]{2}-[0-9]{2})`)
	date := reg.FindAllStringSubmatch(html, -1)[0][1]
	year := date[:4]
	currentDate := time.Now().Format("2006-01-02 15:04:05")
	// categories
	reg, _ = regexp.Compile(`(?s)class="field field-name-field-categories field-type-taxonomy-term-reference field-label-hidden"><div class="field-items">(.+?)</div></div></div>`)
	cathtml := reg.FindAllStringSubmatch(html, -1)
	var categories []string
	if len(cathtml) > 0 {
		cathtml1 := cathtml[0][1]
		reg, _ = regexp.Compile(`<a href="/videos.+?">(.+?)</a>`)
		cats := reg.FindAllStringSubmatch(cathtml1, -1)
		for _, v := range cats {
			if v[1] == "Uncategorized" {
				continue
			}
			categories = append(categories, v[1])
		}
	}

	return DetailInfo{Author: username, VideoName: videoname, Description: description, ReleaseDate: date, Year: year, AddedDate: currentDate, Categories: categories}, nil
}
