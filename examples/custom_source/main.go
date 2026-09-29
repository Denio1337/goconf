package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Denio1337/goenv"
)

// JSONSource demonstrates how easily other configuration formats
// (JSON, YAML, TOML, INI, Consul, etcd, etc.) can be added by implementing
// the goenv.Source interface.
type JSONSource struct {
	path string
}

func NewJSONSource(path string) *JSONSource {
	return &JSONSource{path: path}
}

func (s *JSONSource) Name() string {
	return "json:" + s.path
}

func (s *JSONSource) Load(ctx context.Context) (map[string]any, error) {
	bytes, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading json config: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return data, nil
}

type MetricsConfig struct {
	Enabled bool   `env:"ENABLED" default:"false"`
	Path    string `env:"PATH" default:"/metrics"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST"`
	Port     int    `env:"PORT" default:"5432"`
	Password string `env:"PASSWORD" required:"true"`
}

type Config struct {
	AppName  string         `env:"APP_NAME"`
	Port     int            `env:"PORT"`
	Metrics  MetricsConfig  `prefix:"metrics."`
	Database DatabaseConfig `prefix:"database."`
}

func main() {
	var cfg Config

	// Load configuration using custom JSON source
	err := goenv.Load(&cfg, goenv.WithSource(NewJSONSource("config.json")))
	if err != nil {
		log.Fatalf("failed to load json configuration: %v", err)
	}

	fmt.Println("Successfully loaded config from custom JSON source!")
	fmt.Printf("App: %s (Port: %d)\n", cfg.AppName, cfg.Port)
	fmt.Printf("Metrics: enabled=%t, path=%s\n", cfg.Metrics.Enabled, cfg.Metrics.Path)
	fmt.Printf("Database: %s:%d (Password: %s)\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.Password)
}
