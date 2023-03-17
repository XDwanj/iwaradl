package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"iwaradl/config"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Get the video info json from the API server
func GetVideoInfo(id string) (info VideoInfo, err error) {
	u := "https://api.iwara.tv/video/" + id
	body, err := Fetch(u, "")
	err = json.Unmarshal(body, &info)
	return
}

// Fetch the url and return the response body
func Fetch(u string, xversion string) (data []byte, err error) {
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

	req.Header.Set("User-Agent", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://www.iwara.tv")
	req.Header.Set("Referer", "https://www.iwara.tv/")
	if xversion != "" {
		req.Header.Set("X-Version", xversion)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			println(err.Error())
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		return nil, errors.New("HTTP status code error")
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return
}

func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

// Get the mp4 source url of the video info
func GetVideoUrl(vi VideoInfo) string {
	u := vi.FileUrl
	parsed, err := url.Parse(u)
	expires := parsed.Query().Get("expires")
	xv := vi.File.Id + "_" + expires + "_5nFp9kmbNnHdAFhaqMvt"
	xversion := SHA1(xv)
	body, err := Fetch(u, xversion)
	if err != nil {
		return ""
	}
	var rList []ResolutionInfo
	err = json.Unmarshal(body, &rList)
	if err != nil {
		return ""
	}
	for _, v := range rList {
		if v.Name == "Source" {
			return `https:` + v.Src.Download
		}
	}

	return ""
}

// Get user profile by username
func GetUserProfile(username string) (profile UserProfile, err error) {
	u := "https://api.iwara.tv/profile/" + username
	body, err := Fetch(u, "")
	err = json.Unmarshal(body, &profile)
	return
}

// Get the max page of the user's video list
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

// Get the video list of the user
func GetVideoList(username string) []VideoInfo {
	profile, err := GetUserProfile(username)
	if err != nil {
		return nil
	}
	uid := profile.User.Id
	maxPage := GetMaxPage(uid)
	var list []VideoInfo
	for i := 0; i < maxPage; i++ {
		u := "https://api.iwara.tv/videos?page=" + strconv.Itoa(i) + "&sort=date&user=" + uid
		body, err := Fetch(u, "")
		if err != nil {
			println(err.Error())
			continue
		}
		var vList VideoList
		err = json.Unmarshal(body, &vList)
		if err != nil {
			println(err.Error())
			continue
		}
		for _, v := range vList.Results {
			list = append(list, v)
		}
	}
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

// Get the detail information from video info
func GetDetailInfo(vi VideoInfo) (DetailInfo, error) {
	var di DetailInfo
	di.Author = vi.User.Name
	di.VideoName = vi.Title
	di.Description = vi.Body
	di.ReleaseDate = vi.CreatedAt.Format("2006-01-02 15:04:05")
	di.Year = di.ReleaseDate[:4]
	di.AddedDate = time.Now().Format("2006-01-02 15:04:05")
	var categories []string
	for _, v := range vi.Tags {
		categories = append(categories, v.Id)
	}
	di.Categories = categories
	return di, nil
}
