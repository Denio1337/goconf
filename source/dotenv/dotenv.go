// Package dotenv provides a .env file and reader configuration Source.
package dotenv

import (
	"context"
	"io"

	"github.com/Denio1337/goconf/internal/sourceutil"
)

// Source loads configuration from a .env file or an io.Reader.
type Source struct {
	sourceutil.FileSource
	parser *Parser
}

// SourceOption configures the DotEnv source.
type SourceOption func(*Source)

// WithIgnoreMissing sets whether missing files should be ignored instead of returning an error.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.SetIgnoreMissing(ignore)
	}
}

// New creates a new Source that reads from a .env file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewFileSource(path, "dotenv"),
		parser:     NewParser(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Source that reads from an io.Reader.
func NewReader(r io.Reader, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewReaderSource(r, "dotenv"),
		parser:     NewParser(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads and parses the .env configuration.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	return s.ReadAndParse(ctx, func(r io.Reader) (map[string]any, error) {
		raw, err := s.parser.Parse(r)
		if err != nil {
			return nil, err
		}
		res := make(map[string]any, len(raw))
		for k, v := range raw {
			res[k] = v
		}
		return res, nil
	})
}
