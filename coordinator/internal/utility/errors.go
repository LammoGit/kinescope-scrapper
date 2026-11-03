package utility

import (
	"errors"
	"fmt"
	"net/http"
)

// ValidationError
var ValidationError error

// ConflictError
var NotFoundError error

// ConflictError
var ConflictError error

// InternalError
var InternalError error

// NewValidationError returns a new validation error with the given text
func NewValidationError(v string) error {
	return fmt.Errorf("%w%s", ValidationError, v)
}

// NewNotFoundError returns a new not found error with the given text
func NewNotFoundError(v string) error {
	return fmt.Errorf("%w%s", NotFoundError, v)
}

// NewConflictError returns a new conflict error with the given text
func NewConflictError(v string) error {
	return fmt.Errorf("%w%s", ConflictError, v)
}

// NewInternalError returns a new internal error with the given text
func NewInternalError(v string) error {
	return fmt.Errorf("%w%s", InternalError, v)
}

// StatusFromError get a status from an error
func StatusFromError(err error) int {
	switch {
	case errors.Is(err, ValidationError):
		return http.StatusBadRequest
	case errors.Is(err, NotFoundError):
		return http.StatusNotFound
	case errors.Is(err, ConflictError):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
