package server

import (
	"errors"
	"iwaradl/config"
	"iwaradl/downloader"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"
)

const fallbackTemplate = "{{title}}-{{video_id}}"

type TaskOptions struct {
	ProxyURL         string `json:"proxy_url,omitempty"`
	DownloadDir      string `json:"download_dir,omitempty"`
	Cookie           string `json:"cookie,omitempty"`
	MaxRetry         int    `json:"max_retry,omitempty"`
	FilenameTemplate string `json:"filename_template,omitempty"`
}

type TaskOptionsSummary struct {
	ProxyURL         string `json:"proxy_url,omitempty"`
	DownloadDir      string `json:"download_dir"`
	CookieSet        bool   `json:"cookie_set"`
	MaxRetry         int    `json:"max_retry"`
	FilenameTemplate string `json:"filename_template"`
}

type Task struct {
	VID            string
	Status         string // pending / running / completed / failed
	Progress       float32
	CreatedAt      time.Time
	Options        TaskOptions
	OptionsSummary TaskOptionsSummary
}

var (
	store      = make(map[string]*Task)
	mu         sync.RWMutex
	workerOnce sync.Once
	workerWake = make(chan struct{}, 1)
)

type DeleteResult int

const (
	DeleteOK DeleteResult = iota
	DeleteNotFound
	DeleteNotPending
)

func StartWorker() {
	workerOnce.Do(func() {
		go workerLoop()
	})
}

func CreateTask(urls []string, reqOpts TaskOptions) ([]*Task, error) {
	opts, err := resolveTaskOptions(reqOpts)
	if err != nil {
		return nil, err
	}

	vids := downloader.ProcessUrlList(urls)

	mu.Lock()
	var list []*Task
	for _, vid := range vids {
		if store[vid] != nil {
			continue
		}
		t := &Task{
			VID:            vid,
			Status:         "pending",
			Progress:       0,
			CreatedAt:      time.Now(),
			Options:        opts,
			OptionsSummary: summarizeOptions(opts),
		}
		store[t.VID] = t
		list = append(list, cloneTask(t))
	}
	mu.Unlock()

	if len(list) > 0 {
		wakeWorker()
	}

	return list, nil
}

func GetTask(vid string) (*Task, bool) {
	mu.RLock()
	defer mu.RUnlock()
	t, ok := store[vid]
	if !ok {
		return nil, false
	}
	return cloneTask(t), true
}

func ListTasks() []*Task {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]*Task, 0, len(store))
	for _, t := range store {
		list = append(list, cloneTask(t))
	}
	return list
}

func DeleteTask(vid string) DeleteResult {
	mu.Lock()
	defer mu.Unlock()
	t, ok := store[vid]
	if !ok {
		return DeleteNotFound
	}
	if t.Status != "pending" {
		return DeleteNotPending
	}
	delete(store, vid)
	return DeleteOK
}

func wakeWorker() {
	select {
	case workerWake <- struct{}{}:
	default:
	}
}

func workerLoop() {
	for {
		<-workerWake
		for {
			task := pickPendingTask()
			if task == nil {
				break
			}
			downloadTask(task)
		}
	}
}

func pickPendingTask() *Task {
	mu.Lock()
	defer mu.Unlock()
	for _, t := range store {
		if t.Status != "pending" {
			continue
		}
		t.Status = "running"
		t.Progress = 0
		return cloneTask(t)
	}
	return nil
}

