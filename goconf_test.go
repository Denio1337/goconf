package goconf_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Denio1337/goconf"
	"github.com/Denio1337/goconf/internal/store"
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

func TestLoaderInstance(t *testing.T) {
	loader := goconf.New(goconf.WithDotEnvReader(strings.NewReader("PORT=8081\nSECRET=mytoken\n")))
	var cfg ServerConfig
	if err := loader.Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8081 {
		t.Errorf("expected port 8081, got %d", cfg.Port)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected loader.MustLoad to panic")
		}
	}()
	invalidLoader := goconf.New(goconf.WithDotEnvReader(strings.NewReader("PORT=invalid\n")))
	var cfg2 ServerConfig
	invalidLoader.MustLoad(&cfg2)
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

func TestLoad_AlwaysAppendsEnv_WithJSON(t *testing.T) {
	os.Setenv("PORT", "9999")
	defer os.Unsetenv("PORT")

	jsonContent := `{"port": 8080, "host": "127.0.0.1"}`

	var cfg struct {
		Port int    `key:"PORT"`
		Host string `key:"HOST"`
	}

	// We only pass WithJSONReader, but env is always appended with highest priority
	err := goconf.Load(&cfg, goconf.WithJSONReader(strings.NewReader(jsonContent)))
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Port != 9999 {
		t.Errorf("Port: expected 9999 (from OS env override), got %d", cfg.Port)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host: expected 127.0.0.1 (from JSON), got %q", cfg.Host)
	}
}

func TestStrictUnknown_NoFalsePositivesFromAmbientEnv(t *testing.T) {
	var cfg ServerConfig
	// Valid .env content matching ServerConfig fields
	validContent := "PORT=8080\nSECRET=my-secret\nHOST=localhost\n"

	// WithStrictUnknown(true) should succeed without flagging ambient OS env vars (like PATH, SHELL)
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(validContent)),
		goconf.WithStrictUnknown(true),
	)
	if err != nil {
		t.Fatalf("expected nil error for valid config with strict unknown, got: %v", err)
	}
}

func TestLoadWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var cfg struct {
		Port int `key:"PORT"`
	}

	err := goconf.Load(&cfg,
		goconf.WithContext(ctx),
		goconf.WithDotEnvReader(strings.NewReader("PORT=8080\n")),
	)
	if err == nil {
		t.Fatal("expected error on canceled context, got nil")
	}

	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestWithoutAutoEnv(t *testing.T) {
	os.Setenv("GOCONF_TEST_PORT", "9999")
	defer os.Unsetenv("GOCONF_TEST_PORT")

	var cfg struct {
		Port int `key:"GOCONF_TEST_PORT"`
	}

	dotEnvContent := "GOCONF_TEST_PORT=8080\n"

	// 1. With WithoutAutoEnv(): OS env is ignored
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(dotEnvContent)),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port=8080 (OS env disabled), got %d", cfg.Port)
	}

	// 2. Default behavior (auto-env enabled): OS env overrides
	var defaultCfg struct {
		Port int `key:"GOCONF_TEST_PORT"`
	}
	err = goconf.Load(&defaultCfg,
		goconf.WithDotEnvReader(strings.NewReader(dotEnvContent)),
	)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if defaultCfg.Port != 9999 {
		t.Errorf("expected Port=9999 (OS env override), got %d", defaultCfg.Port)
	}
}

type minimalSource struct {
	data map[string]any
	err  error
}

func (m *minimalSource) Load(ctx context.Context) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func TestCustomMinimalSource(t *testing.T) {
	src := &minimalSource{
		data: map[string]any{
			"HOST": "custom-host",
			"PORT": 7777,
		},
	}

	var cfg struct {
		Host string `key:"HOST"`
		Port int    `key:"PORT"`
	}

	err := goconf.Load(&cfg, goconf.WithSource(src), goconf.WithoutAutoEnv())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Host != "custom-host" || cfg.Port != 7777 {
		t.Errorf("expected custom values, got host=%q, port=%d", cfg.Host, cfg.Port)
	}

	// Test error formatting with minimal source (no Name() method)
	failingSrc := &minimalSource{
		err: errors.New("connection failed"),
	}
	err = goconf.Load(&cfg, goconf.WithSource(failingSrc))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "*goconf_test.minimalSource") || !strings.Contains(err.Error(), "connection failed") {
		t.Errorf("expected formatted source type error, got: %v", err)
	}
}

