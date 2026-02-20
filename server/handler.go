package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type CreateReq struct {
	URLs []string `json:"urls"`
}

type TaskResp struct {
	VID       string    `json:"vid"`
	Status    string    `json:"status"`
	Progress  float32   `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
}

// POST /api/tasks
func createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.URLs) == 0 {
		http.Error(w, "urls empty", http.StatusUnprocessableEntity)
		return
	}
	tl := CreateTask(req.URLs)
	if len(tl) == 0 {
		http.Error(w, "no new valid tasks", http.StatusUnprocessableEntity)
		return
	}
	list := make([]TaskResp, len(tl))
	for i, t := range tl {
		list[i] = taskToResp(t)
	}
	respondJSON(w, http.StatusCreated, list)
}

// GET /api/tasks/{vid}
func getTask(w http.ResponseWriter, r *http.Request) {
	t, ok := GetTask(chi.URLParam(r, "vid"))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, taskToResp(t))
}

// GET /api/tasks
func listTasks(w http.ResponseWriter, r *http.Request) {
	tasks := ListTasks()
	list := make([]TaskResp, len(tasks))
	for i, t := range tasks {
		list[i] = taskToResp(t)
	}
	respondJSON(w, http.StatusOK, list)
}

// DELETE /api/tasks/{vid}
func deleteTask(w http.ResponseWriter, r *http.Request) {
	switch DeleteTask(chi.URLParam(r, "vid")) {
	case DeleteOK:
		w.WriteHeader(http.StatusNoContent)
	case DeleteNotFound:
		http.Error(w, "not found", http.StatusNotFound)
	case DeleteNotPending:
		http.Error(w, "only pending task can be deleted", http.StatusConflict)
	}
}

/* ---------- 工具 ---------- */
func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func taskToResp(t *Task) TaskResp {
	return TaskResp{
		VID:       t.VID,
		Status:    t.Status,
		Progress:  t.Progress,
		CreatedAt: t.CreatedAt,
	}
}
