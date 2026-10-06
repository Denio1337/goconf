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
	rawKeys     map[string]string
	strictKeys  map[string]bool
	ambientKeys map[string]bool
}

// New initializes an empty configuration store.
func New() *Store {
	return &Store{
		values:      make(map[string]any),
		rawKeys:     make(map[string]string),
		strictKeys:  make(map[string]bool),
		ambientKeys: make(map[string]bool),
	}
}

// Merge merges entries from a source map into the store.
// If a key already exists, the new value overwrites the old one.
func (s *Store) Merge(src map[string]any) {
	s.MergeWithAmbient(src, true, false)
}

// MergeWithStrict merges entries from a source map into the store.
// If trackStrict is true, merged keys are recorded for strict unknown key validation.
func (s *Store) MergeWithStrict(src map[string]any, trackStrict bool) {
	s.MergeWithAmbient(src, trackStrict, false)
}

// MergeWithAmbient merges entries from a source map into the store, tracking ambient env keys.
func (s *Store) MergeWithAmbient(src map[string]any, trackStrict bool, isAmbient bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rawKeys == nil {
		s.rawKeys = make(map[string]string)
	}
	if s.ambientKeys == nil {
		s.ambientKeys = make(map[string]bool)
	}

	flattenAndMerge("", src, s.values, s.rawKeys, s.ambientKeys, isAmbient)

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
	lower := strings.ToLower(key)
	s.values[lower] = value
	s.rawKeys[lower] = key
}

// Get looks up a value by one or more candidate keys in order of precedence.
// Returns the first matching value found, the key that matched, and true.
// Returns (nil, "", false) if none match.
func (s *Store) Get(candidateKeys ...string) (any, string, bool) {
	return s.GetField(true, "", candidateKeys...)
}

func (s *Store) isKeyAllowed(key string, isTagged bool, prefix string) bool {
	// If key came from ambient OS env, only match if field was explicitly tagged or has a prefix
	return !s.ambientKeys[key] || isTagged || prefix != ""
}

// GetField looks up a value by candidate keys, with support for ignoring ambient OS env variables
// on untagged fields that have no prefix.
func (s *Store) GetField(isTagged bool, prefix string, candidateKeys ...string) (any, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Exact match pass
	for _, k := range candidateKeys {
		if k == "" {
			continue
		}
		lower := strings.ToLower(k)
		if v, ok := s.values[lower]; ok {
			if !s.isKeyAllowed(lower, isTagged, prefix) {
				continue
			}
			matched := s.rawKeys[lower]
			if matched == "" {
				matched = k
			}
			return v, matched, true
		}
	}

	// 2. Delimiter normalization fallback pass (dots <-> underscores)
	for _, k := range candidateKeys {
		if k == "" {
			continue
		}
		lower := strings.ToLower(k)

		if strings.Contains(lower, "__") {
			dotted := strings.ReplaceAll(lower, "__", ".")
			if v, ok := s.values[dotted]; ok {
				if !s.isKeyAllowed(dotted, isTagged, prefix) {
					continue
				}
				matched := s.rawKeys[dotted]
				if matched == "" {
					matched = k
				}
				return v, matched, true
			}
		}

		if strings.Contains(lower, "_") {
			dotted := strings.ReplaceAll(lower, "_", ".")
			if v, ok := s.values[dotted]; ok {
				if !s.isKeyAllowed(dotted, isTagged, prefix) {
					continue
				}
				matched := s.rawKeys[dotted]
				if matched == "" {
					matched = k
				}
				return v, matched, true
			}
		}

		if strings.Contains(lower, ".") {
			underscored := strings.ReplaceAll(lower, ".", "_")
			if v, ok := s.values[underscored]; ok {
				if !s.isKeyAllowed(underscored, isTagged, prefix) {
					continue
				}
				matched := s.rawKeys[underscored]
				if matched == "" {
					matched = k
				}
				return v, matched, true
			}

			doubleUnderscored := strings.ReplaceAll(lower, ".", "__")
			if v, ok := s.values[doubleUnderscored]; ok {
				if !s.isKeyAllowed(doubleUnderscored, isTagged, prefix) {
					continue
				}
				matched := s.rawKeys[doubleUnderscored]
				if matched == "" {
					matched = k
				}
				return v, matched, true
			}
		}
	}

	return nil, "", false
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

func flattenAndMerge(prefix string, current map[string]any, dest map[string]any, rawKeys map[string]string, ambientKeys map[string]bool, isAmbient bool) {
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
		if rawKeys != nil {
			rawKeys[lower] = fullKey
		}
		if isAmbient && ambientKeys != nil {
			ambientKeys[lower] = true
		}

		if subMap, ok := v.(map[string]any); ok {
			flattenAndMerge(fullKey, subMap, dest, rawKeys, ambientKeys, isAmbient)
		}
	}
}
