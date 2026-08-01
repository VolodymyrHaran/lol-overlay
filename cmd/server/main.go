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
