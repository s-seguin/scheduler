package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Server struct {
	Router *chi.Mux
}

func NewServer() *Server {
	router := chi.NewRouter()

	server := &Server{
		Router: router,
	}

	return server
}

func (s *Server) SetupMiddleware() {
	s.Router.Use(middleware.Logger)
}

func (s *Server) MountRouter(path string, router *chi.Mux) {
	s.Router.Mount(path, router)
}

// func (s *Server) MountRoutes() {
// 	s.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
// 		s.Render.HTML(w, http.StatusOK, "index", nil)
// 	})
// }

func (s *Server) Start() {
	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", s.Router)
}
