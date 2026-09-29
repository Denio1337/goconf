package mapsource_test

import (
	"context"
	"testing"

	"github.com/Denio1337/goenv/source/mapsource"
)

func TestMapSourceNew(t *testing.T) {
	data := map[string]any{
		"PORT": 8080,
		"HOST": "localhost",
	}

	src := mapsource.New(data)
	if src.Name() != "mapsource" {
		t.Errorf("expected name 'mapsource', got %q", src.Name())
	}

	loaded, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if loaded["PORT"] != 8080 || loaded["HOST"] != "localhost" {
		t.Errorf("loaded data mismatch: %+v", loaded)
	}

	// Verify isolation (shallow copy)
	loaded["PORT"] = 9999
	reloaded, _ := src.Load(context.Background())
	if reloaded["PORT"] != 8080 {
		t.Errorf("expected source to remain unchanged after modifying loaded map")
	}
}

func TestMapSourceNamed(t *testing.T) {
	src := mapsource.NewNamed("test-source", map[string]any{
		"ENV": "production",
	})

	if src.Name() != "test-source" {
		t.Errorf("expected name 'test-source', got %q", src.Name())
	}

	src.Set("DEBUG", true)
	val, ok := src.Get("DEBUG")
	if !ok || val != true {
		t.Errorf("expected Get(DEBUG)=true, got %v (%t)", val, ok)
	}
}
