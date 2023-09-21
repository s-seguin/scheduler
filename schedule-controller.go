package main

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	renderFuncMap := template.FuncMap{
		"mod": func(i int, x int) int {
			return i % x
		}}
	funcs := []template.FuncMap{renderFuncMap}
	render := render.New(render.Options{Extensions: []string{".html"}, Directory: "views", Funcs: funcs})

	return &ScheduleControllerImpl{
		scheduleService: scheduleService,
		render:          render,
		router:          chi.NewRouter(),
	}
}

func (c *ScheduleControllerImpl) MountRoutes() *chi.Mux {
	c.router.Get("/", c.Index)
	c.router.Get("/calendar", c.GetCalendar)

	return c.router
}

type CalendarViewModel struct {
	Days          []NullableTime
	CurrentMonth  string
	NextMonth     string
	PreviousMonth string
}

type NullableTime struct {
	Time    time.Time
	IsValid bool
}

func (c *ScheduleControllerImpl) Index(w http.ResponseWriter, r *http.Request) {

	c.render.HTML(w, http.StatusOK, "index", nil)
}

func (c *ScheduleControllerImpl) GetCalendar(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01")
	}

	datePieces := strings.Split(date, "-")
	y, err := strconv.Atoi(datePieces[0])
	if err != nil {
		y = time.Now().Year()
	}
	m, err := strconv.Atoi(datePieces[1])
	if err != nil {
		m = int(time.Now().Month())
	}

	var days []NullableTime
	now := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.Local)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	if start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -int(start.Weekday()))
	}

	for start.Before(end) {
		m := start.Month()
		days = append(days, NullableTime{Time: start, IsValid: m == now.Month()})
		start = start.AddDate(0, 0, 1)
	}

	calendarViewModel := CalendarViewModel{
		Days:          days,
		CurrentMonth:  now.Format("Jan 2006"),
		NextMonth:     now.AddDate(0, 1, 0).Format("2006-01"),
		PreviousMonth: now.AddDate(0, -1, 0).Format("2006-01"),
	}
	c.render.HTML(w, http.StatusOK, "calendar", calendarViewModel)
}
