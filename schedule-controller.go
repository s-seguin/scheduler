package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/unrolled/render"
)

type ScheduleController interface {
	MountRoutes() *chi.Mux
	UseMiddleware(middleware func(http.Handler) http.Handler)
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
	// c.router.Get("/", c.Index)
	// c.router.Get("/{scheduleId}/calendar", c.RenderCalendar)
	// c.router.Get("/{scheduleId}/timeslots", c.RenderTimeslots)

	c.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		schedules, err := c.scheduleService.FindAll()
		if err != nil {
			http.Error(w, "Error getting schedules", http.StatusInternalServerError)
			return
		}

		c.render.HTML(w, http.StatusOK, "schedules", &SchedulesViewModel{schedules})
	})

	c.router.Post("/", func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		createdBy := r.FormValue("createdBy")
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

		sundayStartTime := r.FormValue("sundayStartTime")
		sundayEndTime := r.FormValue("sundayEndTime")
		mondayStartTime := r.FormValue("mondayStartTime")
		mondayEndTime := r.FormValue("mondayEndTime")
		tuesdayStartTime := r.FormValue("tuesdayStartTime")
		tuesdayEndTime := r.FormValue("tuesdayEndTime")
		wednesdayStartTime := r.FormValue("wednesdayStartTime")
		wednesdayEndTime := r.FormValue("wednesdayEndTime")
		thursdayStartTime := r.FormValue("thursdayStartTime")
		thursdayEndTime := r.FormValue("thursdayEndTime")
		fridayStartTime := r.FormValue("fridayStartTime")
		fridayEndTime := r.FormValue("fridayEndTime")
		saturdayStartTime := r.FormValue("saturdayStartTime")
		saturdayEndTime := r.FormValue("saturdayEndTime")

		sundayAvailability, err := NewAvailabilityBlockFromStrings(sundayStartTime, sundayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Sunday availability: %s", err), http.StatusBadRequest)
			return
		}

		mondayAvailability, err := NewAvailabilityBlockFromStrings(mondayStartTime, mondayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Monday availability: %s", err), http.StatusBadRequest)
			return
		}

		tuesdayAvailability, err := NewAvailabilityBlockFromStrings(tuesdayStartTime, tuesdayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Tuesday availability: %s", err), http.StatusBadRequest)
			return
		}

		wednesdayAvailability, err := NewAvailabilityBlockFromStrings(wednesdayStartTime, wednesdayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Wednesday availability: %s", err), http.StatusBadRequest)
			return
		}

		fmt.Println("thursdayStartTime", thursdayStartTime, "thursdayEndTime", thursdayEndTime, thursdayEndTime == "")
		thursdayAvailability, err := NewAvailabilityBlockFromStrings(thursdayStartTime, thursdayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Thursday availability: %s", err), http.StatusBadRequest)
			return
		}

		fridayAvailability, err := NewAvailabilityBlockFromStrings(fridayStartTime, fridayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Friday availability: %s", err), http.StatusBadRequest)
			return
		}

		saturdayAvailability, err := NewAvailabilityBlockFromStrings(saturdayStartTime, saturdayEndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing Saturday availability: %s", err), http.StatusBadRequest)
			return
		}

		if sundayAvailability == nil && mondayAvailability == nil && tuesdayAvailability == nil && wednesdayAvailability == nil && thursdayAvailability == nil && fridayAvailability == nil && saturdayAvailability == nil {
			http.Error(w, "You must provide at least one day of availability", http.StatusBadRequest)
			return
		}

		weeklyAvailability.AddAvailabilityForDay(time.Sunday, sundayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Monday, mondayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Tuesday, tuesdayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Wednesday, wednesdayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Thursday, thursdayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Friday, fridayAvailability)
		weeklyAvailability.AddAvailabilityForDay(time.Saturday, saturdayAvailability)

		// fmt.Println("here")
		schedule, err := c.scheduleService.CreateSchedule(name, createdBy, start, end, &weeklyAvailability)
		// fmt.Println("ici")
		if err != nil {
			http.Error(w, "Error creating schedule", http.StatusInternalServerError)
			return
		}

		fmt.Printf("schedule %v\n", schedule)

		// fmt.Printf("sundayStartTime %s sundayEndTime %s\nscheduleId %d", sundayStartTime, sundayEndTime, schedule.ID)
		c.render.HTML(w, http.StatusOK, "schedule-created-partial", nil)
	})

	// c.router.Post("/create-schedule", func(w http.ResponseWriter, r *http.Request) {
	// 	name := r.FormValue("Schedule Name")
	// 	owner := r.FormValue("")
	// })

	c.router.Get("/{scheduleId}", func(w http.ResponseWriter, r *http.Request) {
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

		for currentDay.Before(startOfNextMonth) {
			days = append(days, NullableTime{Time: currentDay, IsThisMonth: currentDay.Month() == date.Month(), HasAvailableAppointment: dayHasTimeSlot(currentDay, schedule.TimeSlots)})
			currentDay = currentDay.AddDate(0, 0, 1)
		}

		c.render.HTML(w, http.StatusOK, "schedule", &ScheduleViewModel{Schedule: schedule, Date: date, NextMonth: startOfNextMonth, PreviousMonth: startOfMonth.AddDate(0, 0, -1), Days: days})
	})

	c.router.Get("/{scheduleId}/timeslots/{timeslotId}/booking-form", func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, "Timeslot is not available", http.StatusNotFound)
			return
		}

		c.render.HTML(w, http.StatusOK, "booking-form", &TimeSlotBookingViewModel{TimeSlot: timeslot, Schedule: schedule})
	})

	c.router.Post("/{scheduleId}/timeslots/{timeslotId}/book", func(w http.ResponseWriter, r *http.Request) {
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

		bookingId, err := c.scheduleService.BookTimeSlot(scheduleId, timeslotId, r.FormValue("name"), r.FormValue("email"))
		if err != nil {
			http.Error(w, "Error booking timeslot", http.StatusInternalServerError)
			return
		}

		c.render.Text(w, http.StatusOK, fmt.Sprintf("Booking successful! Your booking ID is %d <button hx-get=\"/schedules/%d\" hx-target=\"#scheduleContainer\">Return to schedule</button>", bookingId, scheduleId))
	})

	return c.router
}

