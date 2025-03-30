package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"iwaradl/config"
	"iwaradl/util"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var Token string

// GetVideoInfo Get the video info json from the API server
func GetVideoInfo(id string) (info VideoInfo, err error) {
	util.DebugLog("开始获取视频信息，ID: %s", id)
	u := "https://api.iwara.tv/video/" + id
	body, err := Fetch(u, "")
	if err != nil {
		util.DebugLog("获取视频信息失败: %v", err)
		return
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		util.DebugLog("解析视频信息失败: %v", err)
		return
	}
	util.DebugLog("成功获取视频信息，标题: %s", info.Title)
	return
}

// Fetch the url and return the response body
func Fetch(u string, xversion string) (data []byte, err error) {
	util.DebugLog("开始请求URL: %s", u)
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		util.DebugLog("解析代理URL失败: %v", err)
		return nil, err
	}
	tr := &http.Transport{}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
			util.DebugLog("使用代理: %s", config.Cfg.ProxyUrl)
		} else {
			util.DebugLog("代理URL协议错误: %s", parsedUrl.Scheme)
			return nil, errors.New("proxy URL scheme error")
		}
	}
	client := &http.Client{Transport: tr, Timeout: 6 * time.Second}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		util.DebugLog("创建请求失败: %v", err)
		return nil, err
	}
	if config.Cfg.Authorization != "" && Token == "" {
		util.DebugLog("获取访问令牌")
		Token, err = GetAccessToken(config.Cfg.Authorization)
		if err != nil {
			util.DebugLog("获取访问令牌失败: %v", err)
			return
		}
	}
	if Token != "" {
		req.Header.Set("Authorization", "Bearer "+Token)
		util.DebugLog("设置Authorization头")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Origin", "https://www.iwara.tv")
	req.Header.Set("Referer", "https://www.iwara.tv/")
	if xversion != "" {
		req.Header.Set("X-Version", xversion)
		util.DebugLog("设置X-Version头: %s", xversion)
	}

	resp, err := client.Do(req)
	if err != nil {
		util.DebugLog("发送请求失败: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			util.DebugLog("关闭响应体失败: %v", err)
			return
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		util.DebugLog("HTTP状态码错误: %d", resp.StatusCode)
		return nil, errors.New("http status code: " + strconv.Itoa(resp.StatusCode))
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		util.DebugLog("读取响应体失败: %v", err)
		return nil, err
	}
	util.DebugLog("成功获取响应，数据长度: %d", len(data))
	return
}

func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

// GetVideoUrl Get the mp4 source url of the video info
func GetVideoUrl(vi VideoInfo) string {
	util.DebugLog("开始获取视频下载地址，ID: %s", vi.Id)
	u := vi.FileUrl
	parsed, err := url.Parse(u)
	if err != nil {
		util.DebugLog("解析文件URL失败: %v", err)
		return ""
	}
	expires := parsed.Query().Get("expires")
	xv := vi.File.Id + "_" + expires + "_5nFp9kmbNnHdAFhaqMvt"
	xversion := SHA1(xv)
	body, err := Fetch(u, xversion)
	if err != nil {
		util.DebugLog("获取视频地址失败: %v", err)
		return ""
	}
	var rList []ResolutionInfo
	err = json.Unmarshal(body, &rList)
	if err != nil {
		util.DebugLog("解析视频地址失败: %v", err)
		return ""
	}
	for _, v := range rList {
		if v.Name == "Source" {
			util.DebugLog("成功获取视频下载地址")
			return `https:` + v.Src.Download
		}
	}
	util.DebugLog("未找到源视频地址")
	return ""
}

// GetUserProfile Get user profile by username
func GetUserProfile(username string) (profile UserProfile, err error) {
	u := "https://api.iwara.tv/profile/" + username
	body, err := Fetch(u, "")
	err = json.Unmarshal(body, &profile)
	return
}

// GetMaxPage Get the max page of the user's video list
func GetMaxPage(uid string) int {
	u := "https://api.iwara.tv/videos?limit=8&user=" + uid
	body, err := Fetch(u, "")
	if err != nil {
		return -1
	}
	var vList VideoList
	err = json.Unmarshal(body, &vList)
	if err != nil {
		return -1
	}
	if vList.Count <= 0 {
		return 0
	} else if vList.Count <= 32 {
		return 1
	} else {
		return vList.Count/32 + 1
	}
}

// GetVideoList Get the video list of the user
func GetVideoList(username string) []VideoInfo {
	util.DebugLog("开始获取用户视频列表，用户名: %s", username)
	profile, err := GetUserProfile(username)
	if err != nil {
		util.DebugLog("获取用户信息失败: %v", err)
		return nil
	}
	uid := profile.User.Id
	maxPage := GetMaxPage(uid)
	util.DebugLog("用户ID: %s, 最大页数: %d", uid, maxPage)
	var list []VideoInfo
	for i := 0; i < maxPage; i++ {
		u := "https://api.iwara.tv/videos?page=" + strconv.Itoa(i) + "&sort=date&user=" + uid
		body, err := Fetch(u, "")
		if err != nil {
			util.DebugLog("获取第 %d 页失败: %v", i+1, err)
			continue
		}
		var vList VideoList
		err = json.Unmarshal(body, &vList)
		if err != nil {
			util.DebugLog("解析第 %d 页数据失败: %v", i+1, err)
			continue
		}
		for _, v := range vList.Results {
			list = append(list, v)
		}
		util.DebugLog("成功获取第 %d 页，当前共 %d 个视频", i+1, len(list))
	}
	util.DebugLog("完成获取用户视频列表，共 %d 个视频", len(list))
	return list
}

//
//// Get the file size of the video by vid
//func GetVideoSize(ecchi string, vid string) int64 {
//	u := GetVideoUrl(ecchi, vid)
//	resp, err := FetchResp(u)
//	if err != nil {
//		return -1
//	}
//	return resp.ContentLength
//}

// GetDetailInfo Get the detail information from video info
func GetDetailInfo(vi VideoInfo) (DetailInfo, error) {
	util.DebugLog("开始获取视频详细信息，ID: %s", vi.Id)
	var di DetailInfo
	di.Author = vi.User.Name
	di.VideoName = vi.Title
	di.Description = vi.Body
	di.ReleaseDate = vi.CreatedAt.Format("2006-01-02")
	di.Premiered = di.ReleaseDate
	di.Year = di.ReleaseDate[:4]
	di.AddedDate = time.Now().Format("2006-01-02 15:04:05")
	var categories []string
	for _, v := range vi.Tags {
		categories = append(categories, v.Id)
	}
	di.Categories = categories
	util.DebugLog("成功获取视频详细信息，标题: %s", di.VideoName)
	return di, nil
}

// GetAccessToken Get access token
func GetAccessToken(auth string) (string, error) {
	u := "https://api.iwara.tv/user/token"
	parsedUrl, err := url.Parse(config.Cfg.ProxyUrl)
	if err != nil {
		return "", err
	}
	tr := &http.Transport{}
	if config.Cfg.ProxyUrl != "" {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			tr.Proxy = http.ProxyURL(parsedUrl)
		} else {
			return "", errors.New("proxy URL scheme error")
		}
	}

	client := &http.Client{Transport: tr, Timeout: 6 * time.Second}

	req, err := http.NewRequest("POST", u, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://www.iwara.tv")
	req.Header.Set("Referer", "https://www.iwara.tv/")
	req.Header.Set("Authorization", "Bearer "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			println(err.Error())
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		return "", errors.New("status code error: " + strconv.Itoa(resp.StatusCode))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type Token struct {
		AccessToken string `json:"accessToken"`
	}

	var token Token
	err = json.Unmarshal(data, &token)
	return token.AccessToken, nil
}
