package server

import (
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/tasks", createTask)
		r.Get("/tasks", listTasks)
		r.Get("/tasks/{vid}", getTask)
		r.Delete("/tasks/{vid}", deleteTask)
	})
	return r
}

func RunServer(bindAddr string, port int) error {
	StartWorker()
	addr := net.JoinHostPort(bindAddr, strconv.Itoa(port))
	return http.ListenAndServe(addr, NewRouter())
}
