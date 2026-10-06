package goconf

import (
	"errors"

	"github.com/Denio1337/goconf/internal/decoder"
)

var (
	// ErrInvalidTarget is returned when target passed to Load is not a non-nil pointer to a struct.
	ErrInvalidTarget = decoder.ErrInvalidTarget

	// ErrMissingRequired is returned when a required field is missing or empty.
	ErrMissingRequired = decoder.ErrMissingRequired

	// ErrTypeMismatch is returned when a configuration value cannot be converted to the target field type.
	ErrTypeMismatch = decoder.ErrTypeMismatch

	// ErrValidationFailed is returned when a custom validator returns an error.
	ErrValidationFailed = decoder.ErrValidationFailed

	// ErrNotFound is returned or wrapped when a configuration file or source is not found.
	ErrNotFound = errors.New("configuration source not found")
)

// FieldError represents a detailed error associated with a specific struct field during decoding or validation.
type FieldError = decoder.FieldError

// ValidationError is a collection of FieldErrors encountered while parsing or validating configuration.
type ValidationError = decoder.ValidationError
