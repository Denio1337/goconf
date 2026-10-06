// Package yaml provides a YAML file and reader configuration Source.
package yaml

import (
	"context"
	"fmt"
	"io"

	"github.com/Denio1337/goconf/internal/sourceutil"
	"gopkg.in/yaml.v3"
)

// Source loads configuration from a YAML file or an io.Reader.
type Source struct {
	sourceutil.FileSource
}

// SourceOption configures the YAML source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.SetIgnoreMissing(ignore)
	}
}

// New creates a new Source that reads from a YAML file path.
func New(path string, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewFileSource(path, "yaml"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Source that reads from an io.Reader.
func NewReader(r io.Reader, opts ...SourceOption) *Source {
	s := &Source{
		FileSource: sourceutil.NewReaderSource(r, "yaml"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load reads and parses the YAML configuration data into a hierarchical map.
func (s *Source) Load(ctx context.Context) (map[string]any, error) {
	return s.ReadAndParse(ctx, func(r io.Reader) (map[string]any, error) {
		dec := yaml.NewDecoder(r)
		var rawData any
		if err := dec.Decode(&rawData); err != nil {
			if err == io.EOF {
				return make(map[string]any), nil
			}
			return nil, err
		}

		if rawData == nil {
			return make(map[string]any), nil
		}

		cleaned := cleanYAMLMap(rawData)
		res, ok := cleaned.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("YAML root must be a mapping, got %T", rawData)
		}
		return res, nil
	})
}

// cleanYAMLMap converts map[any]any to map[string]any, optimizing allocations where possible.
func cleanYAMLMap(val any) any {
	switch v := val.(type) {
	case map[string]any:
		for k, item := range v {
			v[k] = cleanYAMLMap(item)
		}
		return v
	case map[any]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			res[fmt.Sprint(k)] = cleanYAMLMap(item)
		}
		return res
	case []any:
		for i, item := range v {
			v[i] = cleanYAMLMap(item)
		}
		return v
	default:
		return v
	}
}
