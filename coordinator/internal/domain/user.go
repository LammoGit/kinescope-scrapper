package domain

import (
	"time"
)

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
	ID        int                 `json:"user_id" form:"user_id" binding:"min=0"`
	Username  string              `json:"username" form:"username" binding:"required,min=8"`
	Email     string              `json:"email" form:"email" binding:"required,email"`
	Password  PasswordCredentials `json:"password" form:"password" binding:"required"`
	Role      string              `json:"user_role" form:"user_role" binding:"required,oneof=viewer operator administrator"`
	Avatar    string              `json:"avatar" form:"avatar" binding:"omitempty,base64"`
	CreatedAt time.Time           `json:"created_at" form:"created_at" binding:"required"`
}

// UserUpdate optional fields to be updated
type UserUpdate struct {
	Username *string              `json:"username" form:"username" binding:"omitempty,min=8"`
	Email    *string              `json:"email" form:"email" binding:"omitempty,email"`
	Password *PasswordCredentials `json:"password" form:"password"`
	Avatar   *string              `json:"avatar" form:"avatar" binding:"omitempty,base64"`
	Role     *string              `json:"user_role" form:"user_role" binding:"omitempty,oneof=viewer operator administrator"`
}

// Update applies the non-nil fields of an update to a user
func (u *User) Update(uu UserUpdate) {
	if uu.Username != nil {
		u.Username = *uu.Username
	}

	if uu.Email != nil {
		u.Email = *uu.Email
	}

	if uu.Password != nil {
		u.Password = *uu.Password
	}

	if uu.Avatar != nil {
		u.Avatar = *uu.Avatar
	}

	if uu.Role != nil {
		u.Role = *uu.Role
	}
}
