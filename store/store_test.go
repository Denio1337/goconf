package store_test

import (
	"testing"

	"github.com/Denio1337/goenv/store"
)

func TestStoreNewAndMerge(t *testing.T) {
	s := store.New(map[string]any{
		"PORT": 8080,
	})

	s.Merge(map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	})

	val, key, ok := s.Get("PORT")
	if !ok || val != 8080 || key != "PORT" {
		t.Errorf("expected PORT=8080, got %v (key: %s, ok: %t)", val, key, ok)
	}

	val, key, ok = s.Get("DATABASE_HOST")
	if !ok || val != "localhost" {
		t.Errorf("expected DATABASE_HOST=localhost, got %v (key: %s, ok: %t)", val, key, ok)
	}

	val, key, ok = s.Get("database.port")
	if !ok || val != 5432 {
		t.Errorf("expected database.port=5432, got %v (key: %s, ok: %t)", val, key, ok)
	}

	if !s.Has("database.host") {
		t.Errorf("expected Has(database.host) to be true")
	}

	if s.Has("non_existent") {
		t.Errorf("expected Has(non_existent) to be false")
	}

	all := s.All()
	if len(all) == 0 {
		t.Errorf("expected non-empty map from All()")
	}
}

func TestStoreSet(t *testing.T) {
	s := store.New()
	s.Set("CUSTOM_KEY", "custom_val")

	val, _, ok := s.Get("CUSTOM_KEY")
	if !ok || val != "custom_val" {
		t.Errorf("expected CUSTOM_KEY=custom_val, got %v", val)
	}
}
