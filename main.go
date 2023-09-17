package main

import (
	"log"
)

func main() {
	server := NewServer()
	server.SetupMiddleware()

	db, err := CreateDb("./_sqlite/scheduler.db")
	if err != nil {
		log.Fatal("Error creating db", err)
	}

	err = Migrate(db)
	if err != nil {
		log.Fatal("Error performing migrations", err)
	}

	scheduleRepository := NewSQLScheduleRepository(db)
	scheduleService := NewScheduleService(scheduleRepository)
	scheduleController := NewScheduleController(scheduleService)

	server.MountRouter("/", scheduleController.MountRoutes())
	server.Start()
}
