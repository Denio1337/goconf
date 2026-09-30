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
	Metrics      struct {
		Enabled  bool
		Endpoint string
	}
}

type DatabaseConfig struct {
	Host           string `key:"HOST"`
	Port           int    `key:"PORT" default:"5432"`
	User           string `key:"USER" default:"postgres"`
	Password       string `key:"PASSWORD" required:"true"`
	MaxConnections int    `key:"MAX_CONNECTIONS" default:"10"`
}

type Config struct {
	AppName  string
	Debug    bool
	Server   ServerConfig
	Database DatabaseConfig
}

func main() {
	var cfg Config

	// Load configuration from INI file
	if err := goconf.Load(&cfg, goconf.WithINI("config.ini")); err != nil {
		log.Fatalf("Error loading INI config: %v", err)
	}

	fmt.Println("=== Successfully Loaded Configuration from INI ===")
	fmt.Printf("App: %s (Debug: %t)\n", cfg.AppName, cfg.Debug)
	fmt.Printf("Server: %s:%d (ReadTimeout: %v, Metrics: enabled=%t on %s)\n",
		cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout,
		cfg.Server.Metrics.Enabled, cfg.Server.Metrics.Endpoint)
	fmt.Printf("Database: %s@%s:%d (Password: %s, MaxConns: %d)\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Password, cfg.Database.MaxConnections)
}
