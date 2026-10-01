// Package ini provides an INI file configuration Source.
package ini

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Source loads configuration from an INI file or reader.
type Source struct {
	path          string
	reader        io.Reader
	ignoreMissing bool
	parser        *Parser
}

// SourceOption configures the INI source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.ignoreMissing = ignore
	}
}

// New creates a new Source that reads from an INI file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		path:   path,
		parser: NewParser(),
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
		parser: NewParser(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the descriptive name of the source.
func (s *Source) Name() string {
	if s.path != "" {
		return fmt.Sprintf("ini:%s", s.path)
	}
	return "ini:reader"
}

// Load reads and parses the INI configuration.
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
			return nil, fmt.Errorf("failed to open INI file %q: %w", s.path, err)
		}
		defer f.Close()
		r = f
	}

	data, err := s.parser.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", s.Name(), err)
	}

	return data, nil
}
