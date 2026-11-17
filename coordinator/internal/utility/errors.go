package utility

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// Error kinds
var (
	ValidationError   = errors.New("")
	NotFoundError     = errors.New("")
	ConflictError     = errors.New("")
	InternalError     = errors.New("")
	ForbiddenError    = errors.New("")
	UnauthorizedError = errors.New("")
)

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

// NewForbiddenError returns a new forbidden error with the given text
func NewForbiddenError(v string) error {
	return fmt.Errorf("%w%s", ForbiddenError, v)
}

// NewUnauthorizedError returns a new unauthorized error with the given text
func NewUnauthorizedError(v string) error {
	return fmt.Errorf("%w%s", UnauthorizedError, v)
}

// StatusFromError gets a status from an error
func StatusFromError(err error) int {
	switch {
	case errors.Is(err, ValidationError):
		return http.StatusBadRequest
	case errors.Is(err, NotFoundError):
		return http.StatusNotFound
	case errors.Is(err, ConflictError):
		return http.StatusConflict
	case errors.Is(err, ForbiddenError):
		return http.StatusForbidden
	case errors.Is(err, UnauthorizedError):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// SendError sends an error with the HTTP status matching its kind
func SendError(c *gin.Context, err error) {
	status := StatusFromError(err)

	// Log every error response, client errors are worth a warning and server errors deserve a full error entry
	attrs := []any{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", status,
		"error", err.Error(),
	}
	if status >= 500 {
		slog.Error("Request failed", attrs...)
	} else {
		slog.Warn("Request failed", attrs...)
	}

	c.AbortWithStatusJSON(status, gin.H{
		"errors": []string{
			err.Error(),
		},
		"status": "fail",
	})
}

// trans holds the English translator registered on the gin validator
var trans ut.Translator

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		panic("gin validator is not *validator.Validate")
	}

	// Report fields by their wire name instead of their Go name inside translated messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range []string{"json", "form"} {
			name, _, _ := strings.Cut(fld.Tag.Get(tag), ",")
			if name != "" && name != "-" {
				return name
			}
		}
		return fld.Name
	})

	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ = uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(v, trans)
}

// SendBindingError sends every field error produced by the validator
func SendBindingError(c *gin.Context, err error) {
	slog.Warn("Request validation failed",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"error", err.Error(),
	)

	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"errors": []string{"Invalid request payload syntax"},
			"status": "fail",
		})
		return
	}

	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, fmt.Sprintf("%s: %s", e.Field(), e.Translate(trans)))
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"errors": messages,
		"status": "fail",
	})
}
