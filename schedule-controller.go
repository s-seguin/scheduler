package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"
	"github.com/unrolled/render"
)

type ScheduleController interface {
	MountRoutes() *chi.Mux
}

type ScheduleControllerImpl struct {
	scheduleService ScheduleService
	render          *render.Render
	router          *chi.Mux
	cookieStore     *sessions.CookieStore
}

func NewScheduleController(scheduleService ScheduleService, cookieStore *sessions.CookieStore) ScheduleController {
	renderFuncMap := template.FuncMap{
		"mod": func(i int, x int) int {
			return i % x
		}}
	funcs := []template.FuncMap{renderFuncMap}
	render := render.New(render.Options{Extensions: []string{".html"}, Layout: "layout", Directory: "views", Funcs: funcs})

	return &ScheduleControllerImpl{
		scheduleService: scheduleService,
		render:          render,
		router:          chi.NewRouter(),
		cookieStore:     cookieStore,
	}
}

func (c *ScheduleControllerImpl) MountRoutes() *chi.Mux {
	c.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		profile, err := getAuth0Profile(r)

		if err != nil || profile.IsExpired() {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/schedules", http.StatusFound)
	})

	c.router.Route("/schedules", func(r chi.Router) {
		r.Use(c.isAuthenticated)

		r.Get("/", c.getSchedules)
		r.Get("/new", c.getScheduleForm)
		r.Post("/", c.createSchedule)
		r.Get("/{scheduleId}", c.getScheduleById)
		r.Get("/{scheduleId}/timeslots/{timeslotId}/book", c.getBookingForm)
		r.Post("/{scheduleId}/timeslots/{timeslotId}/book", c.bookTimeslot)
	})

	return c.router
}

type SchedulesViewModel struct {
	Schedules []*Schedule
	User      Auth0Profile
}

type TimeSlotBookingViewModel struct {
	TimeSlot *TimeSlot
	Schedule *Schedule
	User     Auth0Profile
}

type ScheduleViewModel struct {
	Schedule      *Schedule
	Date          time.Time
	NextMonth     time.Time
	PreviousMonth time.Time
	Days          []NullableTime
	User          Auth0Profile
}

type NullableTime struct {
	Time                    time.Time
	HasAvailableAppointment bool
	IsThisMonth             bool
}

