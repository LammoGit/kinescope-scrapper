package domain

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	SALT_SIZE      = 12
	ARGON2_TIME    = 1
	ARGON2_MEMORY  = 64 * 1024
	ARGON2_THREADS = 4
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash []byte    `json:"password_hash"`
	Salt         []byte    `json:"salt"`
	Avatar       string    `json:"avatar"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserTokens struct {
	AcessToken   string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserRole string

const (
	UserRoleViewer        UserRole = "viewer"
	UserRoleOperator      UserRole = "operator"
	UserRoleAdministrator UserRole = "administrator"
)

func GenerateUserTokens(user User) (UserTokens, error) {
	return UserTokens{}, nil
}

func (u *User) VerifyPassword(password string) bool {
	hash := argon2.IDKey(
		[]byte(password),
		u.Salt,
		ARGON2_TIME,
		ARGON2_MEMORY,
		ARGON2_THREADS,
		uint32(len(u.PasswordHash)),
	)
	return bytes.Equal(hash, u.PasswordHash)
}

func (u *User) SetUsername(username string) error {
	u.Username = username
	return nil
}

func (u *User) SetEmail(email string) error {
	firstCheck := strings.SplitN(email, "@", 2)
	if len(firstCheck) != 2 {
		return errors.New("Email is invalid")
	}

	secondCheck := strings.SplitN(email, ".", 2)
	if len(secondCheck) != 2 {
		return errors.New("Email is invalid")
	}

	u.Email = email
	return nil
}

func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return errors.New("Password is too short")
	}

	u.Salt = make([]byte, SALT_SIZE)
	rand.Reader.Read(u.Salt)

	u.PasswordHash = argon2.IDKey(
		[]byte(password),
		u.Salt,
		ARGON2_TIME,
		ARGON2_MEMORY,
		ARGON2_THREADS,
		uint32(len(u.PasswordHash)),
	)

	return nil
}

func (u *User) SetAvatar(avatar string) error {
	_, err := base64.RawStdEncoding.DecodeString(avatar)
	if err != nil {
		return errors.New("Passed avatar image isn't base64 encoded")
	}
	u.Avatar = avatar
	return nil
}

func (u *User) SetRole(role string) error {
	switch UserRole(role) {
	case UserRoleViewer, UserRoleAdministrator, UserRoleOperator:
	default:
		return errors.New("Invalid role")
	}
	u.Role = UserRole(role)
	return nil
}