func TestReviewFix1_AmbientEnvDoesNotOverwriteDefaultsWithoutTag(t *testing.T) {
	var cfg struct {
		User string `default:"default_user"`
		Home string `default:"/default/home"`
		Path string `default:"/default/path"`
	}

	err := goconf.Load(&cfg)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.User != "default_user" {
		t.Errorf("expected User to keep default, got: %q", cfg.User)
	}
	if cfg.Home != "/default/home" {
		t.Errorf("expected Home to keep default, got: %q", cfg.Home)
	}
	if cfg.Path != "/default/path" {
		t.Errorf("expected Path to keep default, got: %q", cfg.Path)
	}
}

func TestReviewFix2_PrefixScoping_NoFallbackToUnprefixed(t *testing.T) {
	type Config struct {
		Host  string `key:"HOST"`
		Redis struct {
			Host string `key:"HOST"`
		} `prefix:"REDIS_"`
	}

	dotEnvContent := "HOST=global.example.com\n"
	var cfg Config
	err := goconf.Load(&cfg, goconf.WithDotEnvReader(strings.NewReader(dotEnvContent)), goconf.WithoutAutoEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "global.example.com" {
		t.Errorf("expected global Host=global.example.com, got %q", cfg.Host)
	}
	if cfg.Redis.Host != "" {
		t.Errorf("expected Redis.Host to be empty (no fallback to root HOST), got %q", cfg.Redis.Host)
	}
}

func TestReviewFix3_StoreDeterminism(t *testing.T) {
	for i := 0; i < 200; i++ {
		st := store.New()
		st.Merge(map[string]any{
			"db.host": "host_dotted",
			"db_host": "host_underscore",
		})

		val1, _, ok1 := st.Get("db.host")
		val2, _, ok2 := st.Get("db_host")

		if !ok1 || val1 != "host_dotted" {
			t.Fatalf("iteration %d: expected db.host to be host_dotted, got %v", i, val1)
		}
		if !ok2 || val2 != "host_underscore" {
			t.Fatalf("iteration %d: expected db_host to be host_underscore, got %v", i, val2)
		}
	}
}

func TestReviewFix4_FloatOverflow_1e30(t *testing.T) {
	var cfg struct {
		Val int64 `key:"val"`
	}

	jsonContent := `{"val": 1e30}`
	err := goconf.Load(&cfg, goconf.WithJSONReader(strings.NewReader(jsonContent)), goconf.WithoutAutoEnv())
	if err == nil {
		t.Fatalf("expected overflow error for 1e30 into int64, got nil (Val=%d)", cfg.Val)
	}
	if !errors.Is(err, goconf.ErrTypeMismatch) {
		t.Errorf("expected ErrTypeMismatch, got: %v", err)
	}
}

func TestReviewFix5_SecretMaskingInErrorsAndSlog(t *testing.T) {
	type SecretConfig struct {
		SecretPort goconf.Secret[int] `key:"SECRET_PORT"`
	}

	sensitiveVal := "super-confidential-password-12345"
	var cfg SecretConfig
	err := goconf.Load(&cfg, goconf.WithDotEnvReader(strings.NewReader("SECRET_PORT="+sensitiveVal+"\n")), goconf.WithoutAutoEnv())
	if err == nil {
		t.Fatal("expected type error, got nil")
	}

	if strings.Contains(err.Error(), sensitiveVal) {
		t.Errorf("err.Error() leaked secret: %v", err)
	}

	formatted := fmt.Sprintf("%+v", err)
	if strings.Contains(formatted, sensitiveVal) {
		t.Errorf("%%+v format leaked secret: %s", formatted)
	}

	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	logger.Error("config failure", "error", err)
	if strings.Contains(logBuf.String(), sensitiveVal) {
		t.Errorf("slog output leaked secret: %s", logBuf.String())
	}
}

func TestReviewFix6_SliceAndMapOfStructs(t *testing.T) {
	type Server struct {
		Host string `key:"host"`
		Port int    `key:"port"`
	}

	type AppConfig struct {
		Clusters []Server          `key:"clusters"`
		Services map[string]Server `key:"services"`
	}

	jsonContent := `{
		"clusters": [
			{"host": "cluster1", "port": 8001},
			{"host": "cluster2", "port": 8002}
		],
		"services": {
			"auth": {"host": "auth.local", "port": 9001},
			"billing": {"host": "billing.local", "port": 9002}
		}
	}`

	var cfg AppConfig
	err := goconf.Load(&cfg, goconf.WithJSONReader(strings.NewReader(jsonContent)), goconf.WithoutAutoEnv())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if len(cfg.Clusters) != 2 || cfg.Clusters[0].Host != "cluster1" || cfg.Clusters[1].Port != 8002 {
		t.Errorf("unexpected Clusters: %+v", cfg.Clusters)
	}
	if len(cfg.Services) != 2 || cfg.Services["auth"].Host != "auth.local" || cfg.Services["billing"].Port != 9002 {
		t.Errorf("unexpected Services: %+v", cfg.Services)
	}
}

func TestReviewFix7_StrictUnknown_WithMapField(t *testing.T) {
	type Config struct {
		Labels map[string]string `key:"labels"`
	}

	jsonContent := `{"labels": {"env": "prod", "region": "eu-central"}}`
	var cfg Config
	err := goconf.Load(&cfg,
		goconf.WithJSONReader(strings.NewReader(jsonContent)),
		goconf.WithStrictUnknown(true),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("expected WithStrictUnknown to succeed on map field, got: %v", err)
	}
	if cfg.Labels["env"] != "prod" || cfg.Labels["region"] != "eu-central" {
		t.Errorf("unexpected labels: %+v", cfg.Labels)
	}
}

func TestReviewFix8_OptionalPointerSection(t *testing.T) {
	type DatabaseConfig struct {
		Host string `key:"HOST" required:"true"`
		Port int    `key:"PORT" required:"true"`
	}

	type Config struct {
		Database *DatabaseConfig `prefix:"DB_"`
		AppPort  int             `key:"APP_PORT"`
	}

	var cfg1 Config
	err := goconf.Load(&cfg1, goconf.WithDotEnvReader(strings.NewReader("APP_PORT=8080\n")), goconf.WithoutAutoEnv())
	if err != nil {
		t.Fatalf("expected optional database to remain nil, got error: %v", err)
	}
	if cfg1.Database != nil {
		t.Errorf("expected cfg1.Database to be nil, got: %+v", cfg1.Database)
	}
	if cfg1.AppPort != 8080 {
		t.Errorf("expected AppPort=8080, got %d", cfg1.AppPort)
	}

	var cfg2 Config
	err = goconf.Load(&cfg2, goconf.WithDotEnvReader(strings.NewReader("APP_PORT=8080\nDB_HOST=pg.local\nDB_PORT=5432\n")), goconf.WithoutAutoEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg2.Database == nil || cfg2.Database.Host != "pg.local" || cfg2.Database.Port != 5432 {
		t.Errorf("expected populated Database, got: %+v", cfg2.Database)
	}
}

func TestReviewFix9_LoaderReusableReader(t *testing.T) {
	jsonContent := `{"port": 9090}`
	loader := goconf.New(
		goconf.WithJSONReader(strings.NewReader(jsonContent)),
		goconf.WithoutAutoEnv(),
	)

	var cfg1 struct {
		Port int `key:"port"`
	}
	if err := loader.Load(&cfg1); err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if cfg1.Port != 9090 {
		t.Errorf("expected 9090, got %d", cfg1.Port)
	}

	var cfg2 struct {
		Port int `key:"port"`
	}
	if err := loader.Load(&cfg2); err != nil {
		t.Fatalf("second load failed with: %v", err)
	}
	if cfg2.Port != 9090 {
		t.Errorf("expected 9090, got %d", cfg2.Port)
	}
}

func TestReviewFix10_WithIgnoreMissing_OrderIndependence(t *testing.T) {
	var cfg struct {
		Port int `key:"PORT" default:"8080"`
	}

	// 1. WithIgnoreMissing placed AFTER file sources
	err := goconf.Load(&cfg,
		goconf.WithDotEnv("non_existent.env"),
		goconf.WithINI("non_existent.ini"),
		goconf.WithJSON("non_existent.json"),
		goconf.WithYAML("non_existent.yaml"),
		goconf.WithTOML("non_existent.toml"),
		goconf.WithIgnoreMissing(true),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("expected ignoreMissing to work when placed AFTER file sources, got: %v", err)
	}

	// 2. WithIgnoreMissing placed BEFORE file sources
	err = goconf.Load(&cfg,
		goconf.WithIgnoreMissing(true),
		goconf.WithDotEnv("non_existent.env"),
		goconf.WithINI("non_existent.ini"),
		goconf.WithJSON("non_existent.json"),
		goconf.WithYAML("non_existent.yaml"),
		goconf.WithTOML("non_existent.toml"),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("expected ignoreMissing to work when placed BEFORE file sources, got: %v", err)
	}
}

func TestReviewFix11_ExplicitEmptyStringPreserved(t *testing.T) {
	var cfg struct {
		Prefix string `key:"PREFIX" default:"default-prefix"`
	}

	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader("PREFIX=\n")),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Prefix != "" {
		t.Errorf("expected explicit empty string to be preserved, got %q", cfg.Prefix)
	}
}

func TestReviewFix12_ErrTypeMismatch_IsWrapped(t *testing.T) {
	var cfg struct {
		Port int `key:"PORT"`
	}

	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader("PORT=invalid_int\n")),
		goconf.WithoutAutoEnv(),
	)
	if err == nil {
		t.Fatal("expected type error, got nil")
	}

	if !errors.Is(err, goconf.ErrTypeMismatch) {
		t.Errorf("expected errors.Is(err, goconf.ErrTypeMismatch) to be true, got %v", err)
	}
}

