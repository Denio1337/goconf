// Package env provides an OS environment variable configuration Source.
package env

import (
	"context"
	"os"
	"strings"
)

// Option configures the environment variable Source.
type Option func(*Source)

// Source loads configuration from OS environment variables.
type Source struct {
	prefix      string
	stripPrefix bool
	environ     []string
}

// WithPrefix filters environment variables to only those that begin with prefix.
func WithPrefix(prefix string) Option {
	return func(s *Source) {
		s.prefix = prefix
	}
}

// WithStripPrefix controls whether the prefix should be trimmed from the resulting keys.
// When true (the default), variables like "APP_PORT" are stored as both "PORT" and "APP_PORT".
func WithStripPrefix(strip bool) Option {
	return func(s *Source) {
		s.stripPrefix = strip
	}
}

// WithEnviron sets a custom slice of environment variable strings (in "KEY=VALUE" form).
// If not specified or nil, os.Environ() is used. This is especially useful for testing.
func WithEnviron(environ []string) Option {
	return func(s *Source) {
		s.environ = environ
	}
}

// New creates a new environment variable configuration Source.
func New(opts ...Option) *Source {
	s := &Source{
		stripPrefix: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the human-readable identifier of this source.
func (s *Source) Name() string {
	if s.prefix != "" {
		return "env:prefix=" + s.prefix
	}
	return "env"
}

// Load reads and filters environment variables into a configuration map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	environ := s.environ
	if environ == nil {
		environ = os.Environ()
	}

	res := make(map[string]any)
	for _, envVar := range environ {
		k, v, ok := strings.Cut(envVar, "=")
		if !ok || k == "" {
			continue
		}

		if s.prefix != "" {
			if !strings.HasPrefix(k, s.prefix) {
				continue
			}
			if s.stripPrefix {
				stripped := strings.TrimPrefix(k, s.prefix)
				if stripped != "" {
					res[stripped] = v
					if strings.Contains(stripped, "__") {
						res[strings.ReplaceAll(stripped, "__", ".")] = v
					}
				}
			}
		}

		res[k] = v
		if strings.Contains(k, "__") {
			res[strings.ReplaceAll(k, "__", ".")] = v
		}
	}

	return res, nil
}
