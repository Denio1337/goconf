# goconf

[![Go Reference](https://pkg.go.dev/badge/github.com/Denio1337/goconf.svg)](https://pkg.go.dev/github.com/Denio1337/goconf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Denio1337/goconf)](https://goreportcard.com/report/github.com/Denio1337/goconf)

**goconf** is a modern, robust, and strictly typed configuration library for Go. It loads and merges configuration from multiple sources (`.env`, `INI`, `JSON`, `YAML`, `TOML`, environment variables, or custom providers) into a target Go struct with comprehensive type validation and cascading precedence.

---

## Key Features

- 🛡️ **Strict Schema & Type Safety**: Eliminates hidden type conversion bugs at startup. Every value is strictly converted and validated according to your struct schema.
- 📋 **Multi-Error Aggregation**: Collects all configuration and validation errors across your entire struct into a single, detailed report instead of aborting on the first failure.
- 🔄 **Cascading Multi-Source Overrides**: Combine multiple configuration layers with deterministic precedence (e.g. INI base defaults &rarr; YAML shared configs &rarr; TOML service configs &rarr; JSON local overrides &rarr; `.env` secrets).
- 🏷️ **Smart Name Resolution & Canonical `key` Tag**: Supports explicit `key` tags as well as automatic zero-tag resolution (supporting `snake_case`, `camelCase`, `kebab-case`, `UPPER_CASE`, `Section.Key`, and common database synonyms like `db_name`/`database`).
- 📁 **Built-in Format Sources**:
  - **OS Environment Variables**: Full 12-Factor app support (`WithEnv()`, `WithEnvPrefix("APP_")`), prefix stripping, and `__` double-underscore hierarchy mapping.
  - **`.env`**: Full support for quotes, multiline values, escape sequences, inline comments, and variable interpolation (`${VAR:-default}`).
  - **`.ini`**: Sections `[section]`, nested subsections `[section.sub]`, comments (`;` and `#`), and hierarchical mapping.
  - **`.json`**: Hierarchical JSON documents with numeric precision preservation.
  - **`.yaml`**: Clean YAML structure mapping.
  - **`.toml`**: Native TOML tables and values.
  - **In-Memory**: Direct programmatic and testing support via `NewStore()`.
- 🧩 **Comprehensive Go Type Support**:
  - Primitives (`int*`, `uint*`, `float*`, `bool`, `string`)
  - Durations (`time.Duration`, e.g., `10s`, `5m`)
  - Dates and timestamps (`time.Time` with automatic layout detection or custom `layout` tag)
  - Network types (`net.IP`, `*url.URL`)
  - Slices (`[]string`, `[]int`, `[]time.Duration`, etc., with custom `sep` delimiter)
  - Maps (`map[string]T`)
  - Pointers (allocated only when a corresponding value is present)
  - Nested and embedded structs with prefix inheritance
  - Custom deserialization via `encoding.TextUnmarshaler`
  - Business logic validation via the `Validator` interface
- 🪶 **Minimal Dependencies**: The core library relies strictly on the Go standard library, with lightweight optional modules for YAML and TOML.

---

## Installation

```bash
go get github.com/Denio1337/goconf
```

> **Requirements**: Go **1.22.0** or higher.

---

## Quick Start

Create a `.env` file:

```env
APP_NAME=MyService
ENVIRONMENT=production
DEBUG=false

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT=30s

DATABASE_HOST=postgres.internal
DATABASE_PORT=5432
DATABASE_PASSWORD=secret-password
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

type ServerConfig struct {
	Host    string        `key:"HOST" default:"localhost"`
	Port    int           `key:"PORT" default:"8080"`
	Timeout time.Duration `key:"TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	Host     string `key:"HOST"`
	Port     int    `key:"PORT" default:"5432"`
	Password string `key:"PASSWORD" required:"true"`
}

type Config struct {
	AppName     string         `key:"APP_NAME"`
	Environment string         `key:"ENVIRONMENT" default:"development"`
	Debug       bool           `key:"DEBUG"`
	Server      ServerConfig   `prefix:"SERVER_"`
	Database    DatabaseConfig `prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	// Load configuration from .env file
	if err := goconf.Load(&cfg, goconf.WithDotEnv(".env")); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Printf("Service: %s (%s)\n", cfg.AppName, cfg.Environment)
	fmt.Printf("Server listening on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database host: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
}
```

---

## Cascading Multi-Source Overrides

`goconf` allows layering multiple configuration sources with strict override precedence. Sources are evaluated in order; later sources override values provided by earlier sources.

```go
package main

import (
	"fmt"
	"log"

	"github.com/Denio1337/goconf"
)

type Config struct {
	AppName  string `key:"app_name"`
	Debug    bool   `key:"debug"`
	Server   ServerConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Host string `key:"host"`
	Port int    `key:"port"`
}

type DatabaseConfig struct {
	Host     string `key:"host"`
	Port     int    `key:"port"`
	Name     string `key:"db_name"`
	User     string `key:"user"`
	Password string `key:"password"`
}

func main() {
	var cfg Config

	err := goconf.Load(&cfg,
		goconf.WithINI("config.ini"),   // 1. Base defaults
		goconf.WithYAML("config.yaml"), // 2. Shared service config
		goconf.WithTOML("config.toml"), // 3. Environment overrides
		goconf.WithJSON("local.json"),  // 4. Local development overrides
		goconf.WithDotEnv(".env"),      // 5. Secrets and environment overrides
	)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Loaded App: %s, DB: %s\n", cfg.AppName, cfg.Database.Name)
}
```

Check out [examples/complex](examples/complex) for a full runnable demonstration combining 5 cascading format layers.

---

## 12-Factor App & OS Environment Variables

`goconf` provides native support for cloud-native and containerized 12-Factor applications where configuration is provided entirely through operating system environment variables without any files on disk:

```go
// Read all OS environment variables
err := goconf.Load(&cfg, goconf.WithEnv())

// Or filter and automatically strip an application prefix (e.g. APP_PORT=8080 -> PORT)
err := goconf.Load(&cfg, goconf.WithEnvPrefix("APP_"))
```

- **Hierarchy mapping**: Double underscores `__` in variable names are automatically mapped to nested struct fields (e.g. `SERVER__PORT=8080` or `DATABASE__HOST=postgres` map cleanly to `cfg.Server.Port` and `cfg.Database.Host`).
- **File + Environment layering**: Easily combine base configuration files with environment variable overrides:
  ```go
  err := goconf.Load(&cfg,
      goconf.WithIgnoreMissing(true),
      goconf.WithDotEnv(".env"), // Developer defaults (if present)
      goconf.WithEnv(),          // Production OS env overrides
  )
  ```

---

## Struct Tags and Options

Tags are defined as exported constants in `tags.go`:

| Constant | Tag Name | Description | Example |
|---|---|---|---|
| `goconf.TagKey` | `key` | Name of the configuration key | `key:"PORT"` |
| `goconf.TagDefault` | `default` | Default value if the key is missing or empty | `default:"8080"` |
| `goconf.TagRequired` | `required` | Marks field as required. Returns an error if missing | `required:"true"` |
| `goconf.TagPrefix` | `prefix` | Key prefix for nested struct fields | `prefix:"DB_"` |
| `goconf.TagSep` | `sep` | Delimiter for slices and maps (default: `,`) | `sep:";"` |
| `goconf.TagLayout` | `layout` | Layout string for parsing `time.Time` | `layout:"2006-01-02"` |

### Shorthand Tag Syntax

Options can be combined inside a single `key` tag:

```go
type ServerConfig struct {
    Port int `key:"PORT,required,default=8080"`
}
```

### Prefix Rules and Inheritance

1. **Explicit prefix**: `prefix:"DB_"` prepends `DB_` to all child keys in the nested struct (`HOST` &rarr; `DB_HOST`).
2. **`key` as prefix**: When applied to a nested struct (e.g. `key:"DATABASE"`), it automatically forms the prefix `DATABASE_`.
3. **Disabling prefix**: `prefix:""` explicitly disables prefix inheritance on a named struct, flattening its fields.
4. **Absolute keys**: If a child field key begins with a slash `/` (e.g., `key:"/GLOBAL_SECRET"`), it **bypasses all parent prefixes**.
5. **Global prefix**: The option `goconf.WithPrefix("APP_")` prepends a global prefix to all keys in the root struct.

### Zero-Tag Automatic Resolution

If struct tags are omitted, `goconf` automatically maps struct fields using canonical naming conventions:
- Field `Host` matches `host`, `HOST`, `server.host`, `server_host`, `SERVER_HOST`.
- Field `Name` under `DatabaseConfig` matches `db_name`, `dbname`, `database`, `name`, `database_name`.
- Hyphenated, underscored, and dotted forms are seamlessly normalized (`server-port`, `server_port`, `server.port`).

---

## Error Inspection & Multi-Error Reporting

Instead of failing on the first error, `goconf` accumulates all schema and type mismatches into a single structured report:

```go
var valErr *goconf.ValidationError
if errors.As(err, &valErr) {
    for _, fe := range valErr.Errors {
        fmt.Printf("Field:       %s\n", fe.Field)       // e.g. "Server.Port"
        fmt.Printf("Key:         %s\n", fe.Key)         // e.g. "SERVER_PORT"
        fmt.Printf("Value:       %v\n", fe.Value)       // e.g. "invalid_number"
        fmt.Printf("Target Type: %s\n", fe.TargetType)  // e.g. "int"
        fmt.Printf("Reason:      %v\n", fe.Err)         // e.g. strconv.ErrSyntax
    }
}
```

Errors can also be tested using standard `errors.Is`:

```go
if errors.Is(err, goconf.ErrMissingRequired) {
    // Handle missing required fields
}
```

---

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

---

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

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
