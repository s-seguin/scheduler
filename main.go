package main

import (
	"log"
	"net/http"
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
		timeslot := timeslot
		if i == 0 {
			bookingId, err := scheduleService.BookTimeSlot(s.ID, timeslot.ID, "stuart", "s@g.com")
			if err != nil {
				log.Fatal("error booking timeslot", err)
			}
			log.Println("booking id: ", bookingId)
			bookingId, err = scheduleService.BookTimeSlot(s.ID, timeslot.ID, "stuart", "should fail")
			if err != nil {
				log.Println("error booking timeslot", err)
			} else {
				log.Println("booking id: ", bookingId)
			}
		}
		log.Println("time slots within range: ", timeslot.ID, timeslot.Start.Local(), timeslot.End.Local(), timeslot.Booking, timeslot.IsAvailable())
	}

	timeslots, err = scheduleService.GetTimeSlotsWithinRange(s.ID, wa.StartDate.AddDate(0, 0, 2), wa.StartDate.AddDate(0, 0, 3))
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range timeslots {
		t := *t
		log.Println("ts: ", t.ID, t.Start.Local(), t.End.Local(), t.Booking, t.IsAvailable())
	}

	server.ServeStatic("/static", http.Dir("./public/static"))
	server.MountRouter("/schedules", scheduleController.MountRoutes())
	server.Start()
}
