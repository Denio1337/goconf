// Package goconf provides a robust, strictly typed configuration management library for Go.
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
//   - Minimal Dependencies: Core library uses the Go standard library only, with lightweight optional modules for YAML and TOML.
package goconf

import (
	"context"
	"fmt"

	"github.com/Denio1337/goconf/source/dotenv"
	"github.com/Denio1337/goconf/source/env"
)

// Loader manages sources, options, and decoding configuration into target structs.
type Loader struct {
	sources       []Source
	prefix        string
	strictUnknown bool
	ignoreMissing bool
	ctx           context.Context
}

// New creates a new Loader with the given options.
func New(opts ...Option) *Loader {
	return NewLoader(opts...)
}

// NewLoader creates a new Loader with the given options.
func NewLoader(opts ...Option) *Loader {
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
	// Default: if no sources were explicitly added, load .env (ignoring if missing)
	// followed by OS environment variables, giving primary priority to environment variables.
	if len(sources) == 0 {
		sources = []Source{
			dotenv.New(".env", dotenv.WithIgnoreMissing(true)),
			env.New(),
		}
	}

	st := NewStore()

	for _, src := range sources {
		data, err := src.Load(l.ctx)
		if err != nil {
			return fmt.Errorf("source %q failed to load: %w", src.Name(), err)
		}
		st.Merge(data)
	}

	decoder := NewDecoder(st)
	decoder.SetPrefix(l.prefix)
	decoder.SetStrictUnknown(l.strictUnknown)

	return decoder.Decode(target)
}

// Load is a top-level convenience function that initializes a Loader, applies options,
// and decodes configuration from sources into target.
// When called without explicit sources, it loads .env (ignoring missing file by default)
// and OS environment variables, giving primary priority to environment variables.
// target must be a non-nil pointer to a struct.
//
// Example:
//
//	type Config struct {
//	    Port int `key:"PORT" default:"8080"`
//	}
//	var cfg Config
//	if err := goconf.Load(&cfg); err != nil {
//	    log.Fatalf("failed to load configuration: %v", err)
//	}
func Load(target any, opts ...Option) error {
	return NewLoader(opts...).Load(target)
}

// MustLoad behaves like Load, but panics if an error occurs.
// Useful during application bootstrapping (e.g. in func main or init).
func MustLoad(target any, opts ...Option) {
	if err := Load(target, opts...); err != nil {
		panic(fmt.Sprintf("goconf: %v", err))
	}
}
