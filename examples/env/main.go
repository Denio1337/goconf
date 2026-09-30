package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Denio1337/goconf"
)

type ServerConfig struct {
	Host         string        `default:"localhost"`
	Port         int           `default:"8080"`
	ReadTimeout  time.Duration `default:"5s"`
	WriteTimeout time.Duration `default:"10s"`
}

type DatabaseConfig struct {
	Host     string
	Port     int    `default:"5432"`
	User     string `default:"postgres"`
	Password string `required:"true"`
	MaxConns int    `default:"10"`
}

type AppConfig struct {
	AppName        string
	Environment    string `default:"development"`
	Debug          bool   `default:"false"`
	AllowedOrigins []string
	Server         ServerConfig
	Database       DatabaseConfig
}

func main() {
	var cfg AppConfig

	// Load configuration: checks OS environment variables with top priority,
	// falling back to .env file if present.
	if err := goconf.Load(&cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("Configuration successfully loaded!")
	fmt.Printf("App: %s (Env: %s, Debug: %t)\n", cfg.AppName, cfg.Environment, cfg.Debug)
	fmt.Printf("Server listening on %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d (MaxConns: %d)\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.MaxConns)
	fmt.Printf("Allowed Origins: %v\n", cfg.AllowedOrigins)
}
