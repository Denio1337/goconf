package goconf

import (
	"errors"
	"net"
	"net/url"
	"testing"
	"time"
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
	MaxWorkers int `key:"MAX_WORKERS"`
}

func (v *ValidatedConfig) Validate() error {
	if v.MaxWorkers <= 0 {
		return errors.New("MAX_WORKERS must be greater than 0")
	}
	return nil
}

func TestDecoderKeyTagAndPrefix(t *testing.T) {
	type DatabaseConfig struct {
		Host        string `key:"HOST"`
		Port        int    `key:"PORT" default:"5432"`
		Username    string `key:"USER" default:"postgres"`
		Password    string `key:"PASSWORD"`
		GlobalToken string `key:"/GLOBAL_TOKEN"` // absolute key (ignores prefix)
	}

	type FlatSettings struct {
		WorkerCount int `key:"WORKERS" default:"4"`
	}

	type AppConfig struct {
		AppName      string         `key:"APP_NAME"`
		Port         int            `key:"PORT" default:"8080"`
		Debug        bool           `key:"DEBUG"`
		Rate         float64        `key:"RATE" default:"1.5"`
		Timeout      time.Duration  `key:"TIMEOUT" default:"5s"`
		CreatedAt    time.Time      `key:"CREATED_AT" layout:"2006-01-02"`
		Endpoint     *url.URL       `key:"ENDPOINT"`
		BindIP       net.IP         `key:"BIND_IP"`
		Tags         []string       `key:"TAGS"`
		AllowedPorts []int          `key:"PORTS" sep:";"`
		Settings     map[string]int `key:"SETTINGS"`
		OptionalVal  *int           `key:"OPTIONAL_VAL"`
		ServicePort  CustomPort     `key:"SERVICE_PORT"`
		Database     DatabaseConfig `prefix:"DB_"`
		FlatConfig   FlatSettings   `prefix:""` // explicitly empty prefix
	}

	st := NewStore()
	st.Merge(map[string]any{
		"APP_NAME":     "TestApp",
		"DEBUG":        "true",
		"TIMEOUT":      "15s",
		"CREATED_AT":   "2026-09-30",
		"ENDPOINT":     "https://api.example.com/v1",
		"BIND_IP":      "127.0.0.1",
		"TAGS":         "dev,backend,v2",
		"PORTS":        "8000;8080;9000",
		"SETTINGS":     "max_retries=3,buffer_size=1024",
		"OPTIONAL_VAL": "42",
		"SERVICE_PORT": "https",
		"DB_HOST":      "db.internal",
		"DB_PASSWORD":  "secret",
		"GLOBAL_TOKEN": "token-12345",
		"WORKERS":      "8", // matches FlatConfig because prefix is ""
	})

	var cfg AppConfig
	d := NewDecoder(st)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if cfg.AppName != "TestApp" {
		t.Errorf("AppName: expected TestApp, got %q", cfg.AppName)
	}
	if cfg.Port != 8080 {
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
	if cfg.CreatedAt.Year() != 2026 || cfg.CreatedAt.Month() != 9 || cfg.CreatedAt.Day() != 30 {
		t.Errorf("CreatedAt: expected 2026-09-30, got %v", cfg.CreatedAt)
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
	if cfg.Database.GlobalToken != "token-12345" {
		t.Errorf("Database.GlobalToken: expected token-12345, got %q", cfg.Database.GlobalToken)
	}
	if cfg.FlatConfig.WorkerCount != 8 {
		t.Errorf("FlatConfig.WorkerCount: expected 8, got %d", cfg.FlatConfig.WorkerCount)
	}
}

func TestPrefixChainingAndKeyPrefix(t *testing.T) {
	type ServerConfig struct {
		Host string `key:"HOST"`
		Port int    `key:"PORT" default:"80"`
	}

	type Config struct {
		// When key:"SERVER" is on struct, it acts as prefix "SERVER_"
		Server ServerConfig `key:"SERVER"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"SERVER_HOST": "api.domain",
		"SERVER_PORT": "8080",
	})

	var cfg Config
	d := NewDecoder(st)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Host != "api.domain" || cfg.Server.Port != 8080 {
		t.Errorf("unexpected server config: %+v", cfg.Server)
	}
}

func TestRootPrefix(t *testing.T) {
	type Config struct {
		Host string `key:"HOST"`
		Port int    `key:"PORT"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"MYAPP_HOST": "localhost",
		"MYAPP_PORT": "3000",
	})

	var cfg Config
	d := NewDecoder(st)
	d.SetPrefix("MYAPP_")
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" || cfg.Port != 3000 {
		t.Errorf("unexpected config with root prefix: %+v", cfg)
	}
}

