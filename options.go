package goconf

import (
	"context"
	"io"

	"github.com/Denio1337/goconf/source/dotenv"
	"github.com/Denio1337/goconf/source/env"
	"github.com/Denio1337/goconf/source/ini"
	"github.com/Denio1337/goconf/source/json"
	"github.com/Denio1337/goconf/source/toml"
	"github.com/Denio1337/goconf/source/yaml"
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

// WithINI adds one or more INI file sources.
func WithINI(filenames ...string) Option {
	return func(l *Loader) {
		for _, name := range filenames {
			l.sources = append(l.sources, ini.New(name, ini.WithIgnoreMissing(l.ignoreMissing)))
		}
	}
}

// WithINIReader adds an INI source that reads from an io.Reader.
func WithINIReader(r io.Reader) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, ini.NewReader(r))
	}
}

// WithJSON adds one or more JSON file sources.
func WithJSON(filenames ...string) Option {
	return func(l *Loader) {
		for _, name := range filenames {
			l.sources = append(l.sources, json.New(name, json.WithIgnoreMissing(l.ignoreMissing)))
		}
	}
}

// WithJSONReader adds a JSON source that reads from an io.Reader.
func WithJSONReader(r io.Reader) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, json.NewReader(r))
	}
}

// WithYAML adds one or more YAML file sources.
func WithYAML(filenames ...string) Option {
	return func(l *Loader) {
		for _, name := range filenames {
			l.sources = append(l.sources, yaml.New(name, yaml.WithIgnoreMissing(l.ignoreMissing)))
		}
	}
}

// WithYAMLReader adds a YAML source that reads from an io.Reader.
func WithYAMLReader(r io.Reader) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, yaml.NewReader(r))
	}
}

// WithTOML adds one or more TOML file sources.
func WithTOML(filenames ...string) Option {
	return func(l *Loader) {
		for _, name := range filenames {
			l.sources = append(l.sources, toml.New(name, toml.WithIgnoreMissing(l.ignoreMissing)))
		}
	}
}

// WithTOMLReader adds a TOML source that reads from an io.Reader.
func WithTOMLReader(r io.Reader) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, toml.NewReader(r))
	}
}

// WithEnv adds OS environment variables as a configuration source.
// By default, it reads all environment variables.
func WithEnv(opts ...env.Option) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, env.New(opts...))
	}
}

// WithEnvPrefix adds OS environment variables matching the specified prefix
// (e.g. "APP_") as a configuration source.
func WithEnvPrefix(prefix string) Option {
	return func(l *Loader) {
		l.sources = append(l.sources, env.New(env.WithPrefix(prefix)))
	}
}

// WithoutAutoEnv disables automatically appending OS environment variables as a configuration source.
// By default, goconf automatically loads OS environment variables with highest priority.
func WithoutAutoEnv() Option {
	return func(l *Loader) {
		l.disableAutoEnv = true
	}
}

// WithAutoEnv controls whether OS environment variables are automatically loaded.
func WithAutoEnv(enable bool) Option {
	return func(l *Loader) {
		l.disableAutoEnv = !enable
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

// WithPrefix sets a global key prefix to be prepended to all configuration fields.
// For example, WithPrefix("APP_") will look for "APP_PORT", "APP_DATABASE_HOST", etc.
func WithPrefix(prefix string) Option {
	return func(l *Loader) {
		l.prefix = prefix
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
