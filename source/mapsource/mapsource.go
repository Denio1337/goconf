// Package mapsource provides an in-memory configuration Source backed by a Go map.
package mapsource

import (
	"context"
)

// Option is a functional option for configuring the map source.
type Option func(*Source)

// WithName sets a custom name for the Source identifier.
func WithName(name string) Option {
	return func(s *Source) {
		if name != "" {
			s.name = name
		}
	}
}

// Source represents an in-memory configuration source backed by a Go map.
type Source struct {
	name string
	data map[string]any
}

// New creates a new in-memory Source populated with the provided data.
// By default, its Name() is "mapsource".
func New(data map[string]any, opts ...Option) *Source {
	s := &Source{
		name: "mapsource",
		data: make(map[string]any, len(data)),
	}
	for k, v := range data {
		s.data[k] = v
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewNamed creates a new Source with an explicit name and data.
func NewNamed(name string, data map[string]any) *Source {
	if name == "" {
		name = "mapsource"
	}
	return New(data, WithName(name))
}

// Name returns the descriptive name of the source.
func (s *Source) Name() string {
	return s.name
}

// Load returns a shallow copy of the configuration map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	cp := make(map[string]any, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp, nil
}

// Set sets or updates a key in the source.
func (s *Source) Set(key string, val any) {
	s.data[key] = val
}

// Get retrieves a key from the source.
func (s *Source) Get(key string) (any, bool) {
	val, ok := s.data[key]
	return val, ok
}
