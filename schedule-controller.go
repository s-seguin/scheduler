package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/unrolled/render"
)

type ScheduleController interface {
	MountRoutes() *chi.Mux
	Index(w http.ResponseWriter, r *http.Request)
}

type ScheduleControllerImpl struct {
	scheduleService ScheduleService
	render          *render.Render
	router          *chi.Mux
}

func NewScheduleController(scheduleService ScheduleService) ScheduleController {
	render := render.New(render.Options{Extensions: []string{".html"}, Directory: "views"})

	return &ScheduleControllerImpl{
		scheduleService: scheduleService,
		render:          render,
		router:          chi.NewRouter(),
	}
}

func (c *ScheduleControllerImpl) MountRoutes() *chi.Mux {
	c.router.Get("/", c.Index)

	return c.router
}

func (c *ScheduleControllerImpl) Index(w http.ResponseWriter, r *http.Request) {
	// c.render.HTML(w, http.StatusOK, "index", nil)
}
