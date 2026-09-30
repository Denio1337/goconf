// Package toml provides a TOML file and reader configuration Source.
package toml

import (
	"context"
	"fmt"
	"io"
	"os"

	stdtoml "github.com/pelletier/go-toml/v2"
)

// Source loads configuration from a TOML file or an io.Reader.
type Source struct {
	path          string
	reader        io.Reader
	ignoreMissing bool
}

// SourceOption configures the TOML source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.ignoreMissing = ignore
	}
}

// New creates a new Source that reads from a TOML file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		path: path,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Source that reads from an io.Reader.
func NewReader(r io.Reader, opts ...SourceOption) *Source {
	s := &Source{
		reader: r,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the human-readable identifier of this source.
func (s *Source) Name() string {
	if s.path != "" {
		return fmt.Sprintf("toml:%s", s.path)
	}
	return "toml:reader"
}

// Load reads and parses the TOML configuration data into a hierarchical map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	var r io.Reader
	if s.reader != nil {
		r = s.reader
	} else {
		f, err := os.Open(s.path)
		if err != nil {
			if os.IsNotExist(err) && s.ignoreMissing {
				return make(map[string]any), nil
			}
			return nil, fmt.Errorf("failed to open TOML file %q: %w", s.path, err)
		}
		defer f.Close()
		r = f
	}

	var data map[string]any
	dec := stdtoml.NewDecoder(r)
	if err := dec.Decode(&data); err != nil {
		if err == io.EOF {
			return make(map[string]any), nil
		}
		if s.path != "" {
			return nil, fmt.Errorf("failed to parse TOML from %q: %w", s.path, err)
		}
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	if data == nil {
		return make(map[string]any), nil
	}

	return data, nil
}