func TestKeyTagInlineOptions(t *testing.T) {
	type Config struct {
		Host string `key:"HOST"`
		Port int    `key:"PORT,default=9000"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"HOST": "api.local",
	})

	var cfg Config
	d := NewDecoder(st)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "api.local" || cfg.Port != 9000 {
		t.Errorf("key tag inline options mismatch: %+v", cfg)
	}
}

func TestDecoderStrictValidationErrors(t *testing.T) {
	type ServerConfig struct {
		Port     int           `key:"PORT" required:"true"`
		Timeout  time.Duration `key:"TIMEOUT"`
		IsActive bool          `key:"ACTIVE"`
		Secret   string        `key:"SECRET" required:"true"`
		Ports    []int         `key:"PORTS"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"PORT":    "not-a-number",
		"TIMEOUT": "invalid-duration",
		"ACTIVE":  "not-a-boolean",
		"PORTS":   "10,twenty,30",
		// SECRET is intentionally missing
	})

	var cfg ServerConfig
	d := NewDecoder(st)
	err := d.Decode(&cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}

	if len(valErr.Errors) != 5 {
		t.Errorf("expected 5 errors, got %d:\n%v", len(valErr.Errors), valErr.Error())
	}
}

func TestDecoderCustomValidator(t *testing.T) {
	st := NewStore()
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
	st := NewStore()
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

func TestDecoderRawSliceAndMap(t *testing.T) {
	type Config struct {
		Tags   []string       `key:"TAGS"`
		Scores []int          `key:"SCORES"`
		Meta   map[string]int `key:"META"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"TAGS":   []any{"prod", "stable"},
		"SCORES": []int{10, 20, 30},
		"META": map[string]any{
			"version": 2,
			"retries": 5,
		},
	})

	var cfg Config
	d := NewDecoder(st)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if len(cfg.Tags) != 2 || cfg.Tags[0] != "prod" || cfg.Tags[1] != "stable" {
		t.Errorf("Tags mismatch: %v", cfg.Tags)
	}
	if len(cfg.Scores) != 3 || cfg.Scores[1] != 20 {
		t.Errorf("Scores mismatch: %v", cfg.Scores)
	}
	if cfg.Meta["version"] != 2 || cfg.Meta["retries"] != 5 {
		t.Errorf("Meta mismatch: %v", cfg.Meta)
	}
}

func TestDecoderDirectTypedValues(t *testing.T) {
	type TypedConfig struct {
		IntVal     int           `key:"INT_VAL"`
		Int64Val   int64         `key:"INT64_VAL"`
		UintVal    uint          `key:"UINT_VAL"`
		FloatVal   float64       `key:"FLOAT_VAL"`
		BoolVal    bool          `key:"BOOL_VAL"`
		Duration   time.Duration `key:"DURATION"`
		FloatAsInt int           `key:"FLOAT_AS_INT"`
	}

	st := NewStore()
	st.Merge(map[string]any{
		"INT_VAL":      42,
		"INT64_VAL":    int64(100500),
		"UINT_VAL":     uint(77),
		"FLOAT_VAL":    3.14159,
		"BOOL_VAL":     true,
		"DURATION":     5 * time.Minute,
		"FLOAT_AS_INT": float64(8080), // whole number float from JSON
	})

	var cfg TypedConfig
	d := NewDecoder(st)
	if err := d.Decode(&cfg); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if cfg.IntVal != 42 || cfg.Int64Val != 100500 || cfg.UintVal != 77 || cfg.FloatVal != 3.14159 || !cfg.BoolVal || cfg.Duration != 5*time.Minute || cfg.FloatAsInt != 8080 {
		t.Errorf("TypedConfig mismatch: %+v", cfg)
	}
}
