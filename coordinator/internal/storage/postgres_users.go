//go:build postgres && !sqlite

package storage

import (
	"context"
	"time"

	"coordinator/internal/domain"
	"coordinator/internal/repos"
)

const userColumns = `
	id, username, email, password_hash, password_salt, role, avatar, created_at`

func scanUser(s rowScanner) (domain.User, error) {
	var u domain.User
	err := s.Scan(
		&u.ID, &u.Username, &u.Email, &u.Password.Hash, &u.Password.Salt,
		&u.Role, &u.Avatar, &u.CreatedAt,
	)
	return u, err
}

func (p *Postgres) GetUserByID(ctx context.Context, id int) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	user, err := scanUser(p.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	return user, withNotFound(err, errUserNotFound)
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	user, err := scanUser(p.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	return user, withNotFound(err, repos.ErrUserNotFoundByEmail)
}

func (p *Postgres) CreateUser(ctx context.Context, username, email string, creds domain.PasswordCredentials) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		Username:  username,
		Email:     email,
		Password:  creds,
		Role:      domain.UserRoleViewer,
		Avatar:    "",
		CreatedAt: time.Now(),
	}

	err := p.db.QueryRowContext(ctx, `
		INSERT INTO users (
			username, email, password_hash, password_salt, role, avatar, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		user.Username, user.Email, user.Password.Hash, user.Password.Salt,
		user.Role, user.Avatar, user.CreatedAt,
	).Scan(&user.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, repos.ErrEmailIsTaken
		}
		return domain.User{}, err
	}

	return user, nil
}

func (p *Postgres) ReplaceUser(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE users SET
			username = $2, email = $3, password_hash = $4, password_salt = $5,
			role = $6, avatar = $7, created_at = $8
		WHERE id = $1`,
		user.ID, user.Username, user.Email, user.Password.Hash, user.Password.Salt,
		user.Role, user.Avatar, user.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repos.ErrEmailIsTaken
		}
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errUserNotFound
	}
	return nil
}

func (p *Postgres) DeleteUserByID(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errUserNotFound
	}
	return nil
}
