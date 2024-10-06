package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sqids/sqids-go"
)

// todo -- use context for all methods

type ScheduleRepository interface {
	FindAll(ctx context.Context, createdBy string) ([]*Schedule, error)
	FindById(ctx context.Context, id int64) (*Schedule, error)
	Store(ctx context.Context, schedule *Schedule) error
	Update(ctx context.Context, scheduleId *Schedule) error
	Delete(ctx context.Context, scheduleId *Schedule) error
	GetAllTimeSlots(ctx context.Context, scheduleId int64) ([]*TimeSlot, error)
	GetTimeSlotsWithinRange(ctx context.Context, scheduleId int64, start time.Time, end time.Time) ([]*TimeSlot, error)
	BookTimeSlot(ctx context.Context, scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) error
	CancelBooking(ctx context.Context, booking *Booking) error
	GetBookings(ctx context.Context, scheduleId int64) ([]*Booking, error)
	GetBookingById(ctx context.Context, bookingId int64) (*Booking, error)
	GetBookingsForUser(ctx context.Context, scheduleId int64, bookerEmail string) ([]*Booking, error)
}

type SQLScheduleRepository struct {
	DB         *sql.DB
	dateLayout string
	sqid       sqids.Sqids
}

func NewSQLScheduleRepository(db *sql.DB, sqid *sqids.Sqids) ScheduleRepository {
	return &SQLScheduleRepository{
		DB:         db,
		dateLayout: "2006-01-02 15:04:05-07:00",
		sqid:       *sqid,
	}
}

func dailyAvailabilityToString(da []*AvailabilityBlock) string {
	var blocks []string
	for _, block := range da {
		blocks = append(blocks, fmt.Sprintf("%02d:%02d-%02d:%02d", block.StartHour, block.StartMin, block.EndHour, block.EndMin))
	}

	return strings.Join(blocks, ",")
}

// string should be in format "HH:MM-HH:MM,HH:MM-HH:MM"
func parseDailyAvailability(availability string) ([]*AvailabilityBlock, error) {
	if availability == "" {
		return nil, nil
	}

	var availabilityBlocks []*AvailabilityBlock
	blocks := strings.Split(availability, ",")

	for _, block := range blocks {
		times := strings.Split(block, "-")
		if len(times) != 2 {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}

		startTimes := strings.Split(times[0], ":")
		if len(startTimes) != 2 {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}

		endTimes := strings.Split(times[1], ":")
		if len(endTimes) != 2 {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}

		startHour, err := strconv.Atoi(startTimes[0])
		if err != nil {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}
		startMin, err := strconv.Atoi(startTimes[1])
		if err != nil {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}
		endHour, err := strconv.Atoi(endTimes[0])
		if err != nil {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}
		endMin, err := strconv.Atoi(endTimes[1])
		if err != nil {
			return nil, fmt.Errorf("invalid availability block: %s", block)
		}

		availabilityBlocks = append(availabilityBlocks, &AvailabilityBlock{
			StartHour: startHour,
			StartMin:  startMin,
			EndHour:   endHour,
			EndMin:    endMin,
		})
	}

	return availabilityBlocks, nil
}

