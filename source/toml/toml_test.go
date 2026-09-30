package toml_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Denio1337/goconf/source/toml"
)

func TestTOMLSource(t *testing.T) {
	input := `
app_name = "TestService"
port = 9090
debug = true

[server]
host = "127.0.0.1"
read_timeout = "15s"

[tags]
values = ["prod", "v1"]
`

	src := toml.NewReader(strings.NewReader(input))
	if src.Name() != "toml:reader" {
		t.Errorf("expected name 'toml:reader', got %q", src.Name())
	}

	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if data["app_name"] != "TestService" {
		t.Errorf("app_name: expected 'TestService', got %v", data["app_name"])
	}
	if data["port"] != int64(9090) && data["port"] != 9090 {
		t.Errorf("port: expected 9090, got %v", data["port"])
	}

	server, ok := data["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map[string]any, got %T", data["server"])
	}
	if server["host"] != "127.0.0.1" {
		t.Errorf("server.host: expected '127.0.0.1', got %v", server["host"])
	}
}

func TestTOMLEmpty(t *testing.T) {
	src := toml.NewReader(strings.NewReader(""))
	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for empty TOML, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty map for empty TOML, got %v", data)
	}
}

func TestTOMLErrors(t *testing.T) {
	src := toml.NewReader(strings.NewReader("invalid toml === syntax"))
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid toml, got nil")
	}

	missingSrc := toml.New("non_existent_file.toml", toml.WithIgnoreMissing(false))
	_, err = missingSrc.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	ignoredSrc := toml.New("non_existent_file.toml", toml.WithIgnoreMissing(true))
	data, err := ignoredSrc.Load(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for ignored missing file, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data for missing file, got %v", data)
	}
}
