package json_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Denio1337/goconf/source/json"
)

func TestJSONSource(t *testing.T) {
	input := `{
		"app_name": "TestService",
		"port": 9090,
		"debug": true,
		"server": {
			"host": "127.0.0.1",
			"read_timeout": "15s"
		},
		"tags": ["prod", "v1"]
	}`

	src := json.NewReader(strings.NewReader(input))
	if src.Name() != "json:reader" {
		t.Errorf("expected name 'json:reader', got %q", src.Name())
	}

	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if data["app_name"] != "TestService" {
		t.Errorf("app_name: expected 'TestService', got %v", data["app_name"])
	}

	server, ok := data["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map[string]any, got %T", data["server"])
	}
	if server["host"] != "127.0.0.1" {
		t.Errorf("server.host: expected '127.0.0.1', got %v", server["host"])
	}
}

func TestJSONErrors(t *testing.T) {
	src := json.NewReader(strings.NewReader(`{invalid_json`))
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid json, got nil")
	}

	missingSrc := json.New("non_existent_file.json", json.WithIgnoreMissing(false))
	_, err = missingSrc.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	ignoredSrc := json.New("non_existent_file.json", json.WithIgnoreMissing(true))
	data, err := ignoredSrc.Load(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for ignored missing file, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data for missing file, got %v", data)
	}
}
