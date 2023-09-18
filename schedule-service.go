package main

import (
	"errors"
	"fmt"
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

type Booking struct {
	ID          int64     `json:"id"`
	TimeSlot    *TimeSlot `json:"timeSlot"`
	BookerName  string    `json:"bookerName"`
	BookerEmail string    `json:"bookerEmail"`
}

func NewBooking(timeSlot *TimeSlot, bookerName string, bookerEmail string) *Booking {
	return &Booking{
		TimeSlot:    timeSlot,
		BookerName:  bookerName,
		BookerEmail: bookerEmail,
	}
}

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
}

type ScheduleServiceImpl struct {
	repository ScheduleRepository
}

func NewScheduleService(repository ScheduleRepository) ScheduleService {
	return &ScheduleServiceImpl{
		repository: repository,
	}
}

func (s *ScheduleServiceImpl) CreateSchedule(name string, createdBy string) (*Schedule, error) {
	schedule := NewSchedule(name, createdBy)

	id, err := s.repository.Store(schedule)
	if err != nil {
		return nil, err
	}

	schedule.ID = id

	return schedule, nil
}

func (s *ScheduleServiceImpl) FindAll() ([]*Schedule, error) {
	return s.repository.FindAll()
}

func (s *ScheduleServiceImpl) FindById(id int64) (*Schedule, error) {
	return s.repository.FindById(id)
}

func (s *ScheduleServiceImpl) GenerateTimeSlots(scheduleId int64, availability *WeeklyAvailability) []*TimeSlot {
	currentDay := availability.StartDate
	timeslots := []*TimeSlot{}

	for currentDay.Before(availability.EndDate) {
		fmt.Println(currentDay)
		currentDay = currentDay.AddDate(0, 0, 1)
		dailyAvailability := availability.GetAvailabilityForDay(currentDay.Weekday())

		for _, availabilityBlock := range dailyAvailability {
			start := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), availabilityBlock.StartHour, availabilityBlock.StartMin, 0, 0, time.Local)
			end := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), availabilityBlock.EndHour, availabilityBlock.EndMin, 0, 0, time.Local)

			for start.Before(end) {
				timeslots = append(timeslots, NewTimeSlot(start, start.Add(availability.TimeSlotDuration)))
				start = start.Add(availability.TimeSlotDuration)
			}
		}
	}

	s.repository.StoreTimeSlots(scheduleId, timeslots)
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

	return &WeeklyAvailability{
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
}

func (s *ScheduleServiceImpl) GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error) {
	return s.repository.GetAllTimeSlots(scheduleId)
}

func (s *ScheduleServiceImpl) GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error) {
	return s.repository.GetTimeSlotsWithinRange(scheduleId, start, end)
}
