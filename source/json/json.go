// Package json provides a JSON file and reader configuration Source.
package json

import (
	"context"
	stdjson "encoding/json"
	"io"

	"github.com/Denio1337/goconf/internal/sourceutil"
)

// Source loads configuration from a JSON file or an io.Reader.
type Source struct {
	sourceutil.FileSource
}

// SourceOption configures the JSON source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.SetIgnoreMissing(ignore)
	}
}

// New creates a new Source that reads from a JSON file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewFileSource(path, "json"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Source that reads from an io.Reader.
func NewReader(r io.Reader, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewReaderSource(r, "json"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads and parses the JSON configuration data into a hierarchical map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	return s.ReadAndParse(ctx, func(r io.Reader) (map[string]any, error) {
		dec := stdjson.NewDecoder(r)
		dec.UseNumber()

		var data map[string]any
		if err := dec.Decode(&data); err != nil {
			return nil, err
		}
		return data, nil
	})
}
