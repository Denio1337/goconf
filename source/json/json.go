// Package json provides a JSON file and reader configuration Source.
package json

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"os"
)

// Source loads configuration from a JSON file or an io.Reader.
type Source struct {
	path          string
	reader        io.Reader
	ignoreMissing bool
}

// SourceOption configures the JSON source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.ignoreMissing = ignore
	}
}

// New creates a new Source that reads from a JSON file path.
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
		return fmt.Sprintf("json:%s", s.path)
	}
	return "json:reader"
}

// Load reads and parses the JSON configuration data into a hierarchical map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var r io.Reader
	if s.reader != nil {
		r = s.reader
	} else {
		f, err := os.Open(s.path)
		if err != nil {
			if os.IsNotExist(err) && s.ignoreMissing {
				return make(map[string]any), nil
			}
			return nil, fmt.Errorf("failed to open JSON file %q: %w", s.path, err)
		}
		defer f.Close()
		r = f
	}

	dec := stdjson.NewDecoder(r)
	dec.UseNumber()

	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		if s.path != "" {
			return nil, fmt.Errorf("failed to parse JSON from %q: %w", s.path, err)
		}
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}
