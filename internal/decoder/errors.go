package decoder

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidTarget is returned when target passed to Load is not a non-nil pointer to a struct.
	ErrInvalidTarget = errors.New("target must be a non-nil pointer to a struct")

	// ErrMissingRequired is returned when a required field is missing or empty.
	ErrMissingRequired = errors.New("required field is missing or empty")

	// ErrTypeMismatch is returned when a configuration value cannot be converted to the target field type.
	ErrTypeMismatch = errors.New("type mismatch or invalid syntax")

	// ErrValidationFailed is returned when a custom validator returns an error.
	ErrValidationFailed = errors.New("validation constraint failed")
)

// FieldError represents a detailed error associated with a specific struct field during decoding or validation.
type FieldError struct {
	Field      string
	Key        string
	Value      any
	TargetType string
	Err        error
	IsSecret   bool
}

// Error returns a formatted, human-readable error message.
func (e *FieldError) Error() string {
	var keyInfo string
	if e.Key != "" {
		keyInfo = fmt.Sprintf(" (key %q)", e.Key)
	}

	var valInfo string
	if e.Value != nil {
		if e.IsSecret {
			valInfo = ` with value "[SECRET]"`
		} else {
			valInfo = fmt.Sprintf(" with value %v", formatValue(e.Value))
		}
	}

	if errors.Is(e.Err, ErrMissingRequired) {
		return fmt.Sprintf("field %q%s: required field is missing or empty", e.Field, keyInfo)
	}

	if e.IsSecret {
		if errors.Is(e.Err, ErrValidationFailed) {
			return fmt.Sprintf("field %q%s: validation constraint failed", e.Field, keyInfo)
		}
		if e.TargetType != "" {
			return fmt.Sprintf("field %q%s%s: cannot convert to %s: invalid syntax or type mismatch", e.Field, keyInfo, valInfo, e.TargetType)
		}
		return fmt.Sprintf("field %q%s: decode or validation failed", e.Field, keyInfo)
	}

	if e.TargetType != "" {
		return fmt.Sprintf("field %q%s%s: cannot convert to %s: %v", e.Field, keyInfo, valInfo, e.TargetType, e.Err)
	}

	return fmt.Sprintf("field %q%s: %v", e.Field, keyInfo, e.Err)
}

// Unwrap returns the underlying error.
func (e *FieldError) Unwrap() error {
	return e.Err
}

// ValidationError is a collection of FieldErrors encountered while parsing or validating configuration.
type ValidationError struct {
	Errors []FieldError
}

// Error formats all collected errors into a structured, readable list.
func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return "goconf: validation failed"
	}
	if len(v.Errors) == 1 {
		return fmt.Sprintf("goconf schema error: %s", v.Errors[0].Error())
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "goconf: schema validation failed with %d error(s):\n", len(v.Errors))
	for i, err := range v.Errors {
		fmt.Fprintf(&sb, "  [%d] %s\n", i+1, err.Error())
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Unwrap returns the list of errors as a slice of error for Go 1.20+ error matching.
func (v *ValidationError) Unwrap() []error {
	errs := make([]error, len(v.Errors))
	for i := range v.Errors {
		errs[i] = &v.Errors[i]
	}
	return errs
}

// HasErrors returns true if any errors were collected.
func (v *ValidationError) HasErrors() bool {
	return len(v.Errors) > 0
}

// Add appends a new FieldError.
func (v *ValidationError) Add(fe FieldError) {
	v.Errors = append(v.Errors, fe)
}

func formatValue(v any) string {
	if _, ok := v.(SecretMarker); ok {
		return `"[SECRET]"`
	}
	str := fmt.Sprintf("%q", fmt.Sprint(v))
	if len(str) > 50 {
		return str[:47] + "...\""
	}
	return str
}
