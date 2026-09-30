package goconf_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Denio1337/goconf"
	"github.com/Denio1337/goconf/source/env"
)

type ServerConfig struct {
	Host     string        `key:"HOST" default:"localhost"`
	Port     int           `key:"PORT" default:"8080"`
	Timeout  time.Duration `key:"TIMEOUT" default:"10s"`
	Secret   string        `key:"SECRET" required:"true"`
	Features []string      `key:"FEATURES"`
}

func TestLoadWithPrefix(t *testing.T) {
	dotenvContent := `
APP_HOST=0.0.0.0
APP_PORT=9090
APP_TIMEOUT=30s
APP_SECRET=super-secret-token
APP_FEATURES=metrics,tracing,auth
`
	var cfg ServerConfig
	err := goconf.Load(&cfg,
		goconf.WithPrefix("APP_"),
		goconf.WithDotEnvReader(strings.NewReader(dotenvContent)),
	)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host: expected 0.0.0.0, got %q", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port: expected 9090, got %d", cfg.Port)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout: expected 30s, got %v", cfg.Timeout)
	}
	if cfg.Secret != "super-secret-token" {
		t.Errorf("Secret: expected super-secret-token, got %q", cfg.Secret)
	}
	if len(cfg.Features) != 3 || cfg.Features[1] != "tracing" {
		t.Errorf("Features: expected [metrics, tracing, auth], got %v", cfg.Features)
	}
}

