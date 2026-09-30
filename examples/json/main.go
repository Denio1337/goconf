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

type Config struct {
	AppName        string         `key:"APP_NAME"`
	Debug          bool           `key:"DEBUG"`
	AllowedOrigins []string       `key:"ALLOWED_ORIGINS"`
	Server         ServerConfig   `prefix:"SERVER_"`
	Database       DatabaseConfig `prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	// Load configuration from JSON file using built-in WithJSON option
	if err := goconf.Load(&cfg, goconf.WithJSON("config.json")); err != nil {
		log.Fatalf("Error loading JSON config: %v", err)
	}

	fmt.Println("=== Successfully Loaded Configuration from JSON ===")
	fmt.Printf("App: %s (Debug: %t)\n", cfg.AppName, cfg.Debug)
	fmt.Printf("Server: %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d (Password: %s, MaxConns: %d)\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Password, cfg.Database.MaxConns)
	fmt.Printf("Allowed Origins: %v\n", cfg.AllowedOrigins)
}
