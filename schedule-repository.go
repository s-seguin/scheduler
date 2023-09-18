package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ScheduleRepository interface {
	FindAll() ([]*Schedule, error)
	FindById(id int64) (*Schedule, error)
	Store(ctx context.Context, schedule *Schedule) error
	Update(schedule *Schedule) error
	Delete(schedule *Schedule) error
	StoreWeeklyAvailability(ctx context.Context, scheduleId int64, weeklyAvailability *WeeklyAvailability) error
	StoreTimeSlots(ctx context.Context, scheduleId int64, newTimeSlots []*TimeSlot) error
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

func (r *SQLScheduleRepository) Store(ctx context.Context, schedule *Schedule) error {
	if schedule.ID != 0 {
		err := r.Update(schedule)
		return err
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	res, err := r.DB.ExecContext(ctx, `INSERT INTO schedule (name, createdBy, createdOn, updatedOn) VALUES (?, ?, ?, ?)`, schedule.Name, schedule.CreatedBy, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return err
	}

	schedule.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *SQLScheduleRepository) Update(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) Delete(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) StoreTimeSlots(ctx context.Context, scheduleId int64, newTimeSlots []*TimeSlot) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	for _, timeSlot := range newTimeSlots {
		_, err := r.DB.ExecContext(ctx, `INSERT INTO timeSlot (start, end, scheduleId, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeSlot.Start.UTC(), timeSlot.End.UTC(), scheduleId, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
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

func (r *SQLScheduleRepository) StoreWeeklyAvailability(ctx context.Context, scheduleId int64, weeklyAvailability *WeeklyAvailability) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	res, err := r.DB.ExecContext(ctx, `INSERT INTO availability (scheduleId, startDate, endDate, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, scheduleId, weeklyAvailability.StartDate.UTC(), weeklyAvailability.EndDate.UTC(), time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return err
	}

	weeklyAvailability.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}

	currentDay := weeklyAvailability.StartDate
	for currentDay.Before(weeklyAvailability.EndDate) {
		dailyAvailability := weeklyAvailability.GetAvailabilityForDay(currentDay.Weekday())
		for _, availabilityBlock := range dailyAvailability {
			startTime := fmt.Sprintf("%02d:%02d", availabilityBlock.StartHour, availabilityBlock.StartMin)
			endTime := fmt.Sprintf("%02d:%02d", availabilityBlock.EndHour, availabilityBlock.EndMin)

			_, err := r.DB.ExecContext(ctx, `INSERT INTO availabilityBlock (availabilityId, day, startTime, endTime, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?, ?)`, weeklyAvailability.ID, currentDay.Weekday(), startTime, endTime, time.Now().UTC(), time.Now().UTC())
			if err != nil {
				return err
			}
		}

		currentDay = currentDay.AddDate(0, 0, 1)
	}

	return tx.Commit()
}
