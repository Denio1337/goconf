package ini_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Denio1337/goconf/source/ini"
)

func TestINIParser(t *testing.T) {
	input := `
# Global configuration
app_name = "My Application"
environment = production

; Server section
[server]
host = 0.0.0.0
port = 8080 ; inline semicolon comment
timeout: 30s # inline hash comment

[database]
host = db.internal
port = 5432
user = "postgres"
password = 'secret#password'

[server.metrics]
enabled = true
path = /metrics
`

	src := ini.NewReader(strings.NewReader(input))
	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if data["app_name"] != "My Application" {
		t.Errorf("app_name: expected 'My Application', got %v", data["app_name"])
	}
	if data["environment"] != "production" {
		t.Errorf("environment: expected 'production', got %v", data["environment"])
	}

	server, ok := data["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server section to be map[string]any, got %T", data["server"])
	}

	if server["host"] != "0.0.0.0" {
		t.Errorf("server.host: expected '0.0.0.0', got %v", server["host"])
	}
	if server["port"] != "8080" {
		t.Errorf("server.port: expected '8080', got %v", server["port"])
	}
	if server["timeout"] != "30s" {
		t.Errorf("server.timeout: expected '30s', got %v", server["timeout"])
	}

	db, ok := data["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database section to be map[string]any, got %T", data["database"])
	}

	if db["password"] != "secret#password" {
		t.Errorf("database.password: expected 'secret#password', got %v", db["password"])
	}

	metrics, ok := server["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected server.metrics to be map[string]any, got %T", server["metrics"])
	}

	if metrics["enabled"] != "true" || metrics["path"] != "/metrics" {
		t.Errorf("server.metrics mismatch: %+v", metrics)
	}
}

func TestINIErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed section", "[server"},
		{"empty section", "[]"},
		{"missing delimiter", "just_a_word_without_delimiter"},
		{"unclosed quote", `key = "unclosed string`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := ini.NewReader(strings.NewReader(tc.input))
			_, err := src.Load(context.Background())
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.input)
			}
		})
	}
}
