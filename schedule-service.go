package main

import (
	"context"
	"errors"
	"time"
)

type AvailabilityBlock struct {
	ID        int64 `json:"id"`
	StartHour int   `json:"startHour"`
	StartMin  int   `json:"startMin"`
	EndHour   int   `json:"endHour"`
	EndMin    int   `json:"endMin"`
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

type WeeklyAvailability struct {
	ID               int64                `json:"id"`
	StartDate        time.Time            `json:"startDate"`
	EndDate          time.Time            `json:"endDate"`
	TimeSlotDuration time.Duration        `json:"timeSlotDuration"`
	Sunday           []*AvailabilityBlock `json:"sunday"`
	Monday           []*AvailabilityBlock `json:"monday"`
	Tuesday          []*AvailabilityBlock `json:"tuesday"`
	Wednesday        []*AvailabilityBlock `json:"wednesday"`
	Thursday         []*AvailabilityBlock `json:"thursday"`
	Friday           []*AvailabilityBlock `json:"friday"`
	Saturday         []*AvailabilityBlock `json:"saturday"`
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
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"createdBy"`
	CreatedOn time.Time `json:"createdOn"`
	UpdatedOn time.Time `json:"updatedOn"`
	// TimeSlots []*TimeSlot `json:"timeSlots"`
}

func NewSchedule(name string, createdBy string) *Schedule {
	return &Schedule{
		Name:      name,
		CreatedBy: createdBy,
	}
}

// func (s *Schedule) AddTimeSlot(timeSlot *TimeSlot) {
// 	s.TimeSlots = append(s.TimeSlots, timeSlot)
// }

type ScheduleService interface {
	CreateSchedule(name string, createdBy string) (*Schedule, error)
	CreateDefaultWeeklyAvailability(scheduleId int64) *WeeklyAvailability
	GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error)
	GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error)
	GenerateTimeSlots(scheduleId int64, availability *WeeklyAvailability) []*TimeSlot
	FindAll() ([]*Schedule, error)
	FindById(id int64) (*Schedule, error)
	BookTimeSlot(scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) (bookingId int64, err error)
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

func (s *ScheduleServiceImpl) CreateSchedule(name string, createdBy string) (*Schedule, error) {
	schedule := NewSchedule(name, createdBy)

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	err := s.repository.Store(ctx, schedule)
	if err != nil {
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

func (s *ScheduleServiceImpl) GenerateTimeSlots(scheduleId int64, availability *WeeklyAvailability) []*TimeSlot {
	currentDay := availability.StartDate
	timeslots := []*TimeSlot{}

	for currentDay.Before(availability.EndDate) {
		dailyAvailability := availability.GetAvailabilityForDay(currentDay.Weekday())

		for _, availabilityBlock := range dailyAvailability {
			start := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), availabilityBlock.StartHour, availabilityBlock.StartMin, 0, 0, time.Local)
			end := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), availabilityBlock.EndHour, availabilityBlock.EndMin, 0, 0, time.Local)

			for start.Before(end) {
				timeslots = append(timeslots, NewTimeSlot(start, start.Add(availability.TimeSlotDuration)))
				start = start.Add(availability.TimeSlotDuration)
			}
		}
		currentDay = currentDay.AddDate(0, 0, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	s.repository.StoreTimeSlots(ctx, scheduleId, timeslots)
	return timeslots
}

func (s *ScheduleServiceImpl) CreateDefaultWeeklyAvailability(scheduleId int64) *WeeklyAvailability {
	sundayAvailability, _ := NewAvailabilityBlock(9, 0, 17, 0)
	mondayAvailability, _ := NewAvailabilityBlock(16, 0, 21, 0)
	tuesdayAvailability, _ := NewAvailabilityBlock(16, 0, 21, 0)
	wednesdayAvailability, _ := NewAvailabilityBlock(16, 0, 21, 0)
	thursdayAvailability, _ := NewAvailabilityBlock(16, 0, 21, 0)
	saturdayMorningAvailability, _ := NewAvailabilityBlock(9, 0, 12, 0)
	saturdayAfternoonAvailability, _ := NewAvailabilityBlock(13, 0, 17, 0)

	wa := &WeeklyAvailability{
		StartDate:        time.Now(),
		EndDate:          time.Now().AddDate(0, 0, 7),
		TimeSlotDuration: 15 * time.Minute,
		Sunday:           []*AvailabilityBlock{sundayAvailability},
		Monday:           []*AvailabilityBlock{mondayAvailability},
		Tuesday:          []*AvailabilityBlock{tuesdayAvailability},
		Wednesday:        []*AvailabilityBlock{wednesdayAvailability},
		Thursday:         []*AvailabilityBlock{thursdayAvailability},
		Friday:           []*AvailabilityBlock{},
		Saturday:         []*AvailabilityBlock{saturdayMorningAvailability, saturdayAfternoonAvailability},
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	s.repository.StoreWeeklyAvailability(ctx, scheduleId, wa)
	return wa
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

func (s *ScheduleServiceImpl) BookTimeSlot(scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) (bookingId int64, err error) {
	if !timeslot.IsAvailable() {
		return 0, errors.New("timeslot is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultRepositoryTimeout)
	defer cancel()

	return s.repository.BookTimeSlot(ctx, scheduleId, timeslot, bookerName, bookerEmail)
}