func (c *ScheduleControllerImpl) getSchedules(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r) // since this is a protected route, we can assume the profile is there
	createdBy := auth0Profile.Sub

	schedules, err := c.scheduleService.FindAll(createdBy)
	if err != nil {
		http.Error(w, "Error getting schedules", http.StatusInternalServerError)
		return
	}

	c.render.HTML(w, http.StatusOK, "schedules/list", &SchedulesViewModel{schedules, auth0Profile}, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) getScheduleForm(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r)

	c.render.HTML(w, http.StatusOK, "schedules/new", &SchedulesViewModel{Schedules: nil, User: auth0Profile}, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) createSchedule(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r) // since this is a protected route, we can assume the profile is there
	createdBy := auth0Profile.Sub

	name := r.FormValue("name")
	startDate := r.FormValue("startDate")
	endDate := r.FormValue("endDate")

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		http.Error(w, "Bad start date", http.StatusBadRequest)
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		http.Error(w, "Bad end date", http.StatusBadRequest)
		return
	}

	weeklyAvailability := WeeklyAvailability{
		Sunday:    []*AvailabilityBlock{},
		Monday:    []*AvailabilityBlock{},
		Tuesday:   []*AvailabilityBlock{},
		Wednesday: []*AvailabilityBlock{},
		Thursday:  []*AvailabilityBlock{},
		Friday:    []*AvailabilityBlock{},
		Saturday:  []*AvailabilityBlock{},
	}

	daysOfWeek := []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	hasAtLeastOneAvailability := false
	for _, day := range daysOfWeek {
		dayStr := strings.ToLower(day.String())
		startTimeKey := fmt.Sprintf("%sStartTime", dayStr)
		endTimeKey := fmt.Sprintf("%sEndTime", dayStr)

		startTime := r.FormValue(startTimeKey)
		endTime := r.FormValue(endTimeKey)

		availabilityBlock, err := NewAvailabilityBlockFromStrings(startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing %s availability: %s", day, err), http.StatusBadRequest)
			return
		}

		if availabilityBlock != nil {
			hasAtLeastOneAvailability = true
			weeklyAvailability.AddAvailabilityForDay(day, availabilityBlock)
		}
	}

	if !hasAtLeastOneAvailability {
		http.Error(w, "You must provide at least one day of availability", http.StatusBadRequest)
		return
	}

	limitToOneBookingPerUser := r.FormValue("limitToOneBookingPerUser") == "on"

	schedule, err := c.scheduleService.CreateSchedule(name, createdBy, start, end, &weeklyAvailability, limitToOneBookingPerUser)
	if err != nil {
		http.Error(w, "Error creating schedule", http.StatusInternalServerError)
		return
	}

	fmt.Printf("schedule %v\n", schedule)

	c.render.HTML(w, http.StatusOK, "schedules/created-partial", nil, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) getScheduleById(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r) // since this is a protected route, we can assume the profile is there

	scheduleId, err := strconv.ParseInt(chi.URLParam(r, "scheduleId"), 10, 64)
	if err != nil {
		http.Error(w, "Schedule ID was not a valid int64", http.StatusBadRequest)
		return
	}

	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02 15:04:05 -0700 MST", dateStr)
	if err != nil {
		log.Println("error parsing date", err)
		date = time.Now()
	}

	schedule, err := c.scheduleService.FindById(scheduleId)
	// todo -- make custom error
	if err == sql.ErrNoRows {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Error getting schedule", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	days := []NullableTime{}
	startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	numDaysTillFirstSunday := int(time.Sunday - startOfMonth.Weekday())
	currentDay := startOfMonth.AddDate(0, 0, numDaysTillFirstSunday)

	// todo -- this should probably be improved so that we don't have to loop through all days and we only pass the timeslots for the provided date to the view
	// todo -- could split it up so that the schedule and timeslots are loaded on separate routes
	for currentDay.Before(startOfNextMonth) {
		days = append(days, NullableTime{Time: currentDay, IsThisMonth: currentDay.Month() == date.Month(), HasAvailableAppointment: schedule.DayHasTimeSlot(currentDay)})
		currentDay = currentDay.AddDate(0, 0, 1)
	}

	scheduleData := &ScheduleViewModel{Schedule: schedule, Date: date, NextMonth: startOfNextMonth, PreviousMonth: startOfMonth.AddDate(0, 0, -1), Days: days, User: auth0Profile}

	c.render.HTML(w, http.StatusOK, "schedules/view", scheduleData, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) getBookingForm(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r) // protected route so can ignore error

	scheduleId, err := strconv.ParseInt(chi.URLParam(r, "scheduleId"), 10, 64)
	if err != nil {
		http.Error(w, "Schedule ID was not a valid int64", http.StatusBadRequest)
		return
	}

	timeslotId, err := strconv.ParseInt(chi.URLParam(r, "timeslotId"), 10, 64)
	if err != nil {
		http.Error(w, "Timeslot ID was not a valid int64", http.StatusBadRequest)
		return
	}

	schedule, err := c.scheduleService.FindById(scheduleId)
	if err != nil {
		http.Error(w, "Error getting schedule", http.StatusInternalServerError)
		return
	}
	// todo -- implement this instead of the for loop
	// timeslot, err := c.scheduleService.GetTimeSlotById(timeslotId)

	var timeslot *TimeSlot
	for _, t := range schedule.TimeSlots {
		if t.ID == timeslotId {
			timeslot = t
			break
		}
	}

	if timeslot == nil {
		http.Error(w, "Timeslot not found", http.StatusNotFound)
		return
	}

	// todo -- return HTML instead
	if !timeslot.IsAvailable() {
		http.Error(w, "Timeslot is not available", http.StatusBadRequest)
		return
	}

	timeslotBookingData := &TimeSlotBookingViewModel{TimeSlot: timeslot, Schedule: schedule, User: auth0Profile}

	c.render.HTML(w, http.StatusOK, "schedules/booking-form", timeslotBookingData, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) bookTimeslot(w http.ResponseWriter, r *http.Request) {
	auth0Profile, _ := getAuth0Profile(r)

	scheduleId, err := strconv.ParseInt(chi.URLParam(r, "scheduleId"), 10, 64)
	if err != nil {
		http.Error(w, "Schedule ID was not a valid int64", http.StatusBadRequest)
		return
	}

	timeslotId, err := strconv.ParseInt(chi.URLParam(r, "timeslotId"), 10, 64)
	if err != nil {
		http.Error(w, "Timeslot ID was not a valid int64", http.StatusBadRequest)
		return
	}

	timeslot, err := c.scheduleService.BookTimeSlot(scheduleId, timeslotId, r.FormValue("name"), r.FormValue("email"))
	if err != nil {
		if errors.Is(err, ErrBookingLimitReached) {
			http.Error(w, "Booking limit reached for provide booking email address", http.StatusBadRequest)
		} else {
			http.Error(w, "Error booking timeslot", http.StatusInternalServerError)
		}
		return
	}

	c.render.HTML(w, http.StatusOK, "schedules/booked-partial", &TimeSlotBookingViewModel{TimeSlot: timeslot, Schedule: &Schedule{ID: scheduleId}, User: auth0Profile}, overrideLayoutIfHtmx(r))
}

func (c *ScheduleControllerImpl) isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		profile, err := getAuth0Profile(r)
		if err != nil || profile.IsExpired() {
			// http.Error(w, "unauthorized", http.StatusUnauthorized)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getAuth0Profile(r *http.Request) (Auth0Profile, error) {
	profile, ok := r.Context().Value(RequestContextKey("profile")).(Auth0Profile)
	if !ok {
		return profile, fmt.Errorf("profile not found in context")
	}
	return profile, nil
}

func overrideLayoutIfHtmx(r *http.Request) render.HTMLOptions {
	if isHtmxRequest(r) {
		return render.HTMLOptions{Layout: "layout-htmx-partial"}
	}

	return render.HTMLOptions{}
}

func isHtmxRequest(r *http.Request) bool {
	hxReqHeader := r.Header["Hx-Request"]
	return len(hxReqHeader) == 1 && strings.ToLower(hxReqHeader[0]) == "true"
}
