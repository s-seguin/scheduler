package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseTime(time string) (int, int, error) {
	parts := strings.Split(time, ":")

	if len(parts) != 2 {
		return 0, 0, errors.New("time must be in the format HH:MM")
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return hour, min, nil
}

// todo -- should this belong to the Service?
func NewAvailabilityBlockFromStrings(start string, end string) (*AvailabilityBlock, error) {
	if start == "" && end == "" {
		return nil, nil
	}

	startHour, startMin, err := parseTime(start)
	if err != nil {
		return nil, err
	}

	endHour, endMin, err := parseTime(end)
	if err != nil {
		return nil, err
	}

	if startHour > endHour {
		return nil, errors.New("startHour must be less than endHour")
	}

	return NewAvailabilityBlock(startHour, startMin, endHour, endMin)
}

func NewAvailabilityBlock(startHour int, startMin int, endHour int, endMin int) (*AvailabilityBlock, error) {
	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
		return nil, errors.New("startHour must be between 0 and 23")
	}

	if startMin < 0 || startMin > 59 || endMin < 0 || endMin > 59 {
		return nil, errors.New("startMin must be between 0 and 59")
	}

	if startHour > endHour {
		return nil, errors.New("startHour must be less than endHour")
	}

	if startHour == endHour && startMin > endMin {
		return nil, errors.New("startMin must be less than endMin")
	}

	if startMin%5 != 0 || endMin%5 != 0 {
		return nil, errors.New("startMin and endMin must be a 5min increment")
	}

	// s := fmt.Sprintf("%02d:%02d", startHour, startMin)
	// e := fmt.Sprintf("%02d:%02d", endHour, endMin)

	return &AvailabilityBlock{
		StartHour: startHour,
		StartMin:  startMin,
		EndHour:   endHour,
		EndMin:    endMin,
	}, nil
}

type AvailabilityBlock struct {
	ID        int64 `json:"id"`
	StartHour int   `json:"startHour"`
	StartMin  int   `json:"startMin"`
	EndHour   int   `json:"endHour"`
	EndMin    int   `json:"endMin"`
}
type WeeklyAvailability struct {
	ID        int64                `json:"id"`
	Sunday    []*AvailabilityBlock `json:"sunday"`
	Monday    []*AvailabilityBlock `json:"monday"`
	Tuesday   []*AvailabilityBlock `json:"tuesday"`
	Wednesday []*AvailabilityBlock `json:"wednesday"`
	Thursday  []*AvailabilityBlock `json:"thursday"`
	Friday    []*AvailabilityBlock `json:"friday"`
	Saturday  []*AvailabilityBlock `json:"saturday"`
}

func (a *WeeklyAvailability) GetAvailabilityForDay(day time.Weekday) []*AvailabilityBlock {
	switch day {
	case time.Sunday:
		return a.Sunday
	case time.Monday:
		return a.Monday
	case time.Tuesday:
		return a.Tuesday
	case time.Wednesday:
		return a.Wednesday
	case time.Thursday:
		return a.Thursday
	case time.Friday:
		return a.Friday
	case time.Saturday:
		return a.Saturday
	default:
		return []*AvailabilityBlock{}
	}
}

func (a *WeeklyAvailability) AddAvailabilityForDay(day time.Weekday, availabilityBlock *AvailabilityBlock) {
	if availabilityBlock == nil {
		return
	}

	switch day {
	case time.Sunday:
		a.Sunday = append(a.Sunday, availabilityBlock)
	case time.Monday:
		a.Monday = append(a.Monday, availabilityBlock)
	case time.Tuesday:
		a.Tuesday = append(a.Tuesday, availabilityBlock)
	case time.Wednesday:
		a.Wednesday = append(a.Wednesday, availabilityBlock)
	case time.Thursday:
		a.Thursday = append(a.Thursday, availabilityBlock)
	case time.Friday:
		a.Friday = append(a.Friday, availabilityBlock)
	case time.Saturday:
		a.Saturday = append(a.Saturday, availabilityBlock)
	}
}

