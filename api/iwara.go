package api

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iwaradl/config"
	"iwaradl/util"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tlsClient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

var (
	Token         string
	Client        tlsClient.HttpClient
	runtimeCookie string
	runtimeMu     sync.Mutex
	commHeaders   = http.Header{
		"accept":             {"application/json, text/plain, */*"},
		"accept-encoding":    {"gzip, deflate, br, zstd"},
		"accept-language":    {"zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7,ja;q=0.6"},
		"priority":           {"u=1, i"},
		"sec-ch-ua":          {`"Google Chrome";v="146", "Not_A Brand";v="8", "Chromium";v="146"`},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {`"Windows"`},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-site"},
		"sec-gpc":            {"1"},
		"user-agent":         {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {
			"accept",
			"accept-encoding",
			"accept-language",
			"x-site",
			"origin",
			"priority",
			"referer",
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-platform",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"user-agent",
		},
	}
)

func SwitchHeaders(host string) http.Header {
	headers := make(http.Header)
	for k, v := range commHeaders {
		headers[k] = append([]string(nil), v...)
	}
	headers.Set("x-site", host)
	headers.Set("origin", "https://"+host)
	headers.Set("referer", "https://"+host+"/")
	return headers
}

func defaultClientProfile() profiles.ClientProfile {
	return profiles.Chrome_146_PSK
}

func initClient(proxyURL string) error {
	var err error
	options := []tlsClient.HttpClientOption{
		tlsClient.WithTimeoutSeconds(60),
		tlsClient.WithClientProfile(defaultClientProfile()),
		//tlsClient.WithNotFollowRedirects(),
		tlsClient.WithCookieJar(tlsClient.NewCookieJar()),
		// tls_client.WithInsecureSkipVerify(),
	}
	if proxyURL != "" && (strings.HasPrefix(proxyURL, "http") || strings.HasPrefix(proxyURL, "socks5")) {
		options = append(options, tlsClient.WithProxyUrl(proxyURL))
		util.DebugLog("Using proxy: %s", proxyURL)
	}
	Client, err = tlsClient.NewHttpClient(tlsClient.NewNoopLogger(), options...)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	if err := initClient(config.Cfg.ProxyUrl); err != nil {
		panic(err)
	}
}

func ExecuteWithRuntimeOptions(proxyURL string, cookie string, fn func() int) int {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	prevCookie := runtimeCookie
	runtimeCookie = cookie

	if err := initClient(proxyURL); err != nil {
		runtimeCookie = prevCookie
		_ = initClient(config.Cfg.ProxyUrl)
		return fn()
	}

	result := fn()

	runtimeCookie = prevCookie
	_ = initClient(config.Cfg.ProxyUrl)
	return result
}

// GetVideoInfo Get the video info JSON from the API server
func GetVideoInfo(id string, host string) (info VideoInfo, err error) {
	util.DebugLog("Starting to get video info, ID: %s", id)
	u := "https://api.iwara.tv/video/" + id
	body, err := Fetch(u, "", host)
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
func Fetch(u string, xversion string, host string) (data []byte, err error) {
	util.DebugLog("Starting to request URL: %s", u)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		util.DebugLog("Failed to create request: %v", err)
		return nil, err
	}

	req.Header = make(http.Header)
	for k, v := range SwitchHeaders(host) {
		req.Header[k] = append([]string(nil), v...)
	}

	if (config.Cfg.Authorization != "" || config.Cfg.Email != "") && Token == "" {
		util.DebugLog("Getting access token")
		Token, err = GetAccessToken(config.Cfg.Authorization, host)
		if err != nil {
			// Try to refresh the authorization token
			if config.Cfg.Email != "" && config.Cfg.Password != "" {
				newAuth, refreshErr := RefreshAuthToken(host)
				if refreshErr == nil {
					Token, err = GetAccessToken(newAuth, host)
				} else {
					util.DebugLog("Failed to refresh authorization token: %v", refreshErr)
					//return nil, refreshErr
				}
			}
		}
	}
	if Token != "" {
		req.Header.Set("Authorization", "Bearer "+Token)
		util.DebugLog("Setting Authorization header")
	}
	if runtimeCookie != "" {
		req.Header.Set("Cookie", runtimeCookie)
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
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			util.DebugLog("Failed to read error response body: %v", readErr)
		}
		err = formatHTTPError(resp, body)
		util.DebugLog("HTTP error details: %v", err)
		return nil, err
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		util.DebugLog("Failed to read response body: %v", err)
		return nil, err
	}
	util.DebugLog("Successfully got response, data length: %d", len(data))
	return
}

func formatHTTPError(resp *http.Response, body []byte) error {
	parts := []string{"http status code: " + strconv.Itoa(resp.StatusCode)}

	if v := headerValueIgnoreCase(resp.Header, "cf-mitigated"); v != "" {
		parts = append(parts, "cf-mitigated="+v)
	}
	if v := headerValueIgnoreCase(resp.Header, "server"); v != "" {
		parts = append(parts, "server="+v)
	}
	if v := headerValueIgnoreCase(resp.Header, "content-type"); v != "" {
		parts = append(parts, "content-type="+v)
	}
	if snippet := summarizeErrorBody(body, 120); snippet != "" {
		parts = append(parts, "body="+snippet)
	}

	return errors.New(strings.Join(parts, "; "))
}

