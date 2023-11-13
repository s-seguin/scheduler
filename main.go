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
	scheduleController := NewScheduleController(scheduleService)

	// todo -- we should reorg this so that we aren't passing the server to the controller
	scheduleController.UseMiddleware(server.IsAuthenticated)

	server.ServeStatic("/static", http.Dir("./public/static"))
	server.MountRouter("/schedules", scheduleController.MountRoutes())
	server.Start()
}