func TestLoadWithINI(t *testing.T) {
	iniContent := `
[server]
host = 127.0.0.1
port = 7070
timeout = 45s
secret = ini-secret-token
features = logging,monitoring
`
	var cfg struct {
		Server ServerConfig `prefix:"SERVER_"`
	}

	err := goconf.Load(&cfg, goconf.WithINIReader(strings.NewReader(iniContent)))
	if err != nil {
		t.Fatalf("unexpected load error from INI: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Host: expected 127.0.0.1, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("Port: expected 7070, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != 45*time.Second {
		t.Errorf("Timeout: expected 45s, got %v", cfg.Server.Timeout)
	}
	if cfg.Server.Secret != "ini-secret-token" {
		t.Errorf("Secret: expected ini-secret-token, got %q", cfg.Server.Secret)
	}
	if len(cfg.Server.Features) != 2 || cfg.Server.Features[0] != "logging" {
		t.Errorf("Features: expected [logging, monitoring], got %v", cfg.Server.Features)
	}
}

func TestLoadWithJSON(t *testing.T) {
	jsonContent := `{
		"server": {
			"host": "192.168.1.10",
			"port": 9000,
			"timeout": "25s",
			"secret": "json-jwt-secret",
			"features": ["api", "graphql"]
		}
	}`

	var cfg struct {
		Server ServerConfig `prefix:"SERVER_"`
	}

	err := goconf.Load(&cfg, goconf.WithJSONReader(strings.NewReader(jsonContent)))
	if err != nil {
		t.Fatalf("unexpected load error from JSON: %v", err)
	}

	if cfg.Server.Host != "192.168.1.10" {
		t.Errorf("Host: expected 192.168.1.10, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Port: expected 9000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != 25*time.Second {
		t.Errorf("Timeout: expected 25s, got %v", cfg.Server.Timeout)
	}
	if cfg.Server.Secret != "json-jwt-secret" {
		t.Errorf("Secret: expected json-jwt-secret, got %q", cfg.Server.Secret)
	}
	if len(cfg.Server.Features) != 2 || cfg.Server.Features[0] != "api" {
		t.Errorf("Features: expected [api, graphql], got %v", cfg.Server.Features)
	}
}

func TestLoadWithYAML(t *testing.T) {
	yamlContent := `
server:
  host: "10.0.0.5"
  port: 8088
  timeout: "12s"
  secret: "yaml-token-secret"
  features:
    - grpc
    - rest
`

	var cfg struct {
		Server ServerConfig `prefix:"SERVER_"`
	}

	err := goconf.Load(&cfg, goconf.WithYAMLReader(strings.NewReader(yamlContent)))
	if err != nil {
		t.Fatalf("unexpected load error from YAML: %v", err)
	}

	if cfg.Server.Host != "10.0.0.5" {
		t.Errorf("Host: expected 10.0.0.5, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8088 {
		t.Errorf("Port: expected 8088, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != 12*time.Second {
		t.Errorf("Timeout: expected 12s, got %v", cfg.Server.Timeout)
	}
	if cfg.Server.Secret != "yaml-token-secret" {
		t.Errorf("Secret: expected yaml-token-secret, got %q", cfg.Server.Secret)
	}
	if len(cfg.Server.Features) != 2 || cfg.Server.Features[0] != "grpc" {
		t.Errorf("Features: expected [grpc, rest], got %v", cfg.Server.Features)
	}
}

func TestLoadWithTOML(t *testing.T) {
	tomlContent := `
[server]
host = "10.0.0.12"
port = 7070
timeout = "18s"
secret = "toml-token-secret"
features = ["metrics", "tracing"]
`

	var cfg struct {
		Server ServerConfig `prefix:"SERVER_"`
	}

	err := goconf.Load(&cfg, goconf.WithTOMLReader(strings.NewReader(tomlContent)))
	if err != nil {
		t.Fatalf("unexpected load error from TOML: %v", err)
	}

	if cfg.Server.Host != "10.0.0.12" {
		t.Errorf("Host: expected 10.0.0.12, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("Port: expected 7070, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != 18*time.Second {
		t.Errorf("Timeout: expected 18s, got %v", cfg.Server.Timeout)
	}
	if cfg.Server.Secret != "toml-token-secret" {
		t.Errorf("Secret: expected toml-token-secret, got %q", cfg.Server.Secret)
	}
	if len(cfg.Server.Features) != 2 || cfg.Server.Features[0] != "metrics" {
		t.Errorf("Features: expected [metrics, tracing], got %v", cfg.Server.Features)
	}
}

func TestStrictUnknownKeys(t *testing.T) {
	envContent := "PORT=8080\nSECRET=xyz\nUNKNOWN_TYPO=something\n"

	var cfg ServerConfig
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(envContent)),
		goconf.WithStrictUnknown(true),
	)
	if err == nil {
		t.Fatal("expected strict error on unknown key, got nil")
	}

	var valErr *goconf.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	foundUnknown := false
	for _, e := range valErr.Errors {
		if strings.Contains(e.Err.Error(), "UNKNOWN_TYPO") {
			foundUnknown = true
			break
		}
	}
	if !foundUnknown {
		t.Errorf("expected unknown key error for UNKNOWN_TYPO in: %v", err)
	}
}

func TestMustLoadPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected MustLoad to panic on missing required field")
		}
	}()

	var cfg ServerConfig
	// Missing SECRET will fail validation
	goconf.MustLoad(&cfg, goconf.WithDotEnvReader(strings.NewReader("PORT=1234\n")))
}

func TestMissingDotEnvFile(t *testing.T) {
	var cfg ServerConfig
	// Default behavior should fail if file is specified and missing
	err := goconf.Load(&cfg, goconf.WithDotEnv("non_existent_file.env"))
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	// But WithIgnoreMissing(true) should not fail on the file opening
	err = goconf.Load(&cfg,
		goconf.WithIgnoreMissing(true),
		goconf.WithDotEnv("non_existent_file.env"),
	)
	var valErr *goconf.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestCascadingMultiSourceOverrides(t *testing.T) {
	type Config struct {
		AppName string `key:"APP_NAME"`
		Port    int    `key:"PORT"`
		Debug   bool   `key:"DEBUG"`
	}

	envContent := "APP_NAME=FromEnv\nPORT=8000\nDEBUG=true\n"
	jsonContent := `{"port": 8080}`
	yamlContent := "app_name: FromYAML\ndebug: false\n"

	var cfg Config
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(envContent)),
		goconf.WithJSONReader(strings.NewReader(jsonContent)),
		goconf.WithYAMLReader(strings.NewReader(yamlContent)),
	)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.AppName != "FromYAML" {
		t.Errorf("AppName: expected FromYAML, got %q", cfg.AppName)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port: expected 8080 (from JSON), got %d", cfg.Port)
	}
	if cfg.Debug != false {
		t.Errorf("Debug: expected false (from YAML), got %t", cfg.Debug)
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Setenv("TEST_APP_SERVER_PORT", "9999")
	os.Setenv("TEST_APP_SERVER_HOST", "127.0.0.1")
	os.Setenv("TEST_APP_SECRET", "env-secret")
	defer func() {
		os.Unsetenv("TEST_APP_SERVER_PORT")
		os.Unsetenv("TEST_APP_SERVER_HOST")
		os.Unsetenv("TEST_APP_SECRET")
	}()

	type Config struct {
		Server struct {
			Host string `key:"HOST"`
			Port int    `key:"PORT"`
		} `prefix:"SERVER_"`
		Secret string `key:"SECRET"`
	}

	var cfg Config
	// Pure environment variables, no config file
	err := goconf.Load(&cfg, goconf.WithEnvPrefix("TEST_APP_"))
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected Server.Port=9999, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected Server.Host=127.0.0.1, got %q", cfg.Server.Host)
	}
	if cfg.Secret != "env-secret" {
		t.Errorf("expected Secret=env-secret, got %q", cfg.Secret)
	}
}

func TestLoadWithEnvOverride(t *testing.T) {
	dotEnvContent := "PORT=8080\nDEBUG=false\n"

	var cfg struct {
		Port  int  `key:"PORT"`
		Debug bool `key:"DEBUG"`
	}

	// Environment variable overrides .env
	customEnv := []string{
		"PORT=9090",
	}

	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(dotEnvContent)),
		goconf.WithEnv(env.WithEnviron(customEnv)),
	)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected Port=9090 (from env override), got %d", cfg.Port)
	}
	if cfg.Debug != false {
		t.Errorf("expected Debug=false (from .env), got %t", cfg.Debug)
	}
}

