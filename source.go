package goconf

import (
	"context"
)

// Source represents an abstraction for configuration data sources.
// Any format provider (dotenv, system environment variables, JSON, YAML, TOML, remote KV stores)
// implements this interface.
type Source interface {
	// Load reads and parses configuration data into a key-value or hierarchical map.
	Load(ctx context.Context) (map[string]any, error)
}

// NamedSource is an optional interface that a Source can implement
// to provide a descriptive identifier for logging and error reporting.
type NamedSource interface {
	Name() string
}