func (r *SQLScheduleRepository) FindAll(ctx context.Context, createdBy string) ([]*Schedule, error) {
	// todo -- this is missing timeslots and bookings
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, createdBy, createdOn, updatedOn, limitToOneBookingPerUser, isShared FROM schedule WHERE createdBy = ?`, createdBy)
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
		var limitToOneBookingPerUser bool
		var isShared bool

		err := rows.Scan(&scheduleId, &name, &createdBy, &createdOn, &updatedOn, &limitToOneBookingPerUser, &isShared)
		if err != nil {
			return nil, err
		}

		createdOnTime, err := time.Parse(r.dateLayout, createdOn)
		if err != nil {
			return nil, fmt.Errorf("error parsing createdOn: %w", err)
		}

		updatedOnTime, err := time.Parse(r.dateLayout, updatedOn)
		if err != nil {
			return nil, fmt.Errorf("error parsing updatedOn: %w", err)
		}

		scheduleSqid, err := r.sqid.Encode([]uint64{uint64(scheduleId)})
		if err != nil {
			return nil, err
		}

		schedules = append(schedules, &Schedule{
			ID:                       scheduleId,
			Sqid:                     scheduleSqid,
			Name:                     name,
			CreatedBy:                createdBy,
			CreatedOn:                createdOnTime,
			UpdatedOn:                updatedOnTime,
			LimitToOneBookingPerUser: limitToOneBookingPerUser,
			IsShared:                 isShared,
		})
	}

	return schedules, nil
}

func (r *SQLScheduleRepository) FindById(ctx context.Context, id int64) (*Schedule, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			s.id, 
			s.name, 
			s.timezone, 
			s.timeSlotDurationMin,
			s.limitToOneBookingPerUser,
			s.isShared,
			s.sundayAvailability,
			s.mondayAvailability,
			s.tuesdayAvailability,
			s.wednesdayAvailability,
			s.thursdayAvailability,
			s.fridayAvailability,
			s.saturdayAvailability,
			s.createdBy, 
			s.createdOn, 
			s.updatedOn,
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
		FROM schedule AS s 
		LEFT OUTER JOIN timeSlot AS t ON s.id = t.scheduleId
		LEFT OUTER JOIN booking AS b ON t.id = b.timeSlotId 
		WHERE s.id = ?
		ORDER BY t.start ASC	
	`, id)
	if err != nil {
		return nil, err
	}

	var scheduleId int64
	var name string
	var timezone string
	var timeSlotDurationMin int64
	var limitToOneBookingPerUser bool
	var isShared bool
	var sundayAvailability string
	var mondayAvailability string
	var tuesdayAvailability string
	var wednesdayAvailability string
	var thursdayAvailability string
	var fridayAvailability string
	var saturdayAvailability string

	var createdBy string
	var createdOn string
	var updatedOn string
	var createdOnTime time.Time
	var updatedOnTime time.Time

	var timeslots []*TimeSlot

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	for rows.Next() {
		var timeSlotId sql.NullInt64
		var start sql.NullString
		var end sql.NullString
		var tsCreatedOn sql.NullString
		var tsUpdatedOn sql.NullString

		var bookingId sql.NullInt64
		var bookerName sql.NullString
		var bookerEmail sql.NullString
		var bookingCreatedOn sql.NullString
		var bookingUpdatedOn sql.NullString

		err := rows.Scan(
			&scheduleId,
			&name,
			&timezone,
			&timeSlotDurationMin,
			&limitToOneBookingPerUser,
			&isShared,
			&sundayAvailability,
			&mondayAvailability,
			&tuesdayAvailability,
			&wednesdayAvailability,
			&thursdayAvailability,
			&fridayAvailability,
			&saturdayAvailability,
			&createdBy,
			&createdOn,
			&updatedOn,
			&timeSlotId,
			&start,
			&end,
			&tsCreatedOn,
			&tsUpdatedOn,
			&bookingId,
			&bookerName,
			&bookerEmail,
			&bookingCreatedOn,
			&bookingUpdatedOn,
		)
		if err != nil {
			return nil, err
		}

		createdOnTime, err = time.Parse(r.dateLayout, createdOn)
		if err != nil {
			return nil, fmt.Errorf("error parsing createdOn: %w", err)
		}

		updatedOnTime, err = time.Parse(r.dateLayout, updatedOn)
		if err != nil {
			return nil, fmt.Errorf("error parsing updatedOn: %w", err)
		}

		if !timeSlotId.Valid {
			continue
		}

		startTime, err := time.Parse(r.dateLayout, start.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing start: %w", err)
		}

		endTime, err := time.Parse(r.dateLayout, end.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing end: %w", err)
		}

		tsCreatedOnTime, err := time.Parse(r.dateLayout, tsCreatedOn.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing tsCreatedOn: %w", err)
		}

		tsUpdatedOnTime, err := time.Parse(r.dateLayout, tsUpdatedOn.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing tsUpdatedOn: %w", err)
		}

		timeslotSqid, err := r.sqid.Encode([]uint64{uint64(timeSlotId.Int64)})
		if err != nil {
			return nil, fmt.Errorf("error encoding timeslot sqid: %w", err)
		}

		if !bookingId.Valid {
			timeslots = append(timeslots, &TimeSlot{
				ID:        timeSlotId.Int64,
				Sqid:      timeslotSqid,
				Start:     startTime,
				End:       endTime,
				CreatedOn: tsCreatedOnTime,
				UpdatedOn: tsUpdatedOnTime,
			})
			continue
		}

		bookingCreatedOnTime, err := time.Parse(r.dateLayout, bookingCreatedOn.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing bookingCreatedOn: %w", err)
		}

		bookingUpdatedOnTime, err := time.Parse(r.dateLayout, bookingUpdatedOn.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing bookingUpdatedOn: %w", err)
		}

		bookingSqid, err := r.sqid.Encode([]uint64{uint64(bookingId.Int64)})
		if err != nil {
			return nil, fmt.Errorf("error encoding timeslot sqid: %w", err)
		}

		timeslots = append(timeslots, &TimeSlot{
			ID:        timeSlotId.Int64,
			Sqid:      timeslotSqid,
			Start:     startTime,
			End:       endTime,
			CreatedOn: tsCreatedOnTime,
			UpdatedOn: tsUpdatedOnTime,
			Booking: &Booking{
				ID:          bookingId.Int64,
				Sqid:        bookingSqid,
				TimeSlot:    nil,
				BookerName:  bookerName.String,
				BookerEmail: bookerEmail.String,
				CreatedOn:   bookingCreatedOnTime,
				UpdatedOn:   bookingUpdatedOnTime,
			},
		})
		continue
	}

	sundayAvailabilityBlocks, err := parseDailyAvailability(sundayAvailability)
	if err != nil {
		return nil, err
	}
	mondayAvailabilityBlocks, err := parseDailyAvailability(mondayAvailability)
	if err != nil {
		return nil, err
	}
	tuesdayAvailabilityBlocks, err := parseDailyAvailability(tuesdayAvailability)
	if err != nil {
		return nil, err
	}
	wednesdayAvailabilityBlocks, err := parseDailyAvailability(wednesdayAvailability)
	if err != nil {
		return nil, err
	}
	thursdayAvailabilityBlocks, err := parseDailyAvailability(thursdayAvailability)
	if err != nil {
		return nil, err
	}
	fridayAvailabilityBlocks, err := parseDailyAvailability(fridayAvailability)
	if err != nil {
		return nil, err
	}
	saturdayAvailabilityBlocks, err := parseDailyAvailability(saturdayAvailability)
	if err != nil {
		return nil, err
	}

	scheduleSqid, err := r.sqid.Encode([]uint64{uint64(scheduleId)})
	if err != nil {
		return nil, err
	}

	return &Schedule{
		ID:                       scheduleId,
		Sqid:                     scheduleSqid,
		Name:                     name,
		CreatedBy:                createdBy,
		CreatedOn:                createdOnTime,
		UpdatedOn:                updatedOnTime,
		LimitToOneBookingPerUser: limitToOneBookingPerUser,
		IsShared:                 isShared,
		TimeSlots:                timeslots,
		WeeklyAvailability: &WeeklyAvailability{
			Sunday:    sundayAvailabilityBlocks,
			Monday:    mondayAvailabilityBlocks,
			Tuesday:   tuesdayAvailabilityBlocks,
			Wednesday: wednesdayAvailabilityBlocks,
			Thursday:  thursdayAvailabilityBlocks,
			Friday:    fridayAvailabilityBlocks,
			Saturday:  saturdayAvailabilityBlocks,
		},
	}, nil
}

