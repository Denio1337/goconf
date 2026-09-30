# goconf

[![Go Reference](https://pkg.go.dev/badge/github.com/Denio1337/goconf.svg)](https://pkg.go.dev/github.com/Denio1337/goconf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Denio1337/goconf)](https://goreportcard.com/report/github.com/Denio1337/goconf)

**goconf** is a modern, robust, and strictly typed configuration library for Go. It loads and merges configuration from multiple sources (`.env`, `INI`, `JSON`, `YAML`, `TOML`, environment variables, or custom providers) into a target Go struct with comprehensive type validation and cascading precedence.

## Quick Start

### Installation

```bash
go get github.com/Denio1337/goconf
```

> **Requirements**: Go **1.22.0** or higher.


### Usage

Create a `.env` file (or provide environment variables in your deployment environment):

```env
APP_NAME=MyService
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
DATABASE_HOST=postgres.internal
DATABASE_PASSWORD=super-secret-production-password
```

Define your configuration struct and load it:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Denio1337/goconf"
)

// ServerConfig demonstrates nested structures and custom validation via Validator.
// Note: `key` tags are completely optional!
type ServerConfig struct {
	Host         string        `default:"localhost"`
	Port         int           `default:"8080"`
	ReadTimeout  time.Duration `default:"5s"`
	WriteTimeout time.Duration `default:"10s"`
}

// Validate implements goconf.Validator to enforce domain constraints after decoding.
func (s *ServerConfig) Validate() error {
	if s.Port < 1024 || s.Port > 65535 {
		return fmt.Errorf("server port %d must be in range 1024-65535", s.Port)
	}
	return nil
}

type DatabaseConfig struct {
	Host     string                `default:"localhost"`
	Port     int                   `default:"5432"`
	User     string                `default:"postgres"`
	// Protected against accidental leaks
    Password goconf.Secret[string] `required:"true"` 
}

type Config struct {
	AppName  string `default:"MyService"`
	Debug    bool   `default:"false"`
	Server   ServerConfig
	Database DatabaseConfig
}

func main() {
	var cfg Config

	// Load configuration: checks OS environment variables with top priority,
	// falling back to .env (missing .env is automatically ignored).
	if err := goconf.Load(&cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("--- Configuration Loaded Successfully ---")
	fmt.Printf("App: %s (Debug: %t)\n", cfg.AppName, cfg.Debug)
	fmt.Printf("Server: %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d\n\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port)

	// Safe printing demonstration with goconf.Secret[T]:
	fmt.Println("--- Sensitive Data Protection (Secret[T]) ---")
	fmt.Printf("Database struct dump (%%+v) : %+v\n", cfg.Database)
	fmt.Printf("Direct field print (%%s)     : %s\n", cfg.Database.Password)
	fmt.Printf("Raw value via .Value()      : %s\n", cfg.Database.Password.Value())
}
```

#### Output:

```text
--- Configuration Loaded Successfully ---
App: MyService (Debug: false)
Server: localhost:8080 (ReadTimeout: 10s)
Database: postgres@postgres.internal:5432

--- Sensitive Data Protection (Secret[T]) ---
Database struct dump (%+v) : {Host:postgres.internal Port:5432 User:postgres Password:[SECRET]}
Direct field print (%s)     : [SECRET]
Raw value via .Value()      : super-secret-production-password
```

## Key Features

- 🛡️ **Strict Schema & Type Safety**: Eliminates hidden type conversion bugs at startup. Every value is strictly converted and validated according to your struct schema.
- 📋 **Multi-Error Aggregation**: Collects all configuration and validation errors across your entire struct into a single, detailed report instead of aborting on the first failure.
- 🔄 **Cascading Multi-Source Overrides**: Combine multiple configuration layers with deterministic precedence (e.g. INI base defaults &rarr; YAML shared configs &rarr; TOML service configs &rarr; JSON local overrides &rarr; `.env` secrets).
- 🏷️ **Smart Name Resolution & Canonical `key` Tag**: Supports explicit `key` tags as well as automatic zero-tag resolution (supporting `snake_case`, `camelCase`, `kebab-case`, `UPPER_CASE`, `Section.Key`).
- 📁 **Built-in Format Sources**:
  - **OS Environment Variables**: Full 12-Factor app support (`WithEnv()`, `WithEnvPrefix("APP_")`), prefix stripping, and `__` double-underscore hierarchy mapping. [Example](examples/basic/).
  - **`.env`**: Full support for quotes, multiline values, escape sequences, inline comments, and variable interpolation (`${VAR:-default}`). [Example](examples/basic/main.go).
  - **`.ini`**: Sections `[section]`, nested subsections `[section.sub]`, comments (`;` and `#`), and hierarchical mapping. [Example](examples/ini/main.go).
  - **`.json`**: Hierarchical JSON documents with numeric precision preservation. [Example](examples/json/main.go).
  - **`.yaml`**: Clean YAML structure mapping. [Example](examples/yaml/main.go).
  - **`.toml`**: Native TOML tables and values. [Example](examples/toml/main.go).
- 🧩 **Comprehensive Go Type Support**:
  - Primitives (`int`, `uint`, `float`, `bool`, `string`)
  - Durations (`time.Duration`, e.g., `10s`, `5m`)
  - Dates and timestamps (`time.Time` with automatic layout detection or custom `layout` tag)
  - Network types (`net.IP`, `url.URL`)
  - Slices (`[]string`, `[]int`, `[]time.Duration`, etc., with custom `sep` delimiter)
  - Maps (`map[string]T`)
  - Pointers (allocated only when a corresponding value is present)
  - Nested and embedded structs with prefix inheritance
  - Custom deserialization via `encoding.TextUnmarshaler`
- 🔒 **Sensitive Data Protection (`Secret[T]`)**: Generic wrapper protecting passwords, tokens, API keys. Values are masked as `[SECRET]` in `fmt.Print*`, `log.Print*`, and JSON serialization, while accessible via `.Value()`.
- ✅ **Custom Validation (`Validator`)**: Domain-level business rule validation by implementing `Validate() error` on configuration structs, invoked automatically upon decoding.
- 🪶 **Minimal Dependencies**: The core library relies strictly on the Go standard library, with lightweight optional modules for YAML and TOML.

## Cascading Multi-Source Overrides

`goconf` allows layering multiple configuration sources with strict override precedence. Sources are evaluated in order; later sources override values provided by earlier sources.

```go

goconf.WithINI("config.ini")   // 1. Base defaults
goconf.WithYAML("config.yaml") // 2. Shared service config
goconf.WithTOML("config.toml") // 3. Environment overrides
goconf.WithJSON("local.json")  // 4. Local development overrides
goconf.WithDotEnv(".env")      // 5. Secrets and environment overrides
```

Check out [examples/complex](examples/complex) for a full runnable demonstration combining 5 cascading format layers.

## Struct Tags and Options

Tags are defined as exported constants in [tags.go](tags.go):

| Constant | Tag Name | Description | Example |
|---|---|---|---|
| `goconf.TagKey` | `key` | Name of the configuration key | `key:"PORT"` |
| `goconf.TagDefault` | `default` | Default value if the key is missing or empty | `default:"8080"` |
| `goconf.TagRequired` | `required` | Marks field as required. Returns an error if missing | `required:"true"` |
| `goconf.TagPrefix` | `prefix` | Key prefix for nested struct fields | `prefix:"DB_"` |
| `goconf.TagSep` | `sep` | Delimiter for slices and maps (default: `,`) | `sep:";"` |
| `goconf.TagLayout` | `layout` | Layout string for parsing `time.Time` | `layout:"2006-01-02"` |

### Prefix Rules and Inheritance

1. **Explicit prefix**: `prefix:"DB_"` prepends `DB_` to all child keys in the nested struct (`HOST` &rarr; `DB_HOST`).
2. **`key` as prefix**: When applied to a nested struct (e.g. `key:"DATABASE"`), it automatically forms the prefix `DATABASE_`.
3. **Disabling prefix**: `prefix:""` explicitly disables prefix inheritance on a named struct, flattening its fields.
4. **Absolute keys**: If a child field key begins with a slash `/` (e.g., `key:"/GLOBAL_SECRET"`), it **bypasses all parent prefixes**.
5. **Global prefix**: The option `goconf.WithPrefix("APP_")` prepends a global prefix to all keys in the root struct.

## Error Inspection & Multi-Error Reporting

Instead of failing on the first error, `goconf` accumulates all schema and type mismatches into a single structured report. See [example](examples/validation_errors/main.go).

## Protecting Sensitive Data (`Secret[T]`)

Wrap sensitive fields (passwords, tokens, private keys) in `goconf.Secret[T]` to ensure they are never accidentally leaked in terminal output, application logs, or JSON serialization:

```go
type DatabaseConfig struct {
    Host     string                `key:"HOST"`
    Password goconf.Secret[string] `key:"PASSWORD"`
    Port     int                   `key:"PORT" default:"5432"`
}

func main() {
    var cfg DatabaseConfig
    goconf.Load(&cfg)

    // Printing the struct or field directly always masks the value as [SECRET]:
    fmt.Printf("%+v\n", cfg)      // Output: {Host:localhost Password:[SECRET] Port:5432}
    fmt.Println(cfg.Password)     // Output: [SECRET]
    log.Println(cfg)              // Output: {localhost [SECRET] 5432}

    // Access the raw secret value safely when needed:
    rawPassword := cfg.Password.Value()
    db.Connect(rawPassword)
}
```

## Custom Validation (`Validator`)

Any struct can implement the `Validator` interface to enforce custom business constraints after decoding:

```go
type ServerConfig struct {
    Port int `key:"PORT" default:"8080"`
}

func (s *ServerConfig) Validate() error {
    if s.Port < 1024 || s.Port > 65535 {
        return fmt.Errorf("port %d is out of allowed user range (1024-65535)", s.Port)
    }
    return nil
}
```

## Extensible Source Architecture

Custom sources (e.g., etcd, HashiCorp Vault, AWS Secrets Manager, Kubernetes ConfigMaps) can be added by implementing the `Source` interface:

```go
type Source interface {
    Name() string
    Load(ctx context.Context) (map[string]any, error)
}
```

### Custom Source Example:

```go
type CustomSource struct {
    endpoint string
}

func (s *CustomSource) Name() string {
    return "custom:" + s.endpoint
}

func (s *CustomSource) Load(ctx context.Context) (map[string]any, error) {
    // Fetch data and return a flat or hierarchical map
    return map[string]any{
        "APP_NAME": "ClusterApp",
        "DATABASE_PORT": 5432,
    }, nil
}
```

Pass custom sources directly into `goconf.Load`:

```go
err := goconf.Load(&cfg, goconf.WithSource(&CustomSource{endpoint: "https://api.internal"}))
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
