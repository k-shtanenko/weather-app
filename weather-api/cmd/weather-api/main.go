package main

import (
	_ "github.com/k-shtanenko/weather-app/weather-api/docs"
	"github.com/k-shtanenko/weather-app/weather-api/internal/bootstrap"
)

// @schemes http

// @title Weather API Service
// @version 1.0.0
// @description Микросервис для обработки погодных данных и генерации отчетов.

// @host localhost:8080
// @BasePath /api/v1

func main() {
	bootstrap.Bootstrap()
}
