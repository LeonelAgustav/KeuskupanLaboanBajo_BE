package main

import (
	"fmt"
	"os"
	"time"

	"KeuskupanLaboanBajo_BE/config"
	"KeuskupanLaboanBajo_BE/middleware"
	"KeuskupanLaboanBajo_BE/routes"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func main() {
	db := config.ConnectDB()

	// ponytail: env validation - prevents running in production with weak/default secrets
	if os.Getenv("APP_ENV") == "production" {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" || secret == "keuskupan-dev-secret-ganti-di-prod" {
			fmt.Println("❌ CRITICAL: JWT_SECRET must be set to a strong unique value in production!")
			os.Exit(1)
		}
		if dbPass := os.Getenv("DB_PASS"); dbPass == "" || dbPass == "ARCS" {
			fmt.Println("❌ CRITICAL: DB_PASS must not be default in production!")
			os.Exit(1)
		}
	}

	middleware.DB = db

	e := echo.New()

	e.HTTPErrorHandler = middleware.ErrorHandler
	e.Validator = &middleware.CustomValidator{Validator: validator.New()}

	routes.RegisterRoutes(e, db)

	// OWASP: server timeouts, prevents slowloris DoS
	e.Server.ReadTimeout = 15 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 60 * time.Second
	e.Server.MaxHeaderBytes = 1 << 20 // 1MB

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	bindAddr := os.Getenv("BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "0.0.0.0:" + port
	} else {
		bindAddr = bindAddr + ":" + port
	}
	fmt.Printf("starting web server at %s\n", bindAddr)
	e.Logger.Fatal(e.Start(bindAddr))
}
