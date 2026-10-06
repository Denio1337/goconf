package store_test

import (
	"testing"

	"github.com/Denio1337/goconf/internal/store"
)

func TestStore_GetAndCaseInsensitive(t *testing.T) {
	s := store.New()
	s.Set("PORT", 8080)
	s.Set("app_name", "MyService")

	// Exact match
	val, ok := s.Get("PORT")
	if !ok || val != 8080 {
		t.Fatalf("expected PORT=8080, got %v (ok=%v)", val, ok)
	}

	// Lowercase candidate
	val, ok = s.Get("port")
	if !ok || val != 8080 {
		t.Fatalf("expected port=8080, got %v (ok=%v)", val, ok)
	}

	// Uppercase candidate for lowercase key
	val, ok = s.Get("APP_NAME")
	if !ok || val != "MyService" {
		t.Fatalf("expected APP_NAME=MyService, got %v (ok=%v)", val, ok)
	}
}

func TestStore_DottedAndUnderscoreKeys(t *testing.T) {
	s := store.New()
	s.Merge(map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	})

	// Dotted candidate
	val, ok := s.Get("database.host")
	if !ok || val != "localhost" {
		t.Fatalf("expected database.host=localhost, got %v", val)
	}

	// Underscored candidate
	val, ok = s.Get("DATABASE_HOST")
	if !ok || val != "localhost" {
		t.Fatalf("expected DATABASE_HOST=localhost, got %v", val)
	}

	// Double underscored candidate
	val, ok = s.Get("DATABASE__PORT")
	if !ok || val != 5432 {
		t.Fatalf("expected DATABASE__PORT=5432, got %v", val)
	}
}

func TestStore_Overrides(t *testing.T) {
	s := store.New()
	s.Merge(map[string]any{
		"PORT": 8000,
	})
	s.Merge(map[string]any{
		"port": 9000,
	})

	val, ok := s.Get("PORT")
	if !ok || val != 9000 {
		t.Fatalf("expected PORT=9000 after override, got %v", val)
	}
}

func TestStore_StrictKeys(t *testing.T) {
	s := store.New()
	s.Merge(map[string]any{
		"server": map[string]any{
			"port": 8080,
		},
	})

	strict := s.StrictKeys()
	if !strict["server.port"] {
		t.Errorf("expected server.port in strictKeys: %v", strict)
	}

	// Ambient merge should not track strict keys
	s.Merge(map[string]any{
		"ambient_key": "val",
	}, true)

	strict = s.StrictKeys()
	if strict["ambient_key"] {
		t.Errorf("did not expect ambient_key in strictKeys: %v", strict)
	}
}

func TestStore_AmbientKeysFiltering(t *testing.T) {
	s := store.New()
	s.Merge(map[string]any{
		"USER": "ambient_user",
	}, true)

	// Untagged without prefix -> should not match ambient key
	_, ok := s.Get("USER", false, "")
	if ok {
		t.Fatalf("expected untagged candidate without prefix to ignore ambient key")
	}

	// Tagged -> should match
	val, ok := s.Get("USER", true, "")
	if !ok || val != "ambient_user" {
		t.Fatalf("expected tagged candidate to match ambient key, got %v", val)
	}

	// Default call without optional args -> defaults to tagged, should match
	val, ok = s.Get("USER")
	if !ok || val != "ambient_user" {
		t.Fatalf("expected default Get to match ambient key, got %v", val)
	}

	// With prefix -> should match
	val, ok = s.Get("USER", false, "APP")
	if !ok || val != "ambient_user" {
		t.Fatalf("expected candidate with prefix to match ambient key, got %v", val)
	}

	// Non-existent key -> returns nil and false
	val, ok = s.Get("NON_EXISTENT")
	if ok || val != nil {
		t.Fatalf("expected non-existent key to return nil, false, got %v, %v", val, ok)
	}
}

func TestStore_DeepMergeMaps(t *testing.T) {
	s := store.New()

	// Source 1 (e.g. base config)
	s.Merge(map[string]any{
		"features": map[string]any{
			"cache":   true,
			"logging": false,
			"nested": map[string]any{
				"a": 1,
				"b": 2,
			},
		},
	})

	// Source 2 (e.g. override config)
	s.Merge(map[string]any{
		"features": map[string]any{
			"logging": true, // overridden
			"metrics": true, // added
			"nested": map[string]any{
				"b": 20, // overridden in nested
				"c": 30, // added in nested
			},
		},
	})

	rawMap, ok := s.Get("features")
	if !ok {
		t.Fatal("expected features to be found")
	}

	m, ok := rawMap.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", rawMap)
	}

	if m["cache"] != true {
		t.Errorf("expected cache=true (from base), got %v", m["cache"])
	}
	if m["logging"] != true {
		t.Errorf("expected logging=true (overridden), got %v", m["logging"])
	}
	if m["metrics"] != true {
		t.Errorf("expected metrics=true (from override), got %v", m["metrics"])
	}

	nested, ok := m["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map[string]any, got %T", m["nested"])
	}
	if nested["a"] != 1 {
		t.Errorf("expected nested.a=1, got %v", nested["a"])
	}
	if nested["b"] != 20 {
		t.Errorf("expected nested.b=20 (overridden), got %v", nested["b"])
	}
	if nested["c"] != 30 {
		t.Errorf("expected nested.c=30, got %v", nested["c"])
	}
}