func headerValueIgnoreCase(h http.Header, key string) string {
	for headerKey, values := range h {
		if strings.EqualFold(headerKey, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func summarizeErrorBody(body []byte, limit int) string {
	if len(body) == 0 {
		return ""
	}

	summary := strings.Join(strings.Fields(string(body)), " ")
	if summary == "" {
		return ""
	}
	if len(summary) > limit {
		summary = summary[:limit] + "..."
	}
	return fmt.Sprintf("%s", summary)
}

func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

// GetVideoUrl Get the mp4 source url of the video info
func GetVideoUrl(vi VideoInfo, host string) (string, string) {
	util.DebugLog("Starting to get video download URL, ID: %s", vi.Id)
	u := vi.FileUrl
	parsed, err := url.Parse(u)
	if err != nil {
		util.DebugLog("Failed to parse file URL: %v", err)
		return "", ""
	}
	expires := parsed.Query().Get("expires")
	xv := vi.File.Id + "_" + expires + "_mSvL05GfEmeEmsEYfGCnVpEjYgTJraJN"
	xversion := SHA1(xv)
	body, err := Fetch(u, xversion, host)
	if err != nil {
		util.DebugLog("Failed to get video URL: %v", err)
		return "", ""
	}
	var rList []ResolutionInfo
	err = json.Unmarshal(body, &rList)
	if err != nil {
		util.DebugLog("Failed to parse video URL: %v", err)
		return "", ""
	}
	for _, v := range rList {
		if v.Name == "Source" {
			util.DebugLog("Successfully got video download URL")
			return `https:` + v.Src.Download, v.Name
		}
	}
	if len(rList) > 0 {
		v := rList[0]
		return `https:` + v.Src.Download, v.Name
	}
	util.DebugLog("Source video URL not found")
	return "", ""
}

// GetUserProfile Get user profile by username
func GetUserProfile(username string, host string) (profile UserProfile, err error) {
	u := "https://api.iwara.tv/profile/" + username
	body, err := Fetch(u, "", host)
	if err != nil {
		util.DebugLog("Failed to get user profile: %v", err)
		return
	}
	err = json.Unmarshal(body, &profile)
	return
}

// GetVideoListByUser Get the video list of the user
func GetVideoListByUser(username string, host string) []VideoInfo {
	util.DebugLog("Starting to get user video list, username: %s", username)
	profile, err := GetUserProfile(username, host)
	if err != nil {
		util.DebugLog("Failed to get user info: %v", err)
		return nil
	}

	uid := profile.User.Id
	var list []VideoInfo
	retry := 3

	for i := 0; ; i++ {
		u := "https://api.iwara.tv/videos?rating=all&sort=date&page=" + strconv.Itoa(i) + "&user=" + uid
		body, err := Fetch(u, "", host)
		if err != nil {
			util.DebugLog("Failed to get page %d: %v", i+1, err)
			if retry > 0 {
				i--
				retry--
				continue
			} else {
				break
			}
		}
		var vList VideoList
		err = json.Unmarshal(body, &vList)
		if err != nil {
			util.DebugLog("Failed to parse page %d data: %v", i+1, err)
			if retry > 0 {
				i--
				retry--
				continue
			} else {
				break
			}
		}
		list = append(list, vList.Results...)
		if len(vList.Results) < vList.Limit {
			break
		}
		util.DebugLog("Successfully got page %d, current total videos: %d", i+1, len(list))
	}
	util.DebugLog("Completed getting user video list, total videos: %d", len(list))
	return list
}

// GetVideoList Get video list
// sort: "date", "trending", "popularity", "views", "likes"
// page: 0, 1, 2, 3, ...
// rating: "all", "general", "ecchi"
func GetVideoList(sort string, page int, rating string, host string) (list VideoList, err error) {
	u := "https://api.iwara.tv/videos?sort=" + sort + "&page=" + strconv.Itoa(page) + "&rating=" + rating
	data, err := Fetch(u, "", host)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &list)
	return
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
func GetAccessToken(auth string, host string) (string, error) {
	u := "https://api.iwara.tv/user/token"

	req, err := http.NewRequest("POST", u, nil)
	if err != nil {
		return "", err
	}

	req.Header = make(http.Header)
	for k, v := range SwitchHeaders(host) {
		req.Header[k] = append([]string(nil), v...)
	}
	req.Header.Set("content-type", "application/json")

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
func RefreshAuthToken(host string) (string, error) {
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

	req.Header = make(http.Header)
	for k, v := range SwitchHeaders(host) {
		req.Header[k] = append([]string(nil), v...)
	}
	req.Header.Set("content-type", "application/json")

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
