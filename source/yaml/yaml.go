// Package yaml provides a YAML file and reader configuration Source.
package yaml

import (
	"context"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Source loads configuration from a YAML file or an io.Reader.
type Source struct {
	path          string
	reader        io.Reader
	ignoreMissing bool
}

// SourceOption configures the YAML source.
type SourceOption func(*Source)

// WithIgnoreMissing configures whether missing files should be ignored.
func WithIgnoreMissing(ignore bool) SourceOption {
	return func(s *Source) {
		s.ignoreMissing = ignore
	}
}

// New creates a new Source that reads from a YAML file path.
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
		return fmt.Sprintf("yaml:%s", s.path)
	}
	return "yaml:reader"
}

// Load reads and parses the YAML configuration data into a hierarchical map.
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
			return nil, fmt.Errorf("failed to open YAML file %q: %w", s.path, err)
		}
		defer f.Close()
		r = f
	}

	dec := yaml.NewDecoder(r)
	var rawData any
	if err := dec.Decode(&rawData); err != nil {
		if err == io.EOF {
			return make(map[string]any), nil
		}
		if s.path != "" {
			return nil, fmt.Errorf("failed to parse YAML from %q: %w", s.path, err)
		}
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
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
}

func cleanYAMLMap(val any) any {
	switch v := val.(type) {
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			res[k] = cleanYAMLMap(item)
		}
		return res
	case map[any]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			res[fmt.Sprint(k)] = cleanYAMLMap(item)
		}
		return res
	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			res[i] = cleanYAMLMap(item)
		}
		return res
	default:
		return v
	}
}
