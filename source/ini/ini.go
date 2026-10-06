// Package ini provides an INI file configuration Source.
package ini

import (
	"context"
	"io"

	"github.com/Denio1337/goconf/internal/sourceutil"
)

// Source loads configuration from an INI file or reader.
type Source struct {
	sourceutil.FileSource
	parser *Parser
}

// SourceOption configures the INI source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.SetIgnoreMissing(ignore)
	}
}

// New creates a new Source that reads from an INI file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewFileSource(path, "ini"),
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
		FileSource: sourceutil.NewReaderSource(r, "ini"),
		parser:     NewParser(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads and parses the INI configuration.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	return s.ReadAndParse(ctx, func(r io.Reader) (map[string]any, error) {
		return s.parser.Parse(r)
	})
}