func (r *SQLScheduleRepository) Store(ctx context.Context, schedule *Schedule) error {
	if schedule.ID != 0 {
		err := r.Update(ctx, schedule)
		return err
	}

	sundayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Sunday)
	mondayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Monday)
	tuesdayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Tuesday)
	wednesdayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Wednesday)
	thursdayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Thursday)
	fridayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Friday)
	saturdayAvailability := dailyAvailabilityToString(schedule.WeeklyAvailability.Saturday)

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO schedule (
					name, 
					start,
					end,
					timezone, 
					timeslotDurationMin,
					limitToOneBookingPerUser,
					isShared,
					sundayAvailability, 
					mondayAvailability, 
					tuesdayAvailability, 
					wednesdayAvailability, 
					thursdayAvailability, 
					fridayAvailability, 
					saturdayAvailability, 
					createdBy, 
					createdOn, 
					updatedOn
				) VALUES (?, ?, ?, ?, ?,? , ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		schedule.Name,
		schedule.Start,
		schedule.End,
		schedule.Timezone,
		schedule.TimeSlotDuration.Minutes(),
		schedule.LimitToOneBookingPerUser,
		schedule.IsShared,
		sundayAvailability,
		mondayAvailability,
		tuesdayAvailability,
		wednesdayAvailability,
		thursdayAvailability,
		fridayAvailability,
		saturdayAvailability,
		schedule.CreatedBy,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return err
	}

	schedule.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}

	for _, timeSlot := range schedule.TimeSlots {
		_, err := r.DB.ExecContext(ctx, `INSERT INTO timeSlot (start, end, scheduleId, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeSlot.Start, timeSlot.End, schedule.ID, time.Now(), time.Now())
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *SQLScheduleRepository) Update(ctx context.Context, schedule *Schedule) error {
	fmt.Println("updating schedule")
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	res, err := r.DB.ExecContext(ctx,
		`UPDATE schedule 
		SET
			name = ?,
			start = ?,
			end = ?,
			timezone = ?,
			timeslotDurationMin = ?,
			limitToOneBookingPerUser = ?,
			isShared = ?,
			updatedOn = ?
		WHERE id = ?
		`,
		schedule.Name,
		schedule.Start,
		schedule.End,
		schedule.Timezone,
		schedule.TimeSlotDuration.Minutes(),
		schedule.LimitToOneBookingPerUser,
		schedule.IsShared,
		time.Now(),
		schedule.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 1 {
		return fmt.Errorf("more than one row affected")
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

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
		ON t.id = b.timeSlotId AND t.bookingId = b.id
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
		ON t.id = b.timeSlotId AND t.bookingId = b.id
		WHERE t.scheduleId = ? AND t.start >= ? AND t.end <= ?`, scheduleId, start, end)
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

		timeslotSqid, err := r.sqid.Encode([]uint64{uint64(id)})
		if err != nil {
			return nil, fmt.Errorf("error encoding timeslot sqid: %w", err)
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

			bookingSqid, err := r.sqid.Encode([]uint64{uint64(bookingId.Int64)})
			if err != nil {
				return nil, fmt.Errorf("error encoding booking sqid: %w", err)
			}

			timeslots = append(timeslots, &TimeSlot{
				ID:        id,
				Sqid:      timeslotSqid,
				Start:     startTime,
				End:       endTime,
				CreatedOn: createdOnTime,
				UpdatedOn: updatedOnTime,
				Booking: &Booking{
					ID:          bookingId.Int64,
					Sqid:        bookingSqid,
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
			Sqid:      timeslotSqid,
			Start:     startTime,
			End:       endTime,
			CreatedOn: createdOnTime,
			UpdatedOn: updatedOnTime,
		})
	}

	return timeslots, nil
}

func (r *SQLScheduleRepository) CancelBooking(ctx context.Context, booking *Booking) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	_, err = r.DB.ExecContext(ctx, `UPDATE timeSlot SET bookingId = NULL WHERE id = ?`, booking.TimeSlot.ID)
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(ctx, `DELETE FROM booking WHERE id = ?`, booking.ID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	// if transaction is successful, set booking to nil
	booking.TimeSlot.Booking = nil

	return nil
}

func (r *SQLScheduleRepository) BookTimeSlot(ctx context.Context, scheduleId int64, timeslot *TimeSlot, bookerName string, bookerEmail string) error {
	var bookingId int64

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // defer rollback incase anything fails

	err = r.DB.QueryRowContext(ctx, `SELECT bookingId FROM timeSlot WHERE id = ? AND scheduleId = ? AND bookingId IS NOT NULL`, timeslot.ID, scheduleId).Scan(&bookingId)
	if err != sql.ErrNoRows {
		return fmt.Errorf("timeslot already booked. bookingId: %d", bookingId)
	}

	bookingCreatedOn := time.Now()
	bookingUpdatedOn := time.Now()

	res, err := r.DB.ExecContext(ctx, `INSERT INTO booking (timeSlotId, bookerName, bookerEmail, createdOn, updatedOn) VALUES (?, ?, ?, ?, ?)`, timeslot.ID, bookerName, bookerEmail, bookingCreatedOn, bookingUpdatedOn)
	if err != nil {
		return err
	}

	bookingId, err = res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(ctx, `UPDATE timeSlot SET bookingId = ? WHERE id = ?`, bookingId, timeslot.ID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	bookingSqid, err := r.sqid.Encode([]uint64{uint64(bookingId)})
	if err != nil {
		return fmt.Errorf("error encoding booking sqid: %w", err)
	}

	timeslot.Booking = &Booking{
		ID:          bookingId,
		Sqid:        bookingSqid,
		TimeSlot:    timeslot,
		BookerName:  bookerName,
		BookerEmail: bookerEmail,
		CreatedOn:   bookingCreatedOn,
		UpdatedOn:   bookingUpdatedOn,
	}

	return nil
}

func (r *SQLScheduleRepository) GetBookingById(ctx context.Context, bookingId int64) (*Booking, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT
			b.id,
			b.bookerName,
			b.bookerEmail,
			b.createdOn,
			b.updatedOn,
			t.id,
			t.start,
			t.end,
			t.createdOn,
			t.updatedOn
		FROM booking AS b
		LEFT OUTER JOIN timeSlot AS t
		ON b.timeSlotId = t.id
		WHERE b.id = ?`, bookingId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings, err := r.parseBookingSqlRows(rows)
	if err != nil {
		return nil, err
	}

	if len(bookings) == 0 {
		return nil, sql.ErrNoRows
	}

	if len(bookings) > 1 {
		return nil, fmt.Errorf("more than one booking found")
	}

	return bookings[0], nil

}

func (r *SQLScheduleRepository) GetBookings(ctx context.Context, scheduleId int64) ([]*Booking, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT 
			b.id,
			b.bookerName,
			b.bookerEmail,
			b.createdOn,
			b.updatedOn,
			t.id,
			t.start,
			t.end,
			t.createdOn,
			t.updatedOn
		FROM booking AS b
		LEFT OUTER JOIN timeSlot AS t
		ON b.timeSlotId = t.id
		WHERE t.scheduleId = ?`, scheduleId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.parseBookingSqlRows(rows)
}

func (r *SQLScheduleRepository) GetBookingsForUser(ctx context.Context, scheduleId int64, bookerEmail string) ([]*Booking, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT 
			b.id,
			b.bookerName,
			b.bookerEmail,
			b.createdOn,
			b.updatedOn,
			t.id,
			t.start,
			t.end,
			t.createdOn,
			t.updatedOn
		FROM booking AS b
		LEFT OUTER JOIN timeSlot AS t
		ON b.timeSlotId = t.id
		WHERE t.scheduleId = ? AND b.bookerEmail = ?`, scheduleId, bookerEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.parseBookingSqlRows(rows)
}

func (r *SQLScheduleRepository) parseBookingSqlRows(rows *sql.Rows) ([]*Booking, error) {
	var bookings []*Booking

	for rows.Next() {
		var bookingId int64
		var bookerName string
		var bookerEmail string
		var createdOn string
		var updatedOn string

		var timeSlotId sql.NullInt64
		var start string
		var end string
		var timeSlotCreatedOn string
		var timeSlotUpdatedOn string

		err := rows.Scan(&bookingId, &bookerName, &bookerEmail, &createdOn, &updatedOn, &timeSlotId, &start, &end, &timeSlotCreatedOn, &timeSlotUpdatedOn)
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

		bookingSqid, err := r.sqid.Encode([]uint64{uint64(bookingId)})
		if err != nil {
			return nil, fmt.Errorf("error encoding booking sqid: %w", err)
		}

		if !timeSlotId.Valid {
			bookings = append(bookings, &Booking{
				ID:          bookingId,
				Sqid:        bookingSqid,
				BookerName:  bookerName,
				BookerEmail: bookerEmail,
				CreatedOn:   createdOnTime,
				UpdatedOn:   updatedOnTime,
			})
			continue
		}

		timeslotSqid, err := r.sqid.Encode([]uint64{uint64(timeSlotId.Int64)})
		if err != nil {
			return nil, fmt.Errorf("error encoding timeslot sqid: %w", err)
		}

		startTime, err := time.Parse(r.dateLayout, start)
		if err != nil {
			return nil, err
		}

		endTime, err := time.Parse(r.dateLayout, end)
		if err != nil {
			return nil, err
		}

		timeSlotCreatedOnTime, err := time.Parse(r.dateLayout, timeSlotCreatedOn)
		if err != nil {
			return nil, err
		}

		timeSlotUpdatedOnTime, err := time.Parse(r.dateLayout, timeSlotUpdatedOn)
		if err != nil {
			return nil, err
		}

		bookings = append(bookings, &Booking{
			ID:          bookingId,
			Sqid:        bookingSqid,
			BookerName:  bookerName,
			BookerEmail: bookerEmail,
			CreatedOn:   createdOnTime,
			UpdatedOn:   updatedOnTime,
			TimeSlot: &TimeSlot{
				ID:        timeSlotId.Int64,
				Sqid:      timeslotSqid,
				Start:     startTime,
				End:       endTime,
				CreatedOn: timeSlotCreatedOnTime,
				UpdatedOn: timeSlotUpdatedOnTime,
			},
		})
	}

	return bookings, nil
}
