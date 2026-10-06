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
//   - Minimal Dependencies: Core library uses the Go standard library only, with lightweight modules for YAML and TOML.
package goconf

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/Denio1337/goconf/internal/decoder"
	"github.com/Denio1337/goconf/internal/store"
	"github.com/Denio1337/goconf/source/dotenv"
	"github.com/Denio1337/goconf/source/env"
)

// Validator is an optional interface that structs or fields can implement
// to execute custom business-level validation logic after decoding.
type Validator = decoder.Validator

// Loader manages sources, options, and decoding configuration into target structs.
// It is safe for concurrent use by multiple goroutines.
type Loader struct {
	mu             sync.RWMutex
	sources        []Source
	prefix         string
	strictUnknown  bool
	ignoreMissing  bool
	disableAutoEnv bool
	ctx            context.Context
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
	l.mu.RLock()
	prefix := l.prefix
	strictUnknown := l.strictUnknown
	ignoreMissing := l.ignoreMissing
	disableAutoEnv := l.disableAutoEnv
	ctx := l.ctx
	sources := make([]Source, 0, len(l.sources)+2)
	sources = append(sources, l.sources...)
	l.mu.RUnlock()

	// Default to .env if no sources were explicitly added
	if len(sources) == 0 {
		sources = append(sources, dotenv.New(".env", dotenv.WithIgnoreMissing(true)))
	}

	var autoEnvSource Source
	if !disableAutoEnv {
		autoEnvSource = env.New()
		sources = append(sources, autoEnvSource)
	}

	st := store.New()

	for _, src := range sources {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("loading canceled: %w", err)
		}

		data, err := src.Load(ctx)
		if err != nil && ignoreMissing && isNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("source %q failed to load: %w", sourceName(src), err)
		}

		isAmbient := src == autoEnvSource
		st.MergeWithAmbient(data, !isAmbient, isAmbient)
	}

	dec := decoder.New(st)
	dec.SetPrefix(prefix)
	dec.SetStrictUnknown(strictUnknown)

	return dec.Decode(target)
}

// MustLoad behaves like Load, but panics if an error occurs.
func (l *Loader) MustLoad(target any) {
	if err := l.Load(target); err != nil {
		panic(fmt.Sprintf("goconf: %v", err))
	}
}

// Load is a top-level convenience function that initializes a Loader, applies options,
// and decodes configuration from sources into target.
// When called without explicit sources, it loads .env (ignoring missing file by default)
// and OS environment variables, giving primary priority to environment variables.
// target must be a non-nil pointer to a struct.
func Load(target any, opts ...Option) error {
	return New(opts...).Load(target)
}

// MustLoad behaves like Load, but panics if an error occurs.
// Useful during application bootstrapping (e.g. in func main or init).
func MustLoad(target any, opts ...Option) {
	New(opts...).MustLoad(target)
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrNotFound)
}

func sourceName(s Source) string {
	if n, ok := s.(NamedSource); ok {
		return n.Name()
	}
	if str, ok := s.(fmt.Stringer); ok {
		return str.String()
	}
	return fmt.Sprintf("%T", s)
}