func TestLoadDefaultBehavior(t *testing.T) {
	// 1. When no .env exists and only OS env is provided
	os.Setenv("PORT", "7777")
	os.Setenv("APP_NAME", "DefaultEnvApp")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("APP_NAME")
	}()

	var cfg struct {
		Port    int    `key:"PORT"`
		AppName string `key:"APP_NAME"`
		Host    string `key:"HOST" default:"localhost"`
	}

	// Calling Load with only &cfg (no options passed)
	err := goconf.Load(&cfg)
	if err != nil {
		t.Fatalf("unexpected error when loading defaults with only OS env: %v", err)
	}

	if cfg.Port != 7777 {
		t.Errorf("Port: expected 7777, got %d", cfg.Port)
	}
	if cfg.AppName != "DefaultEnvApp" {
		t.Errorf("AppName: expected DefaultEnvApp, got %q", cfg.AppName)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host: expected localhost, got %q", cfg.Host)
	}
}

func TestLoadDefaultBehavior_EnvOverridesDotEnv(t *testing.T) {
	// Create temporary .env in current directory
	envContent := []byte("PORT=8080\nDB_NAME=dotenv_db\n")
	if err := os.WriteFile(".env", envContent, 0644); err != nil {
		t.Fatalf("failed to create temp .env: %v", err)
	}
	defer os.Remove(".env")

	// Set OS environment variable that should override .env
	os.Setenv("PORT", "9999")
	defer os.Unsetenv("PORT")

	var cfg struct {
		Port   int    `key:"PORT"`
		DBName string `key:"DB_NAME"`
	}

	// Call Load with only &cfg
	err := goconf.Load(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// OS env has primary priority and must override .env
	if cfg.Port != 9999 {
		t.Errorf("Port: expected 9999 (from OS env), got %d", cfg.Port)
	}
	// DBName comes from .env since it wasn't in OS env
	if cfg.DBName != "dotenv_db" {
		t.Errorf("DBName: expected dotenv_db (from .env), got %q", cfg.DBName)
	}
}
