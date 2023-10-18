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

	server.ServeStatic("/static", http.Dir("./public/static"))
	server.MountRouter("/schedules", scheduleController.MountRoutes())
	server.Start()
}
