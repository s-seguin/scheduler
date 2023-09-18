package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// todo -- use context for all methods

type ScheduleRepository interface {
	FindAll(ctx context.Context) ([]*Schedule, error)
	FindById(ctx context.Context, id int64) (*Schedule, error)
	Store(ctx context.Context, schedule *Schedule) error
	Update(ctx context.Context, scheduleId *Schedule) error
	Delete(ctx context.Context, scheduleId *Schedule) error
	StoreWeeklyAvailability(ctx context.Context, scheduleId int64, weeklyAvailability *WeeklyAvailability) error
	StoreTimeSlots(ctx context.Context, scheduleId int64, newTimeSlots []*TimeSlot) error
	GetAllTimeSlots(ctx context.Context, scheduleId int64) ([]*TimeSlot, error)
	GetTimeSlotsWithinRange(ctx context.Context, scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error)
	BookTimeSlot(ctx context.Context, scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) (bookingId int64, err error)
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

func (r *SQLScheduleRepository) FindAll(ctx context.Context) ([]*Schedule, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, createdBy, createdOn, updatedOn FROM schedule`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*Schedule

	for rows.Next() {
		var scheduleId int64
		var name string
		var createdBy string
		var createdOn string
		var updatedOn string

		err := rows.Scan(&scheduleId, &name, &createdBy, &createdOn, &updatedOn)
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

		schedules = append(schedules, &Schedule{
			ID:        scheduleId,
			Name:      name,
			CreatedBy: createdBy,
			CreatedOn: createdOnTime,
			UpdatedOn: updatedOnTime,
		})
	}

	return schedules, nil
}

func (r *SQLScheduleRepository) FindById(ctx context.Context, id int64) (*Schedule, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, name, createdBy, createdOn, updatedOn FROM schedule WHERE id = ?`, id)

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
		err := r.Update(ctx, schedule)
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

func (r *SQLScheduleRepository) Update(ctx context.Context, schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) Delete(ctx context.Context, scheduleId *Schedule) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	_, err = r.DB.ExecContext(ctx, `DELETE FROM schedule WHERE id = ?`, scheduleId)
	if err != nil {
		return err
	}

	return tx.Commit()
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

func (r *SQLScheduleRepository) GetAllTimeSlots(ctx context.Context, scheduleId int64) ([]*TimeSlot, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT 
			t.id, 
			t.start, 
			t.end, 
			t.createdOn, 
			t.updatedOn,
			b.id,
			b.bookerName,
			b.bookerEmail,
			b.createdOn,
			b.updatedOn 
		FROM timeSlot AS t 
		LEFT OUTER JOIN booking AS b 
		WHERE t.scheduleId = ?`, scheduleId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.parseTimeSlotSqlRows(rows)
}

func (r *SQLScheduleRepository) GetTimeSlotsWithinRange(ctx context.Context, scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT 
			t.id, 
			t.start, 
			t.end, 
			t.createdOn, 
			t.updatedOn,
			b.id,
			b.bookerName,
			b.bookerEmail,
			b.createdOn,
			b.updatedOn 
		FROM timeSlot AS t 
		LEFT OUTER JOIN booking AS b 
		WHERE t.scheduleId = ? AND t.start >= ? AND t.end <= ?`, scheduleId, start.UTC(), end.UTC())
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

		var bookingId sql.NullInt64
		var bookerName sql.NullString
		var bookerEmail sql.NullString
		var bookingCreatedOn sql.NullString
		var bookingUpdatedOn sql.NullString

		err := rows.Scan(&id, &start, &end, &createdOn, &updatedOn, &bookingId, &bookerName, &bookerEmail, &bookingCreatedOn, &bookingUpdatedOn)
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

		if bookingId.Valid {
			bookingCreatedOnTime, err := time.Parse(r.dateLayout, bookingCreatedOn.String)
			if err != nil {
				return nil, err
			}

			bookingUpdatedOnTime, err := time.Parse(r.dateLayout, bookingUpdatedOn.String)
			if err != nil {
				return nil, err
			}

			timeslots = append(timeslots, &TimeSlot{
				ID:        id,
				Start:     startTime,
				End:       endTime,
				CreatedOn: createdOnTime,
				UpdatedOn: updatedOnTime,
				Booking: &Booking{
					ID:          bookingId.Int64,
					TimeSlot:    nil,
					BookerName:  bookerName.String,
					BookerEmail: bookerEmail.String,
					CreatedOn:   bookingCreatedOnTime,
					UpdatedOn:   bookingUpdatedOnTime,
				},
			})
			continue
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

func (r *SQLScheduleRepository) BookTimeSlot(ctx context.Context, scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) (bookingId int64, err error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	err = r.DB.QueryRowContext(ctx, `SELECT bookingId FROM timeSlot WHERE id = ? AND scheduleId = ? AND bookingId IS NOT NULL`, timeslot.ID, scheduleId).Scan(&bookingId)
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("timeslot already booked. bookingId: %d", bookingId)
	}

	bookingCreatedOn := time.Now().UTC()
	bookingUpdatedOn := time.Now().UTC()

	res, err := r.DB.ExecContext(ctx, `INSERT INTO booking (timeSlotId, bookerName, bookerEmail, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeslot.ID, bookerName, bookerEmail, bookingCreatedOn, bookingUpdatedOn)
	if err != nil {
		return 0, err
	}

	bookingId, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = r.DB.ExecContext(ctx, `UPDATE timeSlot SET bookingId = ? WHERE id = ?`, bookingId, timeslot.ID)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	timeslot.Booking = &Booking{
		ID:          bookingId,
		TimeSlot:    timeslot,
		BookerName:  bookerName,
		BookerEmail: bookerEmail,
		CreatedOn:   bookingCreatedOn,
		UpdatedOn:   bookingUpdatedOn,
	}

	return bookingId, nil
}
