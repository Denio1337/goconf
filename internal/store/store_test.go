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
	val, matched, ok := s.Get("PORT")
	if !ok || val != 8080 || matched != "PORT" {
		t.Fatalf("expected PORT=8080, got %v (matched %q, ok=%v)", val, matched, ok)
	}

	// Lowercase candidate
	val, matched, ok = s.Get("port")
	if !ok || val != 8080 || matched != "PORT" {
		t.Fatalf("expected port=8080, got %v (matched %q, ok=%v)", val, matched, ok)
	}

	// Uppercase candidate for lowercase key
	val, matched, ok = s.Get("APP_NAME")
	if !ok || val != "MyService" || matched != "app_name" {
		t.Fatalf("expected APP_NAME=MyService, got %v (matched %q, ok=%v)", val, matched, ok)
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
	val, _, ok := s.Get("database.host")
	if !ok || val != "localhost" {
		t.Fatalf("expected database.host=localhost, got %v", val)
	}

	// Underscored candidate
	val, _, ok = s.Get("DATABASE_HOST")
	if !ok || val != "localhost" {
		t.Fatalf("expected DATABASE_HOST=localhost, got %v", val)
	}

	// Double underscored candidate
	val, _, ok = s.Get("DATABASE__PORT")
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

	val, _, ok := s.Get("PORT")
	if !ok || val != 9000 {
		t.Fatalf("expected PORT=9000 after override, got %v", val)
	}
}

func TestStore_StrictKeys(t *testing.T) {
	s := store.New()
	s.MergeWithStrict(map[string]any{
		"server": map[string]any{
			"port": 8080,
		},
	}, true)

	strict := s.StrictKeys()
	if !strict["server.port"] {
		t.Errorf("expected server.port in strictKeys: %v", strict)
	}
}
