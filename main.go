package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/sqids/sqids-go"
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
	// log.Println("Deleting test db")
	//	DeleteTestDb()

	log.Println("Creating db")
	db, err := CreateDb("./_sqlite/scheduler.db")
	if err != nil {
		log.Fatal("Error creating db", err)
	}
	defer db.Close()

	log.Println("Migrating db")
	err = Migrate(db)
	if err != nil {
		log.Fatal("Error performing migrations ", err)
	}

	sqid, err := sqids.New(sqids.Options{MinLength: 6})
	if err != nil {
		log.Fatal("Issue initializing sqid")
	}

	scheduleRepository := NewSQLScheduleRepository(db, sqid)
	scheduleService := NewScheduleService(scheduleRepository)
	scheduleController := NewScheduleController(scheduleService, server.Store, sqid)

	server.ServeStatic("/static", http.Dir("./public/static"))
	server.MountRouter("/", scheduleController.MountRoutes())
	server.Start()
}
