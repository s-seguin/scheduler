package main

import (
	"time"
)

type TimeSlot struct {
	ID        string    `json:"id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	BookingID string    `json:"bookingId"`
}

func (t *TimeSlot) IsAvailable() bool {
	return t.BookingID == ""
}

type Booking struct {
	ID          string `json:"id"`
	TimeSlotID  string `json:"timeSlotId"`
	BookerName  string `json:"bookerName"`
	BookerEmail string `json:"bookerEmail"`
}

type Schedule struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	CreatedBy string      `json:"createdBy"`
	CreatedOn time.Time   `json:"createdOn"`
	UpdatedOn time.Time   `json:"updatedOn"`
	TimeSlots []*TimeSlot `json:"timeSlots"`
}

func NewSchedule(name string, createdBy string) *Schedule {
	return &Schedule{
		Name:      name,
		CreatedBy: createdBy,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
		TimeSlots: []*TimeSlot{},
	}
}

func (s *Schedule) AddTimeSlot(timeSlot *TimeSlot) {
	s.TimeSlots = append(s.TimeSlots, timeSlot)
}

type ScheduleService interface {
	CreateSchedule(name string, createdBy string) (*Schedule, error)
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

	err := s.repository.Store(schedule)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}
