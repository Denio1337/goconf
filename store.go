// Package store provides an in-memory, thread-safe configuration store.
//
// It supports hierarchical key-value flattening, case-insensitive lookups,
// and delimiter normalization (dot, underscore, double-underscore).
package goconf

import (
	"maps"
	"strings"
	"sync"
)

// Store holds the aggregated configuration data from all loaded sources.
// It is safe for concurrent use by multiple goroutines.
type Store struct {
	mu     sync.RWMutex
	values map[string]any
}

// New initializes a new configuration store, optionally populated with initial key-value maps.
func NewStore(initial ...map[string]any) *Store {
	s := &Store{
		values: make(map[string]any),
	}
	for _, m := range initial {
		s.Merge(m)
	}
	return s
}

// Merge merges entries from a source map into the store.
// If a key already exists, the new value overwrites the old one.
// Nested maps are recursively flattened using dot notation and underscore conventions.
func (s *Store) Merge(src map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flattenAndMerge("", src, s.values)
}

// Set explicitly sets a key and value in the store.
func (s *Store) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	setVariants(s.values, key, value)
}

// Get looks up a value by one or more candidate keys in order of precedence.
// Returns the first matching value found, the key that matched, and true.
// Returns (nil, "", false) if none match.
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

// Has checks if any of the candidate keys exist in the store.
func (s *Store) Has(candidateKeys ...string) bool {
	_, _, ok := s.Get(candidateKeys...)
	return ok
}

// All returns a shallow copy of all stored keys and values.
func (s *Store) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]any, len(s.values))
	maps.Copy(res, s.values)
	return res
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
			setVariants(dest, fullKey, v)
			flattenAndMerge(fullKey, subMap, dest)
		} else {
			setVariants(dest, fullKey, v)
		}
	}
}

func setVariants(dest map[string]any, key string, v any) {
	dest[key] = v
	dest[strings.ToUpper(key)] = v
	dest[strings.ToLower(key)] = v

	underscored := strings.ReplaceAll(key, ".", "_")
	dest[underscored] = v
	dest[strings.ToUpper(underscored)] = v
	dest[strings.ToLower(underscored)] = v

	doubleUnderscore := strings.ReplaceAll(key, ".", "__")
	dest[doubleUnderscore] = v
	dest[strings.ToUpper(doubleUnderscore)] = v
	dest[strings.ToLower(doubleUnderscore)] = v
}
