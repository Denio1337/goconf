package decoder

import (
	"errors"
	"fmt"
	"log/slog"
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

// SecretMarker is implemented by Secret[T] to indicate a sensitive value.
type SecretMarker interface {
	IsSecret()
}

// Validator is an optional interface that structs or fields can implement
// to execute custom business-level validation logic after decoding.
type Validator interface {
	Validate() error
}

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

// LogValue implements slog.LogValuer to ensure secrets are never leaked in structured logging.
func (e FieldError) LogValue() slog.Value {
	val := e.Value
	if e.IsSecret {
		val = "[SECRET]"
	}
	errStr := "<nil>"
	if e.Err != nil {
		if e.IsSecret {
			errStr = "validation or syntax error"
		} else {
			errStr = e.Err.Error()
		}
	}
	return slog.GroupValue(
		slog.String("field", e.Field),
		slog.String("key", e.Key),
		slog.Any("value", val),
		slog.String("target_type", e.TargetType),
		slog.String("error", errStr),
		slog.Bool("is_secret", e.IsSecret),
	)
}

// Format implements fmt.Formatter to prevent secret leaks with %+v or %#v.
func (e FieldError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') || f.Flag('#') {
			val := e.Value
			if e.IsSecret {
				val = `"[SECRET]"`
			}
			errStr := "<nil>"
			if e.Err != nil {
				if e.IsSecret {
					errStr = "validation or syntax error"
				} else {
					errStr = e.Err.Error()
				}
			}
			fmt.Fprintf(f, "{Field:%q Key:%q Value:%v TargetType:%q Err:%s IsSecret:%t}",
				e.Field, e.Key, val, e.TargetType, errStr, e.IsSecret)
			return
		}
		fmt.Fprint(f, e.Error())
	case 's':
		fmt.Fprint(f, e.Error())
	case 'q':
		fmt.Fprintf(f, "%q", e.Error())
	default:
		fmt.Fprint(f, e.Error())
	}
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

// LogValue implements slog.LogValuer for ValidationError.
func (v ValidationError) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(v.Errors))
	for i, err := range v.Errors {
		attrs[i] = slog.Any(fmt.Sprintf("[%d]", i+1), err.LogValue())
	}
	return slog.GroupValue(attrs...)
}

// Format implements fmt.Formatter for ValidationError.
func (v ValidationError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') || f.Flag('#') {
			var sb strings.Builder
			fmt.Fprintf(&sb, "ValidationError{\n")
			for _, err := range v.Errors {
				fmt.Fprintf(&sb, "  %+v\n", err)
			}
			fmt.Fprintf(&sb, "}")
			fmt.Fprint(f, sb.String())
			return
		}
		fmt.Fprint(f, v.Error())
	default:
		fmt.Fprint(f, v.Error())
	}
}

func formatValue(v any) string {
	if _, ok := v.(SecretMarker); ok {
		return `"[SECRET]"`
	}
	raw := fmt.Sprint(v)
	runes := []rune(raw)
	if len(runes) > 40 {
		raw = string(runes[:37]) + "..."
	}
	return fmt.Sprintf("%q", raw)
}
