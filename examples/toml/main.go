package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Denio1337/goconf"
)

type ServerConfig struct {
	Host         string        `key:"HOST" default:"localhost"`
	Port         int           `key:"PORT" default:"8080"`
	ReadTimeout  time.Duration `key:"READ_TIMEOUT" default:"5s"`
	WriteTimeout time.Duration `key:"WRITE_TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	Host     string `key:"HOST"`
	Port     int    `key:"PORT" default:"5432"`
	User     string `key:"USER" default:"postgres"`
	Password string `key:"PASSWORD" required:"true"`
	MaxConns int    `key:"MAX_CONNS" default:"10"`
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

	// Load configuration from TOML file using built-in WithTOML option
	if err := goconf.Load(&cfg, goconf.WithTOML("config.toml")); err != nil {
		log.Fatalf("Error loading TOML config: %v", err)
	}

	fmt.Println("=== Successfully Loaded Configuration from TOML ===")
	fmt.Printf("App: %s (Debug: %t)\n", cfg.AppName, cfg.Debug)
	fmt.Printf("Server: %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d (Password: %s, MaxConns: %d)\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Password, cfg.Database.MaxConns)
	fmt.Printf("Allowed Origins: %v\n", cfg.AllowedOrigins)
}