func downloadTask(task *Task) {
	if task == nil {
		return
	}

	downloader.VidList = []string{task.VID}
	downloader.SetProgressHook(updateTaskProgress)
	defer downloader.SetProgressHook(nil)

	retry := task.Options.MaxRetry
	if retry <= 0 {
		retry = 1
	}

	failed := len(downloader.VidList)
	dlOpts := downloader.DownloadOptions{
		RootDir:          task.Options.DownloadDir,
		UseSubDir:        false,
		UseSubDirSet:     true,
		ProxyURL:         task.Options.ProxyURL,
		Cookie:           task.Options.Cookie,
		FilenameTemplate: task.Options.FilenameTemplate,
	}
	for i := 0; i < retry && failed > 0; i++ {
		failed = downloader.ConcurrentDownloadWithOptions(dlOpts)
		if failed > 0 && i < retry-1 {
			time.Sleep(30 * time.Second)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	t, ok := store[task.VID]
	if !ok {
		return
	}
	if t.Status == "running" {
		if downloader.FindHistory(task.VID) {
			t.Status = "completed"
			t.Progress = 1
		} else {
			t.Status = "failed"
			t.Progress = 0
		}
	}
	t.Options.Cookie = ""
	t.OptionsSummary.CookieSet = false
}

func updateTaskProgress(report downloader.ProgressReport) {
	if report.VID == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	t, ok := store[report.VID]
	if !ok {
		return
	}

	if report.Done {
		if report.Success {
			t.Status = "completed"
			t.Progress = 1
		} else {
			t.Status = "failed"
		}
		return
	}

	if t.Status == "pending" {
		t.Status = "running"
	}
	if report.BytesTotal > 0 {
		p := float32(report.BytesComplete) / float32(report.BytesTotal)
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		t.Progress = p
	}
}

func resolveTaskOptions(req TaskOptions) (TaskOptions, error) {
	opts := TaskOptions{
		ProxyURL:         strings.TrimSpace(config.Cfg.ProxyUrl),
		DownloadDir:      "",
		MaxRetry:         config.Cfg.MaxRetry,
		FilenameTemplate: strings.TrimSpace(config.Cfg.FilenameTemplate),
	}
	if opts.MaxRetry <= 0 {
		opts.MaxRetry = 1
	}
	if opts.FilenameTemplate == "" {
		opts.FilenameTemplate = fallbackTemplate
	}

	if v := strings.TrimSpace(req.ProxyURL); v != "" {
		if err := validateProxyURL(v); err != nil {
			return TaskOptions{}, err
		}
		opts.ProxyURL = v
	}
	if v := strings.TrimSpace(req.DownloadDir); v != "" {
		converted := downloader.ConvertExternalTemplate(v)
		if _, err := variableTemplateParser().Parse(converted); err != nil {
			return TaskOptions{}, errors.New("invalid download_dir template")
		}
		opts.DownloadDir = v
	}
	if req.MaxRetry > 0 {
		opts.MaxRetry = req.MaxRetry
	}
	if v := strings.TrimSpace(req.Cookie); v != "" {
		opts.Cookie = v
	}
	if v := strings.TrimSpace(req.FilenameTemplate); v != "" {
		converted := downloader.ConvertExternalTemplate(v)
		if _, err := variableTemplateParser().Parse(converted); err != nil {
			return TaskOptions{}, errors.New("invalid filename_template")
		}
		opts.FilenameTemplate = v
	}

	if opts.DownloadDir == "" {
		opts.DownloadDir = config.Cfg.RootDir
	}
	return opts, nil
}

func validateProxyURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid proxy_url")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "socks5" {
		return errors.New("proxy_url scheme must be http, https or socks5")
	}
	return nil
}

func summarizeOptions(opts TaskOptions) TaskOptionsSummary {
	return TaskOptionsSummary{
		ProxyURL:         opts.ProxyURL,
		DownloadDir:      opts.DownloadDir,
		CookieSet:        opts.Cookie != "",
		MaxRetry:         opts.MaxRetry,
		FilenameTemplate: opts.FilenameTemplate,
	}
}

func variableTemplateParser() *template.Template {
	return template.New("filename").Funcs(template.FuncMap{
		"now":             func(layout ...string) string { return "" },
		"publish_time":    func(layout ...string) string { return "" },
		"title":           func() string { return "" },
		"video_id":        func() string { return "" },
		"author":          func() string { return "" },
		"author_nickname": func() string { return "" },
		"quality":         func() string { return "" },
	})
}

func cloneTask(t *Task) *Task {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}
