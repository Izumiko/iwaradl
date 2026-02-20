package server

import (
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
	store = make(map[string]*Task)
	mu    sync.RWMutex
)

func CreateTask(urls []string) []*Task {
	mu.Lock()
	defer mu.Unlock()
	vids := downloader.ProcessUrlList(urls)

	var list []*Task
	for _, vid := range vids {
		t := &Task{
			VID:       vid,
			Status:    "pending",
			Progress:  0,
			CreatedAt: time.Now(),
		}
		if store[t.VID] != nil {
			continue
		}
		store[t.VID] = t
		list = append(list, t)
	}

	// 异步执行，不阻塞接口
	//go func() {
	//	t.Status = "running"
	//	for i, u := range t.URLs {
	//		_ = downloader.Download(u) // 你已有的函数
	//		t.Progress = int(float64(i+1) / float64(len(t.URLs)) * 100)
	//	}
	//	t.Status = "completed"
	//}()
	return list
}

func GetTask(vid string) (*Task, bool) {
	mu.RLock()
	defer mu.RUnlock()
	t, ok := store[vid]
	return t, ok
}

func ListTasks() []*Task {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]*Task, 0, len(store))
	for _, t := range store {
		list = append(list, t)
	}
	return list
}

func DeleteTask(vid string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := store[vid]; !ok {
		return false
	}
	delete(store, vid)
	return true
}
