// Package store provides an in-memory, thread-safe configuration store
// supporting case-insensitive key lookup, delimiter normalization, and strict key tracking.
package store

import (
	"maps"
	"slices"
	"strings"
	"sync"
)

// Store holds the aggregated configuration data from all loaded sources.
// It is safe for concurrent use by multiple goroutines, supporting hierarchical
// key-value flattening, case-insensitive lookups, and delimiter normalization.
type Store struct {
	mu          sync.RWMutex
	values      map[string]any
	strictKeys  map[string]bool
	ambientKeys map[string]bool
}

// New initializes an empty configuration store.
func New() *Store {
	return &Store{
		values:      make(map[string]any),
		strictKeys:  make(map[string]bool),
		ambientKeys: make(map[string]bool),
	}
}

// Merge merges entries from a source map into the store.
// If a key already exists, the new value overwrites the old one.
// The optional isAmbient flag indicates whether the source provides ambient/uncurated keys
// (ambient keys are recorded and excluded from strict unknown key validation).
func (s *Store) Merge(src map[string]any, isAmbient ...bool) {
	ambient := len(isAmbient) > 0 && isAmbient[0]
	trackStrict := !ambient

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ambientKeys == nil {
		s.ambientKeys = make(map[string]bool)
	}

	flattenAndMerge("", src, s.values, s.ambientKeys, ambient)

	if trackStrict {
		if s.strictKeys == nil {
			s.strictKeys = make(map[string]bool)
		}
		recordStrictKeys("", src, s.strictKeys)
	}
}

// Set explicitly sets a key and value in the store.
func (s *Store) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lower := strings.ToLower(key)
	s.values[lower] = value
}

// Get looks up a value by key in the store, supporting case-insensitive lookup
// and delimiter normalization (dots <-> underscores).
// The first argument key is the only required argument.
// The other two arguments (isTagged bool, prefix string) are optional and control ambient OS env filtering.
// Returns the matching value and true, or (nil, false) if not found.
func (s *Store) Get(key string, opts ...any) (any, bool) {
	if key == "" {
		return nil, false
	}

	isTagged := true
	prefix := ""
	for _, opt := range opts {
		switch v := opt.(type) {
		case bool:
			isTagged = v
		case string:
			prefix = v
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	lower := strings.ToLower(key)

	// 1. Exact match pass
	if v, ok := s.values[lower]; ok {
		if s.isKeyAllowed(lower, isTagged, prefix) {
			return v, true
		}
	}

	// 2. Delimiter normalization fallback pass (dots <-> underscores)
	if strings.Contains(lower, "__") {
		dotted := strings.ReplaceAll(lower, "__", ".")
		if v, ok := s.values[dotted]; ok {
			if s.isKeyAllowed(dotted, isTagged, prefix) {
				return v, true
			}
		}
	}

	if strings.Contains(lower, "_") {
		dotted := strings.ReplaceAll(lower, "_", ".")
		if v, ok := s.values[dotted]; ok {
			if s.isKeyAllowed(dotted, isTagged, prefix) {
				return v, true
			}
		}
	}

	if strings.Contains(lower, ".") {
		underscored := strings.ReplaceAll(lower, ".", "_")
		if v, ok := s.values[underscored]; ok {
			if s.isKeyAllowed(underscored, isTagged, prefix) {
				return v, true
			}
		}

		doubleUnderscored := strings.ReplaceAll(lower, ".", "__")
		if v, ok := s.values[doubleUnderscored]; ok {
			if s.isKeyAllowed(doubleUnderscored, isTagged, prefix) {
				return v, true
			}
		}
	}

	return nil, false
}

// HasPrefix checks if the store contains any keys beginning with the given prefix.
func (s *Store) HasPrefix(prefix string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if prefix == "" {
		return len(s.values) > 0
	}

	lower := strings.ToLower(prefix)
	clean := strings.TrimRight(lower, "_.")
	for k := range s.values {
		if strings.HasPrefix(k, lower) || strings.HasPrefix(k, clean+".") || strings.HasPrefix(k, clean+"_") {
			return true
		}
	}
	return false
}

// StrictKeys returns a set of keys tracked for strict unknown key validation.
// If no strict keys were explicitly tracked, it falls back to all stored keys.
func (s *Store) StrictKeys() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.strictKeys) == 0 {
		all := make(map[string]bool, len(s.values))
		for k := range s.values {
			all[k] = true
		}
		return all
	}

	res := make(map[string]bool, len(s.strictKeys))
	maps.Copy(res, s.strictKeys)
	return res
}

func (s *Store) isKeyAllowed(key string, isTagged bool, prefix string) bool {
	// If key came from ambient OS env, only match if field was explicitly tagged or has a prefix
	return !s.ambientKeys[key] || isTagged || prefix != ""
}

func recordStrictKeys(prefix string, current map[string]any, dest map[string]bool) {
	keys := make([]string, 0, len(current))
	for k := range current {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		v := current[k]
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = prefix + "." + k
		}
		dest[fullKey] = true
		if subMap, ok := v.(map[string]any); ok {
			recordStrictKeys(fullKey, subMap, dest)
		}
	}
}

func flattenAndMerge(prefix string, current map[string]any, dest map[string]any, ambientKeys map[string]bool, isAmbient bool) {
	keys := make([]string, 0, len(current))
	for k := range current {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		v := current[k]
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = prefix + "." + k
		}

		lower := strings.ToLower(fullKey)
		dest[lower] = v
		if isAmbient && ambientKeys != nil {
			ambientKeys[lower] = true
		}

		if subMap, ok := v.(map[string]any); ok {
			flattenAndMerge(fullKey, subMap, dest, ambientKeys, isAmbient)
		}
	}
}
