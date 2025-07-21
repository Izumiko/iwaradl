package api

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	http "github.com/bogdanfinn/fhttp"
	tlsClient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"io"
	"iwaradl/config"
	"iwaradl/util"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	Token       string
	Client      tlsClient.HttpClient
	commHeaders = http.Header{
		"Accept":          {"application/json"},
		"Accept-Language": {"en-US,en;q=0.5"},
		"Content-Type":    {"application/json"},
		"User-Agent":      {"Mozilla/5.0 (Windows 11 x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.6943.99 Safari/537.36"},
		"Origin":          {"https://www.iwara.tv"},
		"Referer":         {"https://www.iwara.tv/"},
	}
)

func initClient() error {
	var err error
	Client, err = tlsClient.NewHttpClient(tlsClient.NewNoopLogger(), tlsClient.WithClientProfile(profiles.Chrome_133))
	if err != nil {
		return err
	}
	if config.Cfg.ProxyUrl != "" && strings.HasPrefix(config.Cfg.ProxyUrl, "http") {
		err := Client.SetProxy(config.Cfg.ProxyUrl)
		if err != nil {
			return err
		}
		util.DebugLog("Using proxy: %s", config.Cfg.ProxyUrl)
	}
	return nil
}

func init() {
	if err := initClient(); err != nil {
		panic(err)
	}
}

// GetVideoInfo Get the video info json from the API server
func GetVideoInfo(id string) (info VideoInfo, err error) {
	util.DebugLog("Starting to get video info, ID: %s", id)
	u := "https://api.iwara.tv/video/" + id
	body, err := Fetch(u, "")
	if err != nil {
		util.DebugLog("Failed to get video info: %v", err)
		return
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		util.DebugLog("Failed to parse video info: %v", err)
		return
	}
	util.DebugLog("Successfully got video info, title: %s", info.Title)
	return
}

// Fetch the url and return the response body
func Fetch(u string, xversion string) (data []byte, err error) {
	util.DebugLog("Starting to request URL: %s", u)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		util.DebugLog("Failed to create request: %v", err)
		return nil, err
	}

	req.Header = commHeaders

	if (config.Cfg.Authorization != "" || config.Cfg.Email != "") && Token == "" {
		util.DebugLog("Getting access token")
		Token, err = GetAccessToken(config.Cfg.Authorization)
		if err != nil {
			// Try to refresh the authorization token
			if config.Cfg.Email != "" && config.Cfg.Password != "" {
				newAuth, refreshErr := RefreshAuthToken()
				if refreshErr == nil {
					Token, err = GetAccessToken(newAuth)
				} else {
					util.DebugLog("Failed to refresh authorization token: %v", refreshErr)
					return nil, refreshErr
				}
			}
		}
	}
	if Token != "" {
		req.Header.Set("Authorization", "Bearer "+Token)
		util.DebugLog("Setting Authorization header")
	}

	if xversion != "" {
		req.Header.Set("X-Version", xversion)
		util.DebugLog("Setting X-Version header: %s", xversion)
	}

	resp, err := Client.Do(req)
	if err != nil {
		util.DebugLog("Failed to send request: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			util.DebugLog("Failed to close response body: %v", err)
			return
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		util.DebugLog("Invalid HTTP status code: %d", resp.StatusCode)
		return nil, errors.New("http status code: " + strconv.Itoa(resp.StatusCode))
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		util.DebugLog("Failed to read response body: %v", err)
		return nil, err
	}
	util.DebugLog("Successfully got response, data length: %d", len(data))
	return
}

func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

// GetVideoUrl Get the mp4 source url of the video info
func GetVideoUrl(vi VideoInfo) string {
	util.DebugLog("Starting to get video download URL, ID: %s", vi.Id)
	u := vi.FileUrl
	parsed, err := url.Parse(u)
	if err != nil {
		util.DebugLog("Failed to parse file URL: %v", err)
		return ""
	}
	expires := parsed.Query().Get("expires")
	xv := vi.File.Id + "_" + expires + "_5nFp9kmbNnHdAFhaqMvt"
	xversion := SHA1(xv)
	body, err := Fetch(u, xversion)
	if err != nil {
		util.DebugLog("Failed to get video URL: %v", err)
		return ""
	}
	var rList []ResolutionInfo
	err = json.Unmarshal(body, &rList)
	if err != nil {
		util.DebugLog("Failed to parse video URL: %v", err)
		return ""
	}
	for _, v := range rList {
		if v.Name == "Source" {
			util.DebugLog("Successfully got video download URL")
			return `https:` + v.Src.Download
		}
	}
	util.DebugLog("Source video URL not found")
	return ""
}

// GetUserProfile Get user profile by username
func GetUserProfile(username string) (profile UserProfile, err error) {
	u := "https://api.iwara.tv/profile/" + username
	body, err := Fetch(u, "")
	if err != nil {
		util.DebugLog("Failed to get user profile: %v", err)
		return
	}
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
	util.DebugLog("Starting to get user video list, username: %s", username)
	profile, err := GetUserProfile(username)
	if err != nil {
		util.DebugLog("Failed to get user info: %v", err)
		return nil
	}
	uid := profile.User.Id
	maxPage := GetMaxPage(uid)
	util.DebugLog("User ID: %s, max pages: %d", uid, maxPage)
	var list []VideoInfo
	for i := 0; i < maxPage; i++ {
		u := "https://api.iwara.tv/videos?page=" + strconv.Itoa(i) + "&sort=date&user=" + uid
		body, err := Fetch(u, "")
		if err != nil {
			util.DebugLog("Failed to get page %d: %v", i+1, err)
			continue
		}
		var vList VideoList
		err = json.Unmarshal(body, &vList)
		if err != nil {
			util.DebugLog("Failed to parse page %d data: %v", i+1, err)
			continue
		}
		list = append(list, vList.Results...)
		util.DebugLog("Successfully got page %d, current total videos: %d", i+1, len(list))
	}
	util.DebugLog("Completed getting user video list, total videos: %d", len(list))
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
	util.DebugLog("Starting to get video details, ID: %s", vi.Id)
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
	util.DebugLog("Successfully got video details, title: %s", di.VideoName)
	return di, nil
}

// GetAccessToken Get access token using authorization token
func GetAccessToken(auth string) (string, error) {
	u := "https://api.iwara.tv/user/token"

	req, err := http.NewRequest("POST", u, nil)
	if err != nil {
		return "", err
	}

	req.Header = commHeaders

	req.Header.Set("Authorization", "Bearer "+auth)

	resp, err := Client.Do(req)
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
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// RefreshAuthToken Refresh Authorization Token with username and password
func RefreshAuthToken() (string, error) {
	u := "https://api.iwara.tv/user/login"

	body := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    config.Cfg.Email,
		Password: config.Cfg.Password,
	}
	bodyData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", u, bytes.NewBuffer(bodyData))
	if err != nil {
		return "", err
	}

	req.Header = commHeaders

	resp, err := Client.Do(req)
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
		AuthToken string `json:"token"`
	}

	var token Token
	err = json.Unmarshal(data, &token)
	if err != nil {
		return "", err
	}

	config.Cfg.Authorization = token.AuthToken
	if err := config.SaveConfig(&config.Cfg); err != nil {
		util.DebugLog("Failed to save config: %v", err)
	}

	return token.AuthToken, nil
}
