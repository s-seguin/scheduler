package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load the env vars: %v", err)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatal("Error creating server", err)
	}
	server.SetupMiddleware()
	server.MountAuthRoutes()

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
	scheduleController := NewScheduleController(scheduleService, server.Store)

	server.ServeStatic("/static", http.Dir("./public/static"))
	server.MountRouter("/", scheduleController.MountRoutes())
	server.Start()
}
