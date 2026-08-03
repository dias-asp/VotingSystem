package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/service"
)

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const q = `INSERT INTO users (id, email, password_hash, role, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrUserExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `SELECT id, email, password_hash, role, created_at FROM users WHERE email = $1`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, service.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	const q = `SELECT id, email, password_hash, role, created_at FROM users WHERE id = $1`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, service.ErrUserNotFound
	}
	return u, err
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") || strings.Contains(msg, "unique constraint")
}
