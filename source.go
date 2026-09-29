package goenv

import (
	"context"

	"github.com/Denio1337/goenv/source/mapsource"
	"github.com/Denio1337/goenv/store"
)

// Source represents an abstraction for configuration data sources.
// Any format provider (dotenv, system environment variables, JSON, YAML, TOML, remote KV stores)
// implements this interface.
type Source interface {
	// Name returns a descriptive identifier for the source (e.g. "dotenv:.env", "os.env", "json:config.json").
	Name() string

	// Load reads and parses configuration data into a key-value or hierarchical map.
	Load(ctx context.Context) (map[string]any, error)
}

// MapSource is an alias for mapsource.Source for backward compatibility.
// Prefer using mapsource.New or mapsource.NewNamed directly.
type MapSource = mapsource.Source

// NewMapSource creates a new in-memory MapSource.
// Prefer using mapsource.New(data) or mapsource.NewNamed(name, data) directly.
func NewMapSource(name string, data map[string]any) *MapSource {
	return mapsource.NewNamed(name, data)
}

// Store is an alias for store.Store for backward compatibility and convenience.
type Store = store.Store

// NewStore initializes a new configuration store.
// Prefer using store.New() directly.
func NewStore(initial ...map[string]any) *Store {
	return store.New(initial...)
}
