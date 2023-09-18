package main

import (
	"log"
)

func main() {
	server := NewServer()
	server.SetupMiddleware()

	// todo -- either flag this or delete it
	DeleteTestDb()

	db, err := CreateDb("./_sqlite/scheduler.db")
	if err != nil {
		log.Fatal("Error creating db", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		log.Fatal("Error performing migrations", err)
	}

	scheduleRepository := NewSQLScheduleRepository(db)
	scheduleService := NewScheduleService(scheduleRepository)
	scheduleController := NewScheduleController(scheduleService)

	// todo -- move this to a test
	s, err := scheduleService.CreateSchedule("test", "stuart seguin")
	if err != nil {
		log.Fatal(err)
	}

	wa := scheduleService.CreateDefaultWeeklyAvailability(s.ID)
	_ = scheduleService.GenerateTimeSlots(s.ID, wa)

	log.Println("Timeslots within range")
	timeslots, err := scheduleService.GetTimeSlotsWithinRange(s.ID, wa.StartDate.AddDate(0, 0, 2), wa.StartDate.AddDate(0, 0, 3))
	if err != nil {
		log.Fatal(err)
	}

	for i, timeslot := range timeslots {
		if i == 0 {
			bookingId, err := scheduleService.BookTimeSlot(s.ID, timeslot, "stuart", "s@g.com")
			bookingId, err = scheduleService.BookTimeSlot(s.ID, timeslot, "stuart", "should fail")
			if err != nil {
				log.Println("error booking timeslot", err)
			} else {
				log.Println("booking id: ", bookingId)
			}
		}
		log.Println("time slots within range: ", timeslot.ID, timeslot.Start.Local(), timeslot.End.Local(), timeslot.Booking, timeslot.IsAvailable())
	}

	server.MountRouter("/schedules", scheduleController.MountRoutes())
	server.Start()
}
