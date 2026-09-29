package goenv_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Denio1337/goenv"
	"github.com/Denio1337/goenv/source/mapsource"
)

type ServerConfig struct {
	Host     string        `env:"HOST" default:"localhost"`
	Port     int           `env:"PORT" default:"8080"`
	Timeout  time.Duration `env:"TIMEOUT" default:"10s"`
	Secret   string        `env:"SECRET" required:"true"`
	Features []string      `env:"FEATURES"`
}

func TestLoadFromReader(t *testing.T) {
	dotenvContent := `
HOST=0.0.0.0
PORT=9090
TIMEOUT=30s
SECRET=super-secret-token
FEATURES=metrics,tracing,auth
`
	var cfg ServerConfig
	err := goenv.Load(&cfg, goenv.WithDotEnvReader(strings.NewReader(dotenvContent)))
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

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := "PORT=7777\nSECRET=from-file\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	var cfg ServerConfig
	err := goenv.Load(&cfg, goenv.WithDotEnv(envPath))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 7777 {
		t.Errorf("Port: expected 7777, got %d", cfg.Port)
	}
	if cfg.Secret != "from-file" {
		t.Errorf("Secret: expected from-file, got %q", cfg.Secret)
	}
	if cfg.Host != "localhost" { // fallback to default
		t.Errorf("Host: expected default localhost, got %q", cfg.Host)
	}
}

func TestSourceOverrides(t *testing.T) {
	baseEnv := "PORT=8080\nSECRET=base-secret\nHOST=base.domain\n"
	overrideSource := mapsource.New(map[string]any{
		"PORT": "9999",
		"HOST": "override.domain",
	}, mapsource.WithName("override"))

	var cfg ServerConfig
	err := goenv.Load(&cfg,
		goenv.WithDotEnvReader(strings.NewReader(baseEnv)),
		goenv.WithSource(overrideSource),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Port and Host should be overridden by the second source
	if cfg.Port != 9999 {
		t.Errorf("Port: expected 9999, got %d", cfg.Port)
	}
	if cfg.Host != "override.domain" {
		t.Errorf("Host: expected override.domain, got %q", cfg.Host)
	}
	// Secret should come from the first source
	if cfg.Secret != "base-secret" {
		t.Errorf("Secret: expected base-secret, got %q", cfg.Secret)
	}
}

func TestWithMap(t *testing.T) {
	var cfg ServerConfig
	err := goenv.Load(&cfg, goenv.WithMap(map[string]any{
		"HOST":   "10.0.0.1",
		"PORT":   5000,
		"SECRET": "map-secret",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "10.0.0.1" || cfg.Port != 5000 || cfg.Secret != "map-secret" {
		t.Errorf("WithMap mismatch: %+v", cfg)
	}
}

func TestStrictUnknownKeys(t *testing.T) {
	envContent := "PORT=8080\nSECRET=xyz\nUNKNOWN_TYPO=something\n"

	var cfg ServerConfig
	err := goenv.Load(&cfg,
		goenv.WithDotEnvReader(strings.NewReader(envContent)),
		goenv.WithStrictUnknown(true),
	)
	if err == nil {
		t.Fatal("expected strict error on unknown key, got nil")
	}

	var valErr *goenv.ValidationError
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
	goenv.MustLoad(&cfg, goenv.WithDotEnvReader(strings.NewReader("PORT=1234\n")))
}

func TestMissingDotEnvFile(t *testing.T) {
	var cfg ServerConfig
	// Default behavior should fail if file is specified and missing
	err := goenv.Load(&cfg, goenv.WithDotEnv("non_existent_file.env"))
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	// But WithIgnoreMissing(true) should not fail on the file opening
	err = goenv.Load(&cfg,
		goenv.WithIgnoreMissing(true),
		goenv.WithDotEnv("non_existent_file.env"),
	)
	// Still fails on validation because required Secret is missing, but NOT on file opening
	var valErr *goenv.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
