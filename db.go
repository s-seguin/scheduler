package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func DeleteTestDb() {
	os.Remove("./_sqlite/scheduler.db")
	os.Remove("./_sqlite/schduler.db-shm")
	os.Remove("./_sqlite/scheduler.db-wal")
}

func CreateDb(name string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	// enable write ahead log
	_, err = db.Exec(`PRAGMA journal_mode = wal;`)
	if err != nil {
		return nil, err
	}

	// enable foreign keys
	_, err = db.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return nil, err
	}

	return db, nil

}

// todo -- should we make a DB struct and make this a receiver?
func Migrate(db *sql.DB) error {
	// todo -- read the migration files from a folder and perform
	//	for now just hard code in functions
	err := createUserTable(db)
	if err != nil {
		return err
	}

	err = createScheduleTable(db)
	if err != nil {
		return err
	}

	err = createTimeSlotTable(db)
	if err != nil {
		return err
	}

	err = createBookingTable(db)
	if err != nil {
		return err
	}

	return nil
}

func createUserTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user (
			id INTEGER NOT NULL PRIMARY KEY,
			name TEXT, 
			email TEXT,
			createdOn TEXT,
			updatedOn TEXT
		);
	`)
	return err
}

func createScheduleTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schedule (
			id INTEGER NOT NULL PRIMARY KEY,
			-- userId INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
			createdBy TEXT,
			name TEXT,
			start TEXT,
			end TEXT,
			timeSlotDurationMin INTEGER,
			timezone TEXT,
			limitToOneBookingPerUser BOOLEAN,
			isShared BOOLEAN,
			sundayAvailability TEXT,
			mondayAvailability TEXT,
			tuesdayAvailability TEXT,
			wednesdayAvailability TEXT,
			thursdayAvailability TEXT,
			fridayAvailability TEXT,
			saturdayAvailability TEXT,
			createdOn TEXT,
			updatedOn TEXT
		);
	`)
	return err
}

func createTimeSlotTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS timeSlot (
			id INTEGER NOT NULL PRIMARY KEY,
			scheduleId INTEGER NOT NULL REFERENCES schedule(id) ON DELETE CASCADE,
			start TEXT, 
			end TEXT,
			bookingId INTEGER REFERENCES booking(id) ON DELETE SET NULL,
			createdOn TEXT,
			updatedOn TEXT,
			UNIQUE(id, bookingId),
			UNIQUE(scheduleId, start, end)
		);
	`)
	return err
}

func createBookingTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS booking (
			id INTEGER NOT NULL PRIMARY KEY,
			timeSlotId INTEGER NOT NULL REFERENCES timeSlot(id) ON DELETE CASCADE,
			bookerName TEXT,
			bookerEmail TEXT,
			createdOn TEXT,
			updatedOn TEXT,
			UNIQUE(id, timeSlotId)
		);


	`)
	return err
}
