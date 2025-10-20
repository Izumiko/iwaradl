package api

import (
	"encoding/xml"
	"time"
)

type UserInfo struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Status     string    `json:"status"`
	Role       string    `json:"role"`
	FollowedBy bool      `json:"followedBy"`
	Following  bool      `json:"following"`
	Friend     bool      `json:"friend"`
	Premium    bool      `json:"premium"`
	SeenAt     time.Time `json:"seenAt"`
	Avatar     FileInfo  `json:"avatar"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	DeletedAt  time.Time `json:"deletedAt"`
}

type FileInfo struct {
	Id            string    `json:"id"`
	Type          string    `json:"type"`
	Path          string    `json:"path"`
	Name          string    `json:"name"`
	Mime          string    `json:"mime"`
	Size          int       `json:"size"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	Duration      int       `json:"duration"`
	NumThumbnails int       `json:"numThumbnails"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Tag struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type VideoInfo struct {
	Id              string    `json:"id"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	Status          string    `json:"status"`
	Rating          string    `json:"rating"`
	Private         bool      `json:"private"`
	Unlisted        bool      `json:"unlisted"`
	Thumbnail       int       `json:"thumbnail"`
	EmbedUrl        string    `json:"embedUrl"`
	Liked           bool      `json:"liked"`
	NumLikes        int       `json:"numLikes"`
	NumViews        int       `json:"numViews"`
	NumComments     int       `json:"numComments"`
	File            FileInfo  `json:"file"`
	CustomThumbnail FileInfo  `json:"customThumbnail"`
	User            UserInfo  `json:"user"`
	Tags            []Tag     `json:"tags"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	DeletedAt       time.Time `json:"deletedAt"`
	FileUrl         string    `json:"fileUrl"`
}

type SrcInfo struct {
	View     string `json:"view"`
	Download string `json:"download"`
}

type ResolutionInfo struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Src       SrcInfo   `json:"src"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Type      string    `json:"type"`
}

type UserProfile struct {
	Body      string    `json:"body"`
	Header    FileInfo  `json:"header"`
	User      UserInfo  `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type VideoList struct {
	Count   int         `json:"count"`
	Limit   int         `json:"limit"`
	Page    int         `json:"page"`
	Results []VideoInfo `json:"results"`
}

type DetailInfo struct {
	XMLName     xml.Name `xml:"musicvideo"`
	Author      string   `xml:"director"`
	VideoName   string   `xml:"title"`
	Description string   `xml:"plot"`
	ReleaseDate string   `xml:"releasedate"`
	Premiered   string   `xml:"premiered"`
	Year        string   `xml:"year"`
	AddedDate   string   `xml:"dateadded"`
	Categories  []string `xml:"genre,omitempty"`
}

// JellyfinNfo a more compatible struct for Jellyfin nfo files
type JellyfinNfo struct {
	XMLName     xml.Name `xml:"musicvideo"`
	Title       string   `xml:"title"`
	Director    string   `xml:"director"`
	Year        string   `xml:"year"`
	Plot        string   `xml:"plot"`
	Runtime     int      `xml:"runtime,omitempty"`
	DateAdded   string   `xml:"dateadded,omitempty"`
	ReleaseDate string   `xml:"releasedate,omitempty"`
	Premiered   string   `xml:"premiered,omitempty"`
	Genre       []string `xml:"genre,omitempty"`
	LockData    bool     `xml:"lockdata,omitempty"`
	Art         struct {
		Poster string `xml:"poster,omitempty"`
	} `xml:"art,omitempty"`
	Fileinfo struct {
		Streamdetails struct {
			Video struct {
				Codec             string  `xml:"codec,omitempty"`
				Micodec           string  `xml:"micodec,omitempty"`
				Bitrate           int     `xml:"bitrate,omitempty"`
				Width             int     `xml:"width,omitempty"`
				Height            int     `xml:"height,omitempty"`
				Aspect            string  `xml:"aspect,omitempty"`
				Aspectratio       string  `xml:"aspectratio,omitempty"`
				Framerate         float32 `xml:"framerate,omitempty"`
				Language          string  `xml:"language,omitempty"`
				Scantype          string  `xml:"scantype,omitempty"`
				Default           string  `xml:"default,omitempty"`
				Forced            string  `xml:"forced,omitempty"`
				Duration          int     `xml:"duration,omitempty"`
				Durationinseconds int     `xml:"durationinseconds,omitempty"`
			} `xml:"video,omitempty"`
			Audio struct {
				Codec        string `xml:"codec,omitempty"`
				Micodec      string `xml:"micodec,omitempty"`
				Bitrate      int    `xml:"bitrate,omitempty"`
				Language     string `xml:"language,omitempty"`
				Scantype     string `xml:"scantype,omitempty"`
				Channels     int    `xml:"channels,omitempty"`
				Samplingrate int    `xml:"samplingrate,omitempty"`
				Default      string `xml:"default,omitempty"`
				Forced       string `xml:"forced,omitempty"`
			} `xml:"audio,omitempty"`
		} `xml:"streamdetails,omitempty"`
	} `xml:"fileinfo,omitempty"`
}
