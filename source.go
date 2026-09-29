package goenv

import (
	"context"
	"strings"
	"sync"
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

// Store holds the aggregated configuration data from all loaded sources.
type Store struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewStore initializes a new empty configuration store.
func NewStore() *Store {
	return &Store{
		values: make(map[string]any),
	}
}

// Merge merges entries from a source map into the store.
// If a key already exists, the new value overwrites the old one.
// Nested maps are flattened using dot notation and underscore notation.
func (s *Store) Merge(src map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flattenAndMerge("", src, s.values)
}

func flattenAndMerge(prefix string, current map[string]any, dest map[string]any) {
	for k, v := range current {
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = prefix + "." + k
		}

		if subMap, ok := v.(map[string]any); ok {
			flattenAndMerge(fullKey, subMap, dest)
		} else {
			dest[fullKey] = v

			// Also store normalized variants for easy lookup
			// e.g. "server.port" -> also "SERVER_PORT", "SERVER__PORT"
			underscored := strings.ReplaceAll(fullKey, ".", "_")
			doubleUnderscore := strings.ReplaceAll(fullKey, ".", "__")

			if _, exists := dest[underscored]; !exists {
				dest[underscored] = v
			}
			if _, exists := dest[doubleUnderscore]; !exists {
				dest[doubleUnderscore] = v
			}
		}
	}
}

// Get looks up a value by one or more candidate keys in order of precedence.
// Returns the first matching value found and true, or (nil, false) if none match.
func (s *Store) Get(candidateKeys ...string) (any, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, k := range candidateKeys {
		if k == "" {
			continue
		}
		// 1. Direct match
		if v, ok := s.values[k]; ok {
			return v, k, true
		}
		// 2. Uppercase match
		upper := strings.ToUpper(k)
		if v, ok := s.values[upper]; ok {
			return v, upper, true
		}
		// 3. Lowercase match
		lower := strings.ToLower(k)
		if v, ok := s.values[lower]; ok {
			return v, lower, true
		}
	}

	return nil, "", false
}

// All returns a copy of all flattened keys and values.
func (s *Store) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]any, len(s.values))
	for k, v := range s.values {
		res[k] = v
	}
	return res
}
