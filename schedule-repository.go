package main

import (
	"database/sql"
	"time"
)

type ScheduleRepository interface {
	FindAll() ([]*Schedule, error)
	FindById(id int64) (*Schedule, error)
	Store(schedule *Schedule) (int64, error)
	Update(schedule *Schedule) error
	Delete(schedule *Schedule) error
	StoreTimeSlots(scheduleId int64, newTimeSlots []*TimeSlot) error
	GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error)
	GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error)
}

type SQLScheduleRepository struct {
	DB         *sql.DB
	dateLayout string
}

func NewSQLScheduleRepository(db *sql.DB) ScheduleRepository {
	return &SQLScheduleRepository{
		DB:         db,
		dateLayout: "2006-01-02 15:04:05-07:00",
	}
}

func (r *SQLScheduleRepository) FindAll() ([]*Schedule, error) {
	return nil, nil
}

func (r *SQLScheduleRepository) FindById(id int64) (*Schedule, error) {
	row := r.DB.QueryRow(`SELECT id, name, createdBy, createdOn, updatedOn FROM schedule WHERE id = ?`, id)

	var scheduleId int64
	var name string
	var createdBy string
	var createdOn string
	var updatedOn string

	err := row.Scan(&scheduleId, &name, &createdBy, &createdOn, &updatedOn)
	if err != nil {
		return nil, err
	}

	createdOnTime, err := time.Parse(r.dateLayout, createdOn)
	if err != nil {
		return nil, err
	}

	updatedOnTime, err := time.Parse(r.dateLayout, updatedOn)
	if err != nil {
		return nil, err
	}

	return &Schedule{
		ID:        scheduleId,
		Name:      name,
		CreatedBy: createdBy,
		CreatedOn: createdOnTime,
		UpdatedOn: updatedOnTime,
	}, nil

}

func (r *SQLScheduleRepository) Store(schedule *Schedule) (int64, error) {
	if schedule.ID != 0 {
		err := r.Update(schedule)
		return schedule.ID, err
	}

	// todo -- use a transaction here
	res, err := r.DB.Exec(`INSERT INTO schedule (name, createdBy, createdOn, updatedOn) VALUES (?, ?, ?, ?)`, schedule.Name, schedule.CreatedBy, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return 0, err
	}

	scheduleId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// for _, timeSlot := range schedule.TimeSlots {
	// 	_, err := r.DB.Exec(`INSERT INTO timeSlot (start, end, scheduleId, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeSlot.Start.UTC(), timeSlot.End.UTC(), scheduleId, time.Now().UTC(), time.Now().UTC())
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// }
	return scheduleId, nil
}

func (r *SQLScheduleRepository) Update(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) Delete(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) StoreTimeSlots(scheduleId int64, newTimeSlots []*TimeSlot) error {
	// todo -- use a transaction here
	for _, timeSlot := range newTimeSlots {
		_, err := r.DB.Exec(`INSERT INTO timeSlot (start, end, scheduleId, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeSlot.Start.UTC(), timeSlot.End.UTC(), scheduleId, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *SQLScheduleRepository) GetAllTimeSlots(scheduleId int64) ([]*TimeSlot, error) {
	rows, err := r.DB.Query(`SELECT id, start, end, createdOn, updatedOn FROM timeSlot WHERE scheduleId = ?`, scheduleId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.parseTimeSlotSqlRows(rows)
}

func (r *SQLScheduleRepository) GetTimeSlotsWithinRange(scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error) {
	rows, err := r.DB.Query(`SELECT id, start, end, createdOn, updatedOn FROM timeSlot WHERE scheduleId = ? AND start >= ? AND end <= ?`, scheduleId, start.UTC(), end.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.parseTimeSlotSqlRows(rows)
}

func (r *SQLScheduleRepository) parseTimeSlotSqlRows(rows *sql.Rows) ([]*TimeSlot, error) {
	var timeslots []*TimeSlot

	for rows.Next() {
		var id int64
		var start string
		var end string
		var createdOn string
		var updatedOn string

		err := rows.Scan(&id, &start, &end, &createdOn, &updatedOn)
		if err != nil {
			return nil, err
		}

		startTime, err := time.Parse(r.dateLayout, start)
		if err != nil {
			return nil, err
		}

		endTime, err := time.Parse(r.dateLayout, end)
		if err != nil {
			return nil, err
		}

		createdOnTime, err := time.Parse(r.dateLayout, createdOn)
		if err != nil {
			return nil, err
		}

		updatedOnTime, err := time.Parse(r.dateLayout, updatedOn)
		if err != nil {
			return nil, err
		}

		timeslots = append(timeslots, &TimeSlot{
			ID:        id,
			Start:     startTime,
			End:       endTime,
			CreatedOn: createdOnTime,
			UpdatedOn: updatedOnTime,
		})
	}

	return timeslots, nil
}
