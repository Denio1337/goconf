package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Denio1337/goconf"
)

type ServerConfig struct {
	Host         string        `env:"HOST" default:"localhost"`
	Port         int           `env:"PORT" default:"8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST"`
	Port     int    `env:"PORT" default:"5432"`
	User     string `env:"USER" default:"postgres"`
	Password string `env:"PASSWORD" required:"true"`
	MaxConns int    `env:"MAX_CONNS" default:"10"`
}

type AppConfig struct {
	AppName        string         `env:"APP_NAME"`
	Environment    string         `env:"ENVIRONMENT" default:"development"`
	Debug          bool           `env:"DEBUG" default:"false"`
	AllowedOrigins []string       `env:"ALLOWED_ORIGINS"`
	Server         ServerConfig   `env-prefix:"SERVER_"`
	Database       DatabaseConfig `env-prefix:"DATABASE_"`
}

func main() {
	var cfg AppConfig

	// Load configuration from .env file
	if err := goconf.Load(&cfg, goconf.WithDotEnv(".env")); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("Configuration successfully loaded!")
	fmt.Printf("App: %s (Env: %s, Debug: %t)\n", cfg.AppName, cfg.Environment, cfg.Debug)
	fmt.Printf("Server listening on %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d (MaxConns: %d)\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.MaxConns)
	fmt.Printf("Allowed Origins: %v\n", cfg.AllowedOrigins)
}
