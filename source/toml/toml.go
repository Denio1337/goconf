// Package toml provides a TOML file and reader configuration Source.
package toml

import (
	"context"
	"io"

	"github.com/Denio1337/goconf/internal/sourceutil"
	stdtoml "github.com/pelletier/go-toml/v2"
)

// Source loads configuration from a TOML file or an io.Reader.
type Source struct {
	sourceutil.FileSource
}

// SourceOption configures the TOML source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.SetIgnoreMissing(ignore)
	}
}

// New creates a new Source that reads from a TOML file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewFileSource(path, "toml"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Source that reads from an io.Reader.
func NewReader(r io.Reader, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewReaderSource(r, "toml"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads and parses the TOML configuration data into a hierarchical map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	return s.ReadAndParse(ctx, func(r io.Reader) (map[string]any, error) {
		var data map[string]any
		dec := stdtoml.NewDecoder(r)
		if err := dec.Decode(&data); err != nil {
			if err == io.EOF {
				return make(map[string]any), nil
			}
			return nil, err
		}
		return data, nil
	})
}
