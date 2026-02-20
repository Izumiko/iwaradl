package server

import (
	"iwaradl/config"
	"iwaradl/downloader"
	"sync"
	"time"
)

type Task struct {
	VID       string
	Status    string // pending / running / completed / failed
	Progress  float32
	CreatedAt time.Time
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

func CreateTask(urls []string) []*Task {
	vids := downloader.ProcessUrlList(urls)

	mu.Lock()
	var list []*Task
	for _, vid := range vids {
		if store[vid] != nil {
			continue
		}
		t := &Task{
			VID:       vid,
			Status:    "pending",
			Progress:  0,
			CreatedAt: time.Now(),
		}
		store[t.VID] = t
		list = append(list, cloneTask(t))
	}
	mu.Unlock()

	if len(list) > 0 {
		wakeWorker()
	}

	return list
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
			vids := pickPendingTasks()
			if len(vids) == 0 {
				break
			}
			downloadBatch(vids)
		}
	}
}

func pickPendingTasks() []string {
	mu.Lock()
	defer mu.Unlock()

	vids := make([]string, 0)
	for vid, t := range store {
		if t.Status != "pending" {
			continue
		}
		t.Status = "running"
		t.Progress = 0
		vids = append(vids, vid)
	}
	return vids
}

func downloadBatch(vids []string) {
	downloader.VidList = append([]string(nil), vids...)

	maxRetry := config.Cfg.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 1
	}

	failed := len(downloader.VidList)
	for i := 0; i < maxRetry && failed > 0; i++ {
		failed = downloader.ConcurrentDownload()
		if failed > 0 && i < maxRetry-1 {
			time.Sleep(30 * time.Second)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	for _, vid := range vids {
		t, ok := store[vid]
		if !ok || t.Status != "running" {
			continue
		}
		if downloader.FindHistory(vid) {
			t.Status = "completed"
			t.Progress = 1
			continue
		}
		t.Status = "failed"
		t.Progress = 0
	}
}

func cloneTask(t *Task) *Task {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}
