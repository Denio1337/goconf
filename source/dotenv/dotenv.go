// Package dotenv provides a .env file and reader configuration Source.
package dotenv

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Source loads configuration from a .env file or an io.Reader.
type Source struct {
	path          string
	reader        io.Reader
	ignoreMissing bool
	parser        *Parser
}

// SourceOption configures the DotEnv source.
type SourceOption func(*Source)

// WithIgnoreMissing sets whether missing files should be ignored instead of returning an error.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.ignoreMissing = ignore
	}
}

// WithParserOptions passes parser options to the source's parser.
func WithParserOptions(opts ...Option) SourceOption {
	return func(s *Source) {
		s.parser = NewParser(opts...)
	}
}

// New creates a new Source that reads from a .env file path.
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

// Name returns the human-readable identifier of this source.
func (s *Source) Name() string {
	if s.path != "" {
		return fmt.Sprintf("dotenv:%s", s.path)
	}
	return "dotenv:reader"
}

// Load reads and parses the .env configuration.
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
			return nil, fmt.Errorf("failed to open dotenv file %q: %w", s.path, err)
		}
		defer f.Close()
		r = f
	}

	raw, err := s.parser.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", s.Name(), err)
	}

	res := make(map[string]any, len(raw))
	for k, v := range raw {
		res[k] = v
	}
	return res, nil
}
