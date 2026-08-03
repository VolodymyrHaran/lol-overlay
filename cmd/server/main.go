package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lol-timer/internal/app"

	"github.com/joho/godotenv"
)

// @title           LoL Timer API
// @version         1.0
// @description     API for tracking League of Legends summoner spell cooldowns.
// @description     Rooms are synchronized with the League Client and updated in real time.
// @description     PostgreSQL is used for persistence and Redis for caching.
//
// @host      localhost:8080
// @BasePath  /
//
// @schemes http
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			".env file not found, using system environment variables",
		)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	application, err := app.New()
	if err != nil {
		log.Fatal("create application: ", err)
	}
	defer application.Close()

	if err := application.Run(ctx); err != nil {
		log.Fatal("run application: ", err)
	}

	log.Println("application stopped")
}
