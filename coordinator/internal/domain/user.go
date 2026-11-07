package domain

import (
	"coordinator/internal/utility"
	"encoding/base64"
	"strings"
	"time"
)

// Errors
var (
	ErrUsernameValidationFailed = utility.NewValidationError("username must consist only of english characters and arabic digits, and be at least 8 characters long")
	ErrEmailValidationFailed    = utility.NewValidationError("passed email is invalid")
	ErrPasswordValidationFailed = utility.NewValidationError("password must be at least 8 characters long, consist of english letters, digits, or allowed symbols ('.', '!', '?', '-', '_', '+', and '@')")
	ErrAvatarValidationFailed   = utility.NewValidationError("passed avatar image isn't base64 encoded")
	ErrRoleValidationFailed     = utility.NewValidationError("user role must be one of these: viewer, operator, and administrator")
)

// Username string with username validation
// String value must consist of english letters, digits, and some symbols
// String value must be at least 8 letters long
type Username string

// Email string with email validation
// String value must be a valid email
type Email string

// UserRole string with user role validation
// String value must be one of the valid user roles
type UserRole string

// Avatar string with avatar validation
// String value must be a valid base64 encoded string
type Avatar string

// Password string with username validation
// String value must consist of english letters, digits, and some symbols
// String value must be at least 8 letters long
type Password string

const (
	UserRoleViewer        UserRole = "viewer"
	UserRoleOperator      UserRole = "operator"
	UserRoleAdministrator UserRole = "administrator"
)

// NewUsername casts a string to an Username with validation
func NewUsername(v string) (Username, error) {
	if len(v) < 8 {
		return "", ErrUsernameValidationFailed
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == ' ':
		default:
			return "", ErrUsernameValidationFailed
		}
	}
	return Username(v), nil
}

// NewEmail casts a string to an Email with validation
func NewEmail(v string) (Email, error) {
	parts := strings.Split(v, "@")
	if len(parts) != 2 {
		return "", ErrEmailValidationFailed
	}
	userPart, domainPart := parts[0], parts[1]

	if len(userPart) == 0 || len(domainPart) == 0 {
		return "", ErrEmailValidationFailed
	}

	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '_' || c == '+' || c == '@':
		default:
			return "", ErrEmailValidationFailed
		}
	}

	if !strings.Contains(domainPart, ".") {
		return "", ErrEmailValidationFailed
	}

	return Email(v), nil
}

// NewPassword casts a string to an Password with validation
func NewPassword(v string) (Password, error) {
	if len(v) < 8 {
		return "", ErrPasswordValidationFailed
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '!' || c == '?' || c == '-' || c == '_' || c == '+' || c == '@':
		default:
			return "", ErrPasswordValidationFailed
		}
	}
	return Password(v), nil
}

// NewAvatar casts a string to an Avatar with validation
func NewAvatar(v string) (Avatar, error) {
	_, err := base64.RawStdEncoding.DecodeString(v)
	if err != nil {
		return "", ErrAvatarValidationFailed
	}
	return Avatar(v), nil
}

// NewUserRole casts a string to a UserRole with validation
func NewUserRole(v string) (UserRole, error) {
	role := UserRole(v)
	switch role {
	case UserRoleViewer, UserRoleOperator, UserRoleAdministrator:
		return role, nil
	default:
		return "", ErrRoleValidationFailed
	}
}

// PasswordCredentials a container for hash-salt pair
type PasswordCredentials struct {
	Hash []byte `json:"password_hash" validate:"required"`
	Salt []byte `json:"salt" validate:"required"`
}

// User account information
type User struct {
	ID        int                 `json:"user_id" validate:"required"`
	Username  Username            `json:"username" validate:"required"`
	Email     Email               `json:"email" validate:"required"`
	Password  PasswordCredentials `json:"password" validate:"required"`
	Role      UserRole            `json:"user_role" validate:"required"`
	Avatar    Avatar              `json:"avatar" validate:"required"`
	CreatedAt time.Time           `json:"created_at" validate:"required"`
}

// UserUpdate optional fields to be updated
type UserUpdate struct {
	Username *Username            `json:"username"`
	Email    *Email               `json:"email"`
	Password *PasswordCredentials `json:"password"`
	Avatar   *Avatar              `json:"avatar"`
	Role     *UserRole            `json:"user_role"`
}