func (c *ScheduleControllerImpl) UseMiddleware(middleware func(http.Handler) http.Handler) {
	c.router.Use(middleware)
}
func dayHasTimeSlot(day time.Time, timeslots []*TimeSlot) bool {
	for _, timeslot := range timeslots {
		if timeslot.Start.Year() != day.Year() && timeslot.End.Year() != day.Year() {
			continue
		}

		if timeslot.Start.Month() != day.Month() && timeslot.End.Month() != day.Month() {
			continue
		}

		if timeslot.Start.Day() == day.Day() || timeslot.End.Day() == day.Day() {
			return true
		}
	}
	return false
}

type SchedulesViewModel struct {
	Schedules []*Schedule
}

type TimeSlotBookingViewModel struct {
	TimeSlot *TimeSlot
	Schedule *Schedule
}

type ScheduleViewModel struct {
	Schedule      *Schedule
	Date          time.Time
	NextMonth     time.Time
	PreviousMonth time.Time
	Days          []NullableTime
}
type CalendarViewModel struct {
	ScheduleId    int64
	Days          []NullableTime
	CurrentMonth  string
	NextMonth     string
	PreviousMonth string
}

type NullableTime struct {
	Time                    time.Time
	HasAvailableAppointment bool
	IsThisMonth             bool
}

func (c *ScheduleControllerImpl) Index(w http.ResponseWriter, r *http.Request) {

	schedules, err := c.scheduleService.FindAll()
	if err != nil {
		http.Error(w, "Error getting schedules", http.StatusInternalServerError)
		return
	}

	c.render.HTML(w, http.StatusOK, "index", schedules)
}

// todo -- we can render the calendar and the timeslots in the same view, the timeslots would be for the current day
func (c *ScheduleControllerImpl) RenderCalendar(w http.ResponseWriter, r *http.Request) {
	scheduleId, err := strconv.ParseInt(chi.URLParam(r, "scheduleId"), 10, 64)
	if err != nil {
		http.Error(w, "Schedule ID was not a valid int64", http.StatusBadRequest)
		return
	}

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

	timeslots, err := c.scheduleService.GetTimeSlotsWithinRange(scheduleId, now, end.AddDate(0, 0, -1))
	if err != nil {
		http.Error(w, "Error getting ts", http.StatusInternalServerError)
		return
	}

	for start.Before(end) {
		m := start.Month()
		// todo -- this is horrendous, need to fix it
		hasAvailableAppointment := false
		for _, t := range timeslots {
			fmt.Printf("timeslot %s available? %v\n", t.Start, t.IsAvailable())
			// 	t := *timeslot
			// 	// log.Printf("t.Start: %v timeslot.End: %v start: %v iA: %b sAs: %b sEs: %b eBe: %b", timeslot.Start, timeslot.End, start, timeslot.IsAvailable(), timeslot.Start.After(start), timeslot.End.Before(start.AddDate(0, 0, 1)), timeslot.End.Before(end))
			// 	log.Print(t, &timeslot, t.IsAvailable(), t.Booking)
			// 	// log.Print(ia, timeslot.Booking, timeslot.Booking == nil)
			if t.IsAvailable() && (t.Start.After(start) || t.Start.Equal(start)) && t.End.Before(start.AddDate(0, 0, 1)) {
				hasAvailableAppointment = true
			}
		}

		days = append(days, NullableTime{Time: start, IsThisMonth: m == now.Month(), HasAvailableAppointment: hasAvailableAppointment})
		start = start.AddDate(0, 0, 1)
	}

	calendarViewModel := CalendarViewModel{
		ScheduleId:    scheduleId,
		Days:          days,
		CurrentMonth:  now.Format("Jan 2006"),
		NextMonth:     now.AddDate(0, 1, 0).Format("2006-01"),
		PreviousMonth: now.AddDate(0, -1, 0).Format("2006-01"),
	}
	c.render.HTML(w, http.StatusOK, "calendar", calendarViewModel)
}

func (c *ScheduleControllerImpl) RenderTimeslots(w http.ResponseWriter, r *http.Request) {
	scheduleId, err := strconv.ParseInt(chi.URLParam(r, "scheduleId"), 10, 64)
	if err != nil {
		http.Error(w, "Schedule ID was not a valid int64", http.StatusBadRequest)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	start, err := time.Parse("2006-01-02 15:04:05 -0700 MST", date)
	if err != nil {
		http.Error(w, "Error parsing date", http.StatusInternalServerError)
		return
	}

	end := start.AddDate(0, 0, 1)

	fmt.Printf("start: %v end: %v\n", start, end)

	timeslots, err := c.scheduleService.GetTimeSlotsWithinRange(scheduleId, start, end)
	if err != nil {
		http.Error(w, "Error getting ts", http.StatusInternalServerError)
		return
	}

	c.render.HTML(w, http.StatusOK, "timeslots", timeslots)
}
