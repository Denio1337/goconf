// Package goenv provides a robust, strictly typed configuration management library for Go.
//
// It loads configuration data from multiple sources (such as .env files, OS environment
// variables, JSON, YAML, TOML) into a single Go struct passed by reference.
//
// Key Features:
//   - Strict Schema Validation: Guarantees full type safety, reporting clear, detailed
//     messages for any mismatched or invalid fields.
//   - Multi-Error Reporting: Collects all validation and type errors across your configuration
//     instead of aborting on the first error.
//   - Extensible Source Architecture: Seamlessly plug in new formats and remote providers
//     by implementing the simple Source interface.
//   - DotEnv (.env) Support: Full support for quoted strings, multiline values, escapes,
//     inline comments, and variable interpolation (${VAR:-default}).
//   - Zero Dependencies: Built entirely using the Go standard library.
package goenv

import (
	"context"
	"fmt"

	"github.com/Denio1337/goenv/source/dotenv"
	"github.com/Denio1337/goenv/store"
)

// Loader manages sources, options, and decoding configuration into target structs.
type Loader struct {
	sources       []Source
	strictUnknown bool
	ignoreMissing bool
	ctx           context.Context
}

// New creates a new Loader with the given options.
func New(opts ...Option) *Loader {
	l := &Loader{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Load reads data from all registered sources and decodes it into target.
// target must be a non-nil pointer to a struct.
func (l *Loader) Load(target any) error {
	sources := l.sources
	// Default to .env if no sources were explicitly added
	if len(sources) == 0 {
		sources = []Source{dotenv.New(".env", dotenv.WithIgnoreMissing(l.ignoreMissing))}
	}

	st := store.New()

	for _, src := range sources {
		data, err := src.Load(l.ctx)
		if err != nil {
			return fmt.Errorf("source %q failed to load: %w", src.Name(), err)
		}
		st.Merge(data)
	}

	decoder := NewDecoder(st)
	decoder.SetStrictUnknown(l.strictUnknown)

	return decoder.Decode(target)
}

// Load is a top-level convenience function that initializes a Loader, applies options,
// and decodes configuration from sources into target.
// target must be a non-nil pointer to a struct.
//
// Example:
//
//	type Config struct {
//	    Port int `env:"PORT" default:"8080"`
//	}
//	var cfg Config
//	if err := goenv.Load(&cfg); err != nil {
//	    log.Fatalf("failed to load configuration: %v", err)
//	}
func Load(target any, opts ...Option) error {
	return New(opts...).Load(target)
}

// MustLoad behaves like Load, but panics if an error occurs.
// Useful during application bootstrapping (e.g. in func main or init).
func MustLoad(target any, opts ...Option) {
	if err := Load(target, opts...); err != nil {
		panic(fmt.Sprintf("goenv: %v", err))
	}
}
