package goconf

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// secretMarker is an unexported interface used by the decoder to recognize Secret[T] fields.
type secretMarker interface {
	isSecret()
}

// Secret wraps a value of any type T to protect sensitive information (such as passwords,
// tokens, and API keys) from accidental leakage in logs, fmt formatting, and JSON payloads.
// All string representations (%v, %+v, %#v, %s, %q) and json.Marshal output "[SECRET]".
type Secret[T any] struct {
	value T
}

// NewSecret creates a new Secret wrapping value.
func NewSecret[T any](value T) Secret[T] {
	return Secret[T]{value: value}
}

// isSecret implements secretMarker.
func (s Secret[T]) isSecret() {}

// Set updates the secret value.
func (s *Secret[T]) Set(val T) {
	s.value = val
}

// Value returns the raw underlying secret value.
func (s Secret[T]) Value() T {
	return s.value
}

// Unmask returns the raw underlying secret value. Alias for Value().
func (s Secret[T]) Unmask() T {
	return s.value
}

// String implements fmt.Stringer, returning "[SECRET]".
func (s Secret[T]) String() string {
	return "[SECRET]"
}

// GoString implements fmt.GoStringer, returning "[SECRET]".
func (s Secret[T]) GoString() string {
	return "[SECRET]"
}

// Format implements fmt.Formatter, ensuring all format verbs output "[SECRET]".
func (s Secret[T]) Format(f fmt.State, verb rune) {
	switch verb {
	case 'q':
		fmt.Fprint(f, `"[SECRET]"`)
	default:
		fmt.Fprint(f, "[SECRET]")
	}
}

// MarshalJSON implements json.Marshaler, serializing as "[SECRET]".
func (s Secret[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal("[SECRET]")
}

// UnmarshalJSON implements json.Unmarshaler, populating the underlying value.
func (s *Secret[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.value)
}

// MarshalText implements encoding.TextMarshaler, returning "[SECRET]".
func (s Secret[T]) MarshalText() ([]byte, error) {
	return []byte("[SECRET]"), nil
}

// LogValue implements slog.LogValuer, masking the secret value in structured logs.
func (s Secret[T]) LogValue() slog.Value {
	return slog.StringValue("[SECRET]")
}
