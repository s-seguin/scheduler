package main

import (
	"log"
	"net/http"
	"strings"

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

func (s *Server) ServeStatic(path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		s.Router.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	s.Router.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func (s *Server) Start() {
	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", s.Router)
}