type TimeSlot struct {
	ID        int64     `json:"id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Booking   *Booking  `json:"booking"`
	CreatedOn time.Time `json:"createdOn"`
	UpdatedOn time.Time `json:"updatedOn"`
}

func NewTimeSlot(start time.Time, end time.Time) *TimeSlot {
	return &TimeSlot{
		Start: start,
		End:   end,
	}
}

func (t *TimeSlot) IsAvailable() bool {
	return t.Booking == nil
}

func (t *TimeSlot) Book(bookerName string, bookerEmail string) {
	t.Booking = NewBooking(t, bookerName, bookerEmail)
}

// todo -- can probably remove the reference to TimeSlot here
type Booking struct {
	ID          int64     `json:"id"`
	TimeSlot    *TimeSlot `json:"timeSlot"`
	BookerName  string    `json:"bookerName"`
	BookerEmail string    `json:"bookerEmail"`
	CreatedOn   time.Time `json:"createdOn"`
	UpdatedOn   time.Time `json:"updatedOn"`
}

func NewBooking(timeSlot *TimeSlot, bookerName string, bookerEmail string) *Booking {
	return &Booking{
		TimeSlot:    timeSlot,
		BookerName:  bookerName,
		BookerEmail: bookerEmail,
	}
}

// later this can hold the public url etc...
type Schedule struct {
	ID                 int64               `json:"id"`
	Name               string              `json:"name"`
	Timezone           string              `json:"timezone"`
	Start              time.Time           `json:"start"`
	End                time.Time           `json:"end"`
	TimeSlotDuration   time.Duration       `json:"timeSlotDuration"`
	WeeklyAvailability *WeeklyAvailability `json:"weeklyAvailability"`
	CreatedBy          string              `json:"createdBy"`
	CreatedOn          time.Time           `json:"createdOn"`
	UpdatedOn          time.Time           `json:"updatedOn"`
	TimeSlots          []*TimeSlot         `json:"timeSlots"`
}

func NewSchedule(name string, createdBy string, start time.Time, end time.Time, weeklyAvailability *WeeklyAvailability) *Schedule {
	return &Schedule{
		Name:               name,
		CreatedBy:          createdBy,
		Start:              start,
		End:                end,
		TimeSlotDuration:   15 * time.Minute,
		Timezone:           time.Local.String(),
		WeeklyAvailability: weeklyAvailability,
	}
}

func (s *Schedule) OverrideTimeSlotDuration(duration time.Duration) {
	s.TimeSlotDuration = duration
}

func (s *Schedule) AddTimeSlot(timeSlot *TimeSlot) {
	s.TimeSlots = append(s.TimeSlots, timeSlot)
}

func (s *Schedule) AddWeeklyAvailability(weeklyAvailability *WeeklyAvailability) {
	s.WeeklyAvailability = weeklyAvailability
}

func (s *Schedule) GenerateTimeSlots() {
	fmt.Println("Generating timeslots")
	for current := s.Start; current.Before(s.End); current = current.AddDate(0, 0, 1) {
		fmt.Println("Generating timeslots for", current.Format("Monday, 2006-01-02"))
		availability := s.WeeklyAvailability.GetAvailabilityForDay(current.Weekday())
		if len(availability) == 0 {
			fmt.Println("No availability for", current.Format("Monday"))
			continue
		}

		for _, a := range availability {
			fmt.Println("Availability: ", a.StartHour, a.StartMin, a.EndHour, a.EndMin)
			start := time.Date(current.Year(), current.Month(), current.Day(), a.StartHour, a.StartMin, 0, 0, time.Local)
			end := time.Date(current.Year(), current.Month(), current.Day(), a.EndHour, a.EndMin, 0, 0, time.Local)

			for start.Before(end) {
				timeSlot := NewTimeSlot(start, start.Add(s.TimeSlotDuration))
				s.AddTimeSlot(timeSlot)
				start = start.Add(s.TimeSlotDuration)
			}
		}
	}
}

type ScheduleService interface {
	CreateSchedule(name string, createdBy string, start time.Time, end time.Time, weeklyAvailability *WeeklyAvailability) (*Schedule, error)
	GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error)
	GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error)
	FindAll() ([]*Schedule, error)
	FindById(id int64) (*Schedule, error)
	BookTimeSlot(scheduleId int64, timeslotId int64, bookerName string, bookerEmail string) (bookingId int64, err error)
}

type ScheduleServiceImpl struct {
	repository               ScheduleRepository
	defaultRepositoryTimeout time.Duration
}

func NewScheduleService(repository ScheduleRepository) ScheduleService {
	return &ScheduleServiceImpl{
		repository:               repository,
		defaultRepositoryTimeout: 15 * time.Second,
	}
}

func (s *ScheduleServiceImpl) CreateSchedule(name string, createdBy string, start time.Time, end time.Time, weeklyAvailability *WeeklyAvailability) (*Schedule, error) {
	fmt.Println("Creating schedule")
	schedule := NewSchedule(name, createdBy, start, end, weeklyAvailability)
	schedule.GenerateTimeSlots()

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	fmt.Println("Storing schedule")
	err := s.repository.Store(ctx, schedule)
	if err != nil {
		fmt.Printf("Error storing schedule: %s", err)
		return nil, err
	}

	return schedule, nil
}

func (s *ScheduleServiceImpl) FindAll() ([]*Schedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.FindAll(ctx)
}

func (s *ScheduleServiceImpl) FindById(id int64) (*Schedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.FindById(ctx, id)
}

func (s *ScheduleServiceImpl) GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.GetAllTimeSlots(ctx, scheduleId)
}

func (s *ScheduleServiceImpl) GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.GetTimeSlotsWithinRange(ctx, scheduleId, start, end)
}

// todo -- should this accept and ID instead?
// todo -- should this return a booking?
func (s *ScheduleServiceImpl) BookTimeSlot(scheduleId int64, timeslotId int64, bookerName string, bookerEmail string) (int64, error) {
	schedule, err := s.FindById(scheduleId)
	if err != nil {
		return 0, err
	}

	var timeslot *TimeSlot
	for _, ts := range schedule.TimeSlots {
		if ts.ID == timeslotId {
			timeslot = ts
			break
		}
	}

	if timeslot == nil {
		return 0, errors.New("timeslot not found")
	}

	if !timeslot.IsAvailable() {
		return 0, errors.New("timeslot is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.BookTimeSlot(ctx, scheduleId, timeslot, bookerName, bookerEmail)
}
