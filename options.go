package goconf

import (
	"context"
	"io"

	"github.com/Denio1337/goconf/source/dotenv"
)

// Option is a functional option for configuring a Loader.
type Option func(*Loader)

// WithSource appends one or more configuration sources to the Loader.
// Sources are evaluated in order; later sources override values from earlier ones.
func WithSource(sources ...Source) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, sources...)
	}
}

// WithDotEnv adds one or more .env file sources.
// If no filenames are passed, it defaults to ".env".
func WithDotEnv(filenames ...string) Option {
	return func(l *Loader) {
		if len(filenames) == 0 {
			filenames = []string{".env"}
		}
		for _, name := range filenames {
			l.sources = append(l.sources, dotenv.New(name, dotenv.WithIgnoreMissing(l.ignoreMissing)))
		}
	}
}

// WithDotEnvReader adds a .env source that reads from an io.Reader.
func WithDotEnvReader(r io.Reader) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, dotenv.NewReader(r))
	}
}

// WithStrictUnknown enables or disables error reporting for keys found in sources
// that do not match any field in the target struct.
func WithStrictUnknown(strict bool) Option {
	return func(l *Loader) {
		l.strictUnknown = strict
	}
}

// WithIgnoreMissing configures whether missing files should be ignored instead of returning an error.
func WithIgnoreMissing(ignore bool) Option {
	return func(l *Loader) {
		l.ignoreMissing = ignore
	}
}

// WithContext sets a context for the loading process.
func WithContext(ctx context.Context) Option {
	return func(l *Loader) {
		if ctx != nil {
			l.ctx = ctx
		}
	}
}
