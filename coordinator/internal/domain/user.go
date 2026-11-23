package domain

import (
	"context"
	"slices"
	"time"
)

// IUserRepository interface specifying methods that *UserRepository must implement
type IUserRepository interface {
	SaveUser(ctx context.Context, user *User) error
	GetUsers(ctx context.Context) ([]User, error)
	GetUserByID(ctx context.Context, id int) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	DeleteUserByID(ctx context.Context, id int) error
}

// User roles
const (
	UserRoleViewer        string = "viewer"
	UserRoleOperator      string = "operator"
	UserRoleAdministrator string = "administrator"
)

// PasswordCredentials a container for hash-salt pair
type PasswordCredentials struct {
	Hash []byte `json:"hash" form:"hash" binding:"required"`
	Salt []byte `json:"salt" form:"salt" binding:"required"`
}

// User account information
type User struct {
	ID        int                 `json:"user_id" form:"user_id" binding:"omitempty,min=0"`
	Username  string              `json:"username" form:"username" binding:"required,min=8"`
	Email     string              `json:"email" form:"email" binding:"required,email"`
	Password  PasswordCredentials `json:"password" form:"password" binding:"required"`
	Role      string              `json:"user_role" form:"user_role" binding:"required,oneof=viewer operator administrator"`
	Avatar    string              `json:"avatar" form:"avatar" binding:"omitempty,base64"`
	CreatedAt time.Time           `json:"created_at" form:"created_at" binding:"required"`
}

// Clone creates a deep-copy of the calling PasswordCredentials
func (pc PasswordCredentials) Clone() PasswordCredentials {
	return PasswordCredentials{
		Hash: slices.Clone(pc.Hash),
		Salt: slices.Clone(pc.Salt),
	}
}

// Clone creates a deep-copy of the calling User
func (user User) Clone() User {
	clone := user
	clone.Password = user.Password.Clone()
	return clone
}
