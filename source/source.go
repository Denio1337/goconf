package source

import (
	"context"
)

// Source represents an abstraction for configuration data sources.
// Any format provider (dotenv, system environment variables, JSON, YAML, TOML, remote KV stores)
// implements this interface.
type Source interface {
	// Name returns a descriptive identifier for the source (e.g. "dotenv:.env", "os.env", "json:config.json").
	Name() string

	// Load reads and parses configuration data into a key-value or hierarchical map.
	Load(ctx context.Context) (map[string]any, error)
}
