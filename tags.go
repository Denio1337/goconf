package goconf

// Canonical struct tag constants used across the library for schema definition.
// These constants are exported so that users and developers have a single,
// reliable reference for all supported tag names.
const (
	// TagKey is the primary struct tag defining the configuration key name.
	// Supports comma-separated options: required, default=VALUE.
	//
	// Example:
	//   Port int `key:"PORT"`
	//   Host string `key:"HOST,required,default=localhost"`
	//   Global string `key:"/GLOBAL_KEY"` // starts with '/' to ignore parent prefixes
	TagKey = "key"

	// TagDefault specifies the default fallback value if the key is not set or empty.
	//
	// Example:
	//   Port int `key:"PORT" default:"8080"`
	TagDefault = "default"

	// TagRequired marks the field as mandatory.
	// If the value is missing in sources and no default is set, a ValidationError is returned.
	//
	// Example:
	//   Password string `key:"DB_PASSWORD" required:"true"`
	TagRequired = "required"

	// TagPrefix specifies a key prefix for all fields within a nested or embedded struct.
	// Use prefix:"" to explicitly disable auto-prefixing on a named nested struct.
	//
	// Example:
	//   Database DatabaseConfig `prefix:"DATABASE_"`
	TagPrefix = "prefix"

	// TagSep defines the item delimiter for slices and maps.
	// Default delimiter is comma (",").
	//
	// Example:
	//   Ports []int `key:"PORTS" sep:";"`
	TagSep = "sep"

	// TagLayout specifies a custom time layout for parsing time.Time fields.
	//
	// Example:
	//   CreatedAt time.Time `key:"CREATED_AT" layout:"2006-01-02"`
	TagLayout = "layout"

	// TagDescription holds a human-readable description for documentation/help.
	//
	// Example:
	//   Port int `key:"PORT" doc:"TCP port to listen on"`
	TagDescription = "doc"
)
