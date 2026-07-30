package main

import (
	"log"

	"github.com/joho/godotenv"

	"lol-timer/internal/app"
)

func main() {

	_ = godotenv.Load()

	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
