package goconf

import (
	"maps"
	"strings"
	"sync"
)

// Store holds the aggregated configuration data from all loaded sources.
// It is safe for concurrent use by multiple goroutines, supporting hierarchical
// key-value flattening, case-insensitive lookups, and delimiter normalization.
type Store struct {
	mu         sync.RWMutex
	values     map[string]any
	rawKeys    map[string]string
	strictKeys map[string]bool
}

// NewStore initializes a new configuration store, optionally populated with initial key-value maps.
func NewStore(initial ...map[string]any) *Store {
	s := &Store{
		values:     make(map[string]any),
		rawKeys:    make(map[string]string),
		strictKeys: make(map[string]bool),
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
	s.MergeWithStrict(src, true)
}

// MergeWithStrict merges entries from a source map into the store.
// If trackStrict is true, merged keys are recorded for strict unknown key validation.
func (s *Store) MergeWithStrict(src map[string]any, trackStrict bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rawKeys == nil {
		s.rawKeys = make(map[string]string)
	}
	flattenAndMerge("", src, s.values, s.rawKeys)

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

	if s.rawKeys == nil {
		s.rawKeys = make(map[string]string)
	}
	setVariants(s.values, s.rawKeys, key, value)
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
		lower := strings.ToLower(k)
		if v, ok := s.values[lower]; ok {
			matched := s.rawKeys[lower]
			if matched == "" {
				matched = k
			}
			return v, matched, true
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
	for k, v := range s.values {
		orig := s.rawKeys[k]
		if orig == "" {
			orig = k
		}
		res[orig] = v
	}
	return res
}

// StrictKeys returns a set of keys tracked for strict unknown key validation.
// If no strict keys were explicitly tracked, it falls back to All() keys.
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

func recordStrictKeys(prefix string, current map[string]any, dest map[string]bool) {
	for k, v := range current {
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

func flattenAndMerge(prefix string, current map[string]any, dest map[string]any, rawKeys map[string]string) {
	for k, v := range current {
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = prefix + "." + k
		}

		if subMap, ok := v.(map[string]any); ok {
			setVariants(dest, rawKeys, fullKey, v)
			flattenAndMerge(fullKey, subMap, dest, rawKeys)
		} else {
			setVariants(dest, rawKeys, fullKey, v)
		}
	}
}

func setVariants(dest map[string]any, rawKeys map[string]string, key string, v any) {
	lower := strings.ToLower(key)
	dest[lower] = v
	if rawKeys != nil {
		rawKeys[lower] = key
	}

	if strings.Contains(lower, ".") {
		underscored := strings.ReplaceAll(lower, ".", "_")
		dest[underscored] = v
		if rawKeys != nil {
			rawKeys[underscored] = key
		}

		doubleUnderscored := strings.ReplaceAll(lower, ".", "__")
		dest[doubleUnderscored] = v
		if rawKeys != nil {
			rawKeys[doubleUnderscored] = key
		}
	}
}
