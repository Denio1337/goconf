package goenv

import (
	"context"

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

// MapSource is an in-memory Source backed by a Go map.
type MapSource struct {
	name string
	data map[string]any
}

// NewMapSource creates a new in-memory MapSource.
func NewMapSource(name string, data map[string]any) *MapSource {
	if name == "" {
		name = "memory"
	}
	cp := make(map[string]any, len(data))
	for k, v := range data {
		cp[k] = v
	}
	return &MapSource{
		name: name,
		data: cp,
	}
}

// Name returns the source name.
func (m *MapSource) Name() string {
	return m.name
}

// Load returns the copy of in-memory configuration map.
func (m *MapSource) Load(ctx context.Context) (map[string]any, error) {
	return m.data, nil
}

// Store is an alias for store.Store for backward compatibility and convenience.
type Store = store.Store

// NewStore initializes a new configuration store.
// Prefer using store.New() directly.
func NewStore(initial ...map[string]any) *Store {
	return store.New(initial...)
}
