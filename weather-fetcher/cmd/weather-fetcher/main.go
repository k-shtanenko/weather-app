package main

import (
	"log"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/bootstrap"
)

func main() {
	app, err := bootstrap.NewBootstrap()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
