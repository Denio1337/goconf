package goconf_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Denio1337/goconf"
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
