package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Denio1337/goconf"
)

type ServerConfig struct {
	Host     string        `key:"HOST"`
	Port     int           `key:"PORT"`
	Timeout  time.Duration `key:"TIMEOUT"`
	MaxConns int           `key:"MAX_CONNS"`
}

type DatabaseConfig struct {
	Host     string `key:"HOST"`
	Port     int    `key:"PORT"`
	User     string `key:"USER"`
	Password string `key:"PASSWORD"`
	Database string `key:"NAME"`
}

type Config struct {
	AppName     string         `key:"APP_NAME"`
	Environment string         `key:"ENVIRONMENT"`
	Debug       bool           `key:"DEBUG"`
	Features    []string       `key:"FEATURES"`
	Server      ServerConfig   `prefix:"SERVER_"`
	Database    DatabaseConfig `prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	// Load configuration through a 5-layer cascade:
	// Layer 1: config.env   - Base defaults
	// Layer 2: config.ini   - Overrides environment & database user/password
	// Layer 3: config.json  - Overrides server port/timeout & database name
	// Layer 4: config.yaml  - Overrides app name, debug flag & features slice
	// Layer 5: config.toml  - Overrides production environment, server host/max_conns & database host
	err := goconf.Load(&cfg,
		goconf.WithDotEnv("config.env"),
		goconf.WithINI("config.ini"),
		goconf.WithJSON("config.json"),
		goconf.WithYAML("config.yaml"),
		goconf.WithTOML("config.toml"),
	)
	if err != nil {
		log.Fatalf("Failed to load complex configuration: %v", err)
	}

	fmt.Println("================================================================================")
	fmt.Println("             5-LAYER CASCADING CONFIGURATION LOADED SUCCESSFULLY                ")
	fmt.Println("================================================================================")
	fmt.Println()

	fmt.Printf("%-25s : %-30s [from config.yaml (overrode .env)]\n", "App Name", cfg.AppName)
	fmt.Printf("%-25s : %-30s [from config.toml (overrode ini & .env)]\n", "Environment", cfg.Environment)
	fmt.Printf("%-25s : %-30t [from config.yaml (overrode .env)]\n", "Debug", cfg.Debug)
	fmt.Printf("%-25s : [%s] [from config.yaml (overrode .env)]\n", "Features", strings.Join(cfg.Features, ", "))
	fmt.Println()

	fmt.Println("--- Server Configuration ---")
	fmt.Printf("%-25s : %-30s [from config.toml (overrode .env)]\n", "Server Host", cfg.Server.Host)
	fmt.Printf("%-25s : %-30d [from config.json (overrode .env)]\n", "Server Port", cfg.Server.Port)
	fmt.Printf("%-25s : %-30v [from config.json (overrode .env)]\n", "Server Timeout", cfg.Server.Timeout)
	fmt.Printf("%-25s : %-30d [from config.toml (overrode .env)]\n", "Server Max Conns", cfg.Server.MaxConns)
	fmt.Println()

	fmt.Println("--- Database Configuration ---")
	fmt.Printf("%-25s : %-30s [from config.toml (overrode .env)]\n", "DB Host", cfg.Database.Host)
	fmt.Printf("%-25s : %-30d [from config.env  (base default preserved)]\n", "DB Port", cfg.Database.Port)
	fmt.Printf("%-25s : %-30s [from config.ini  (overrode .env)]\n", "DB User", cfg.Database.User)
	fmt.Printf("%-25s : %-30s [from config.ini  (overrode .env)]\n", "DB Password", cfg.Database.Password)
	fmt.Printf("%-25s : %-30s [from config.json (overrode .env)]\n", "DB Name", cfg.Database.Database)
	fmt.Println()
	fmt.Println("================================================================================")
}