func TestQuotedSliceElements(t *testing.T) {
	type Config struct {
		Origins []string `key:"ORIGINS"`
	}

	// 1. Array bracket syntax in .env
	var cfg1 Config
	content1 := `ORIGINS=["https://a.com, https://b.com", "https://c.com"]` + "\n"
	err := goconf.Load(&cfg1,
		goconf.WithDotEnvReader(strings.NewReader(content1)),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"https://a.com, https://b.com", "https://c.com"}
	if len(cfg1.Origins) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %v", len(expected), len(cfg1.Origins), cfg1.Origins)
	}
	for i, exp := range expected {
		if cfg1.Origins[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, cfg1.Origins[i])
		}
	}

	// 2. Single-quoted container in .env preserving inner double quotes
	var cfg2 Config
	content2 := `ORIGINS='"https://a.com, https://b.com", "https://c.com"'` + "\n"
	err = goconf.Load(&cfg2,
		goconf.WithDotEnvReader(strings.NewReader(content2)),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg2.Origins) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %v", len(expected), len(cfg2.Origins), cfg2.Origins)
	}
	for i, exp := range expected {
		if cfg2.Origins[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, cfg2.Origins[i])
		}
	}
}

func TestLargeConfigLineScanner(t *testing.T) {
	type Config struct {
		Cert string `key:"CERT"`
	}

	// 120 KB line (exceeds default 64 KB scanner limit)
	largeVal := strings.Repeat("A", 120*1024)
	content := "CERT=" + largeVal + "\n"

	var cfg Config
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(strings.NewReader(content)),
		goconf.WithoutAutoEnv(),
	)
	if err != nil {
		t.Fatalf("expected scanner to handle >64KB without ErrTooLong, got: %v", err)
	}
	if cfg.Cert != largeVal {
		t.Errorf("large value mismatch: length %d vs %d", len(cfg.Cert), len(largeVal))
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated io.Reader read error")
}

func TestReaderReadErrorHandling(t *testing.T) {
	type Config struct {
		Port int `key:"PORT"`
	}

	var cfg Config
	err := goconf.Load(&cfg,
		goconf.WithDotEnvReader(errReader{}),
		goconf.WithoutAutoEnv(),
	)
	if err == nil {
		t.Fatal("expected error on failed reader, got nil")
	}
	if !strings.Contains(err.Error(), "simulated io.Reader read error") {
		t.Errorf("expected simulated error message, got: %v", err)
	}
}

func TestConcurrentLoader(t *testing.T) {
	type Config struct {
		Port int `key:"PORT" default:"8080"`
	}

	loader := goconf.New(
		goconf.WithDotEnvReader(strings.NewReader("PORT=9090\n")),
		goconf.WithoutAutoEnv(),
	)

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var cfg Config
			if err := loader.Load(&cfg); err != nil {
				errCh <- err
				return
			}
			if cfg.Port != 9090 {
				errCh <- fmt.Errorf("unexpected port: %d", cfg.Port)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent load error: %v", err)
	}
}
