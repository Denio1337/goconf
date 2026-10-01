package goconf_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Denio1337/goconf"
)

type SecretTestConfig struct {
	Password goconf.Secret[string]        `key:"PASSWORD"`
	APIKey   goconf.Secret[string]        `key:"API_KEY" default:"default-key"`
	DBPort   goconf.Secret[int]           `key:"DB_PORT"`
	Timeout  goconf.Secret[time.Duration] `key:"TIMEOUT"`
	Tokens   goconf.Secret[[]string]      `key:"TOKENS"`
	PtrToken *goconf.Secret[string]       `key:"PTR_TOKEN"`
}

func TestSecretFormatting(t *testing.T) {
	sec := goconf.NewSecret("super-secret-password-123")

	// Raw values
	if sec.Value() != "super-secret-password-123" {
		t.Errorf("expected raw value 'super-secret-password-123', got %q", sec.Value())
	}

	// Stringer / fmt
	if got := fmt.Sprint(sec); got != "[SECRET]" {
		t.Errorf("fmt.Sprint: expected '[SECRET]', got %q", got)
	}
	if got := fmt.Sprintf("%v", sec); got != "[SECRET]" {
		t.Errorf("%%v: expected '[SECRET]', got %q", got)
	}
	if got := fmt.Sprintf("%+v", sec); got != "[SECRET]" {
		t.Errorf("%%+v: expected '[SECRET]', got %q", got)
	}
	if got := fmt.Sprintf("%#v", sec); got != "[SECRET]" {
		t.Errorf("%%#v: expected '[SECRET]', got %q", got)
	}
	if got := fmt.Sprintf("%s", sec); got != "[SECRET]" {
		t.Errorf("%%s: expected '[SECRET]', got %q", got)
	}
	if got := fmt.Sprintf("%q", sec); got != `"[SECRET]"` {
		t.Errorf("%%q: expected '\"[SECRET]\"', got %q", got)
	}

	// JSON marshaling
	b, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if string(b) != `"[SECRET]"` {
		t.Errorf("json.Marshal: expected '\"[SECRET]\"', got %s", string(b))
	}
}

func TestSecretDecoding(t *testing.T) {
	dotenvContent := `
PASSWORD=my-db-secret
DB_PORT=5432
TIMEOUT=30s
TOKENS=tok1,tok2,tok3
PTR_TOKEN=pointer-secret
`
	var cfg SecretTestConfig
	err := goconf.Load(&cfg, goconf.WithDotEnvReader(strings.NewReader(dotenvContent)))
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	// Verify raw values decoded properly
	if cfg.Password.Value() != "my-db-secret" {
		t.Errorf("Password: expected 'my-db-secret', got %q", cfg.Password.Value())
	}
	if cfg.APIKey.Value() != "default-key" {
		t.Errorf("APIKey (default): expected 'default-key', got %q", cfg.APIKey.Value())
	}
	if cfg.DBPort.Value() != 5432 {
		t.Errorf("DBPort: expected 5432, got %d", cfg.DBPort.Value())
	}
	if cfg.Timeout.Value() != 30*time.Second {
		t.Errorf("Timeout: expected 30s, got %v", cfg.Timeout.Value())
	}
	if len(cfg.Tokens.Value()) != 3 || cfg.Tokens.Value()[1] != "tok2" {
		t.Errorf("Tokens: expected [tok1 tok2 tok3], got %v", cfg.Tokens.Value())
	}
	if cfg.PtrToken == nil || cfg.PtrToken.Value() != "pointer-secret" {
		t.Errorf("PtrToken: expected pointer-secret, got %v", cfg.PtrToken)
	}

	// Verify that printing the entire struct NEVER reveals any secret
	printedStruct := fmt.Sprintf("%+v", cfg)
	if strings.Contains(printedStruct, "my-db-secret") {
		t.Errorf("printed struct leaked Password: %s", printedStruct)
	}
	if strings.Contains(printedStruct, "default-key") {
		t.Errorf("printed struct leaked APIKey: %s", printedStruct)
	}
	if strings.Contains(printedStruct, "pointer-secret") {
		t.Errorf("printed struct leaked PtrToken: %s", printedStruct)
	}

	// Verify all secret fields are masked as [SECRET]
	if !strings.Contains(printedStruct, "Password:[SECRET]") {
		t.Errorf("printed struct expected Password:[SECRET], got: %s", printedStruct)
	}
	if !strings.Contains(printedStruct, "APIKey:[SECRET]") {
		t.Errorf("printed struct expected APIKey:[SECRET], got: %s", printedStruct)
	}
	if !strings.Contains(printedStruct, "DBPort:[SECRET]") {
		t.Errorf("printed struct expected DBPort:[SECRET], got: %s", printedStruct)
	}

	// Verify JSON marshaling of struct masks secrets
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(cfg) failed: %v", err)
	}
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, "my-db-secret") || strings.Contains(jsonStr, "default-key") {
		t.Errorf("json.Marshal leaked secret: %s", jsonStr)
	}
}

func TestSecretRequiredValidation(t *testing.T) {
	type RequiredConfig struct {
		SecretKey goconf.Secret[string] `key:"SECRET_KEY" required:"true"`
	}

	var cfg RequiredConfig
	err := goconf.Load(&cfg, goconf.WithDotEnvReader(strings.NewReader("OTHER=123\n")))
	if err == nil {
		t.Fatal("expected validation error for missing required secret, got nil")
	}

	if !errors.Is(err, goconf.ErrMissingRequired) {
		t.Errorf("expected ErrMissingRequired, got: %v", err)
	}
}

func TestSecretUnmaskAndMarshalTextAndSlog(t *testing.T) {
	sec := goconf.NewSecret("p@ssw0rd")

	// Test Unmask
	if sec.Unmask() != "p@ssw0rd" {
		t.Errorf("expected Unmask() to return p@ssw0rd, got %q", sec.Unmask())
	}

	// Test MarshalText
	text, err := sec.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error: %v", err)
	}
	if string(text) != "[SECRET]" {
		t.Errorf("MarshalText: expected [SECRET], got %s", string(text))
	}

	// Test slog.LogValuer
	val := sec.LogValue()
	if val.String() != "[SECRET]" {
		t.Errorf("LogValue: expected [SECRET], got %v", val)
	}
}

func TestSecretFieldErrorMasking(t *testing.T) {
	type BadSecretConfig struct {
		SecretPort goconf.Secret[int] `key:"SECRET_PORT"`
	}

	var cfg BadSecretConfig
	err := goconf.Load(&cfg, goconf.WithDotEnvReader(strings.NewReader("SECRET_PORT=not-an-integer-sensitive-token\n")))
	if err == nil {
		t.Fatal("expected type conversion error, got nil")
	}

	errStr := err.Error()
	if strings.Contains(errStr, "not-an-integer-sensitive-token") {
		t.Errorf("FieldError leaked secret value in error message: %s", errStr)
	}
	if !strings.Contains(errStr, `"[SECRET]"`) && !strings.Contains(errStr, "[SECRET]") {
		t.Errorf("FieldError should have masked secret value, got: %s", errStr)
	}
}

