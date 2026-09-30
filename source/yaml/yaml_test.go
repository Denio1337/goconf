package yaml_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Denio1337/goconf/source/yaml"
)

func TestYAMLSource(t *testing.T) {
	input := `
app_name: TestService
port: 9090
debug: true
server:
  host: 127.0.0.1
  read_timeout: 15s
tags:
  - prod
  - v1
metadata:
  owner: denio
  retries: 3
`

	src := yaml.NewReader(strings.NewReader(input))
	if src.Name() != "yaml:reader" {
		t.Errorf("expected name 'yaml:reader', got %q", src.Name())
	}

	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if data["app_name"] != "TestService" {
		t.Errorf("app_name: expected 'TestService', got %v", data["app_name"])
	}
	if data["port"] != 9090 {
		t.Errorf("port: expected 9090, got %v", data["port"])
	}

	server, ok := data["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map[string]any, got %T", data["server"])
	}
	if server["host"] != "127.0.0.1" {
		t.Errorf("server.host: expected '127.0.0.1', got %v", server["host"])
	}

	tags, ok := data["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected tags to be []any with 2 elements, got %v", data["tags"])
	}
	if tags[0] != "prod" || tags[1] != "v1" {
		t.Errorf("tags mismatch: %v", tags)
	}
}

func TestYAMLEmpty(t *testing.T) {
	src := yaml.NewReader(strings.NewReader(""))
	data, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for empty YAML, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty map for empty YAML, got %v", data)
	}
}

func TestYAMLErrors(t *testing.T) {
	src := yaml.NewReader(strings.NewReader(":\n  invalid:\n    - yaml"))
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid yaml, got nil")
	}

	listSrc := yaml.NewReader(strings.NewReader("- item1\n- item2\n"))
	_, err = listSrc.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for non-mapping root YAML, got nil")
	}

	missingSrc := yaml.New("non_existent_file.yaml", yaml.WithIgnoreMissing(false))
	_, err = missingSrc.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	ignoredSrc := yaml.New("non_existent_file.yaml", yaml.WithIgnoreMissing(true))
	data, err := ignoredSrc.Load(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for ignored missing file, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data for missing file, got %v", data)
	}
}
