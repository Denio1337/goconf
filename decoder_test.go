package goconf

import (
	"errors"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/Denio1337/goconf/store"
)

type CustomPort int

func (c *CustomPort) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "http" {
		*c = 80
		return nil
	}
	if s == "https" {
		*c = 443
		return nil
	}
	return errors.New("unknown service port")
}

type ValidatedConfig struct {
	MaxWorkers int `env:"MAX_WORKERS"`
}

func (v *ValidatedConfig) Validate() error {
	if v.MaxWorkers <= 0 {
		return errors.New("MAX_WORKERS must be greater than 0")
	}
	return nil
}

func TestDecoderSuccess(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"HOST"`
		Port     int    `env:"PORT" default:"5432"`
		Username string `env:"USER" default:"postgres"`
		Password string `env:"PASSWORD"`
	}

	type AppConfig struct {
		AppName      string         `env:"APP_NAME"`
		Port         int            `env:"PORT" default:"8080"`
		Debug        bool           `env:"DEBUG"`
		Rate         float64        `env:"RATE" default:"1.5"`
		Timeout      time.Duration  `env:"TIMEOUT" default:"5s"`
		CreatedAt    time.Time      `env:"CREATED_AT" layout:"2006-01-02"`
		Endpoint     *url.URL       `env:"ENDPOINT"`
		BindIP       net.IP         `env:"BIND_IP"`
		Tags         []string       `env:"TAGS"`
		AllowedPorts []int          `env:"PORTS" sep:";"`
		Settings     map[string]int `env:"SETTINGS"`
		OptionalVal  *int           `env:"OPTIONAL_VAL"`
		ServicePort  CustomPort     `env:"SERVICE_PORT"`
		Database     DatabaseConfig `env-prefix:"DB_"`
	}

	cfgStore := store.New()
	cfgStore.Merge(map[string]any{
		"APP_NAME":     "TestApp",
		"DEBUG":        "true",
		"TIMEOUT":      "15s",
		"CREATED_AT":   "2026-09-29",
		"ENDPOINT":     "https://api.example.com/v1",
		"BIND_IP":      "127.0.0.1",
		"TAGS":         "dev,backend,v2",
		"PORTS":        "8000;8080;9000",
		"SETTINGS":     "max_retries=3,buffer_size=1024",
		"OPTIONAL_VAL": "42",
		"SERVICE_PORT": "https",
		"DB_HOST":      "db.internal",
		"DB_PASSWORD":  "secret",
	})

	var cfg AppConfig
	d := NewDecoder(cfgStore)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if cfg.AppName != "TestApp" {
		t.Errorf("AppName: expected TestApp, got %q", cfg.AppName)
	}
	if cfg.Port != 8080 { // from default
		t.Errorf("Port (default): expected 8080, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Errorf("Debug: expected true, got false")
	}
	if cfg.Rate != 1.5 {
		t.Errorf("Rate: expected 1.5, got %v", cfg.Rate)
	}
	if cfg.Timeout != 15*time.Second {
		t.Errorf("Timeout: expected 15s, got %v", cfg.Timeout)
	}
	if cfg.CreatedAt.Year() != 2026 || cfg.CreatedAt.Month() != 9 || cfg.CreatedAt.Day() != 29 {
		t.Errorf("CreatedAt: expected 2026-09-29, got %v", cfg.CreatedAt)
	}
	if cfg.Endpoint == nil || cfg.Endpoint.Host != "api.example.com" {
		t.Errorf("Endpoint: expected api.example.com, got %v", cfg.Endpoint)
	}
	if cfg.BindIP.String() != "127.0.0.1" {
		t.Errorf("BindIP: expected 127.0.0.1, got %v", cfg.BindIP)
	}
	if len(cfg.Tags) != 3 || cfg.Tags[0] != "dev" || cfg.Tags[1] != "backend" || cfg.Tags[2] != "v2" {
		t.Errorf("Tags: expected [dev, backend, v2], got %v", cfg.Tags)
	}
	if len(cfg.AllowedPorts) != 3 || cfg.AllowedPorts[1] != 8080 {
		t.Errorf("AllowedPorts: expected [8000, 8080, 9000], got %v", cfg.AllowedPorts)
	}
	if cfg.Settings["max_retries"] != 3 || cfg.Settings["buffer_size"] != 1024 {
		t.Errorf("Settings: expected {max_retries:3, buffer_size:1024}, got %v", cfg.Settings)
	}
	if cfg.OptionalVal == nil || *cfg.OptionalVal != 42 {
		t.Errorf("OptionalVal: expected *42, got %v", cfg.OptionalVal)
	}
	if cfg.ServicePort != 443 {
		t.Errorf("ServicePort (TextUnmarshaler): expected 443, got %d", cfg.ServicePort)
	}
	if cfg.Database.Host != "db.internal" || cfg.Database.Port != 5432 || cfg.Database.Username != "postgres" || cfg.Database.Password != "secret" {
		t.Errorf("Database config mismatch: %+v", cfg.Database)
	}
}

func TestDecoderStrictValidationErrors(t *testing.T) {
	type ServerConfig struct {
		Port     int           `env:"PORT" required:"true"`
		Timeout  time.Duration `env:"TIMEOUT"`
		IsActive bool          `env:"ACTIVE"`
		Secret   string        `env:"SECRET" required:"true"`
		Ports    []int         `env:"PORTS"`
	}

	cfgStore := store.New()
	cfgStore.Merge(map[string]any{
		"PORT":    "not-a-number",
		"TIMEOUT": "invalid-duration",
		"ACTIVE":  "not-a-boolean",
		"PORTS":   "10,twenty,30",
		// SECRET is intentionally missing
	})

	var cfg ServerConfig
	d := NewDecoder(cfgStore)
	err := d.Decode(&cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}

	// Should report 5 errors: PORT, TIMEOUT, ACTIVE, PORTS[1], SECRET
	if len(valErr.Errors) != 5 {
		t.Errorf("expected 5 errors, got %d:\n%v", len(valErr.Errors), valErr.Error())
	}

	errStr := valErr.Error()
	t.Logf("Reported formatted error:\n%s", errStr)

	// Verify that specific field errors are present
	fieldErrorsMap := make(map[string]FieldError)
	for _, fe := range valErr.Errors {
		fieldErrorsMap[fe.Field] = fe
	}

	if fe, ok := fieldErrorsMap["Port"]; !ok {
		t.Errorf("expected error for 'Port'")
	} else if fe.Key != "PORT" || fe.TargetType != "int" {
		t.Errorf("Port FieldError mismatch: %+v", fe)
	}

	if _, ok := fieldErrorsMap["Timeout"]; !ok {
		t.Errorf("expected error for 'Timeout'")
	}

	if _, ok := fieldErrorsMap["IsActive"]; !ok {
		t.Errorf("expected error for 'IsActive'")
	}

	if _, ok := fieldErrorsMap["Secret"]; !ok {
		t.Errorf("expected error for 'Secret'")
	} else if !errors.Is(valErr, ErrMissingRequired) {
		t.Errorf("expected errors.Is(valErr, ErrMissingRequired) to be true")
	}

	if fe, ok := fieldErrorsMap["Ports"]; !ok {
		t.Errorf("expected error for 'Ports'")
	} else if fe.Value != "10,twenty,30" {
		t.Errorf("Ports error value mismatch: %v", fe.Value)
	}
}

func TestDecoderCustomValidator(t *testing.T) {
	st := store.New()
	st.Merge(map[string]any{
		"MAX_WORKERS": "-5",
	})

	var cfg ValidatedConfig
	d := NewDecoder(st)
	err := d.Decode(&cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got %v", err)
	}
}

func TestDecoderInvalidTarget(t *testing.T) {
	st := store.New()
	d := NewDecoder(st)

	if err := d.Decode(nil); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("expected ErrInvalidTarget on nil, got %v", err)
	}

	var notPtr struct{}
	if err := d.Decode(notPtr); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("expected ErrInvalidTarget on non-pointer, got %v", err)
	}

	var notStruct int
	if err := d.Decode(&notStruct); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("expected ErrInvalidTarget on pointer to int, got %v", err)
	}
}
