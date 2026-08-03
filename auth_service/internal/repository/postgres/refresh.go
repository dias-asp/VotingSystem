package postgres

import (
	"context"
	"database/sql"
	"errors"

	"auth_service/internal/domain"
	"auth_service/internal/service"
)

var _ service.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	const q = `INSERT INTO refresh_tokens (id, user_id, expires_at, revoked, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, token.ID, token.UserID, token.ExpiresAt, token.Revoked, token.CreatedAt)
	return err
}

func (r *RefreshTokenRepository) Get(ctx context.Context, id string) (domain.RefreshToken, error) {
	const q = `SELECT id, user_id, expires_at, revoked, created_at FROM refresh_tokens WHERE id = $1`
	var t domain.RefreshToken
	err := r.db.QueryRowContext(ctx, q, id).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RefreshToken{}, service.ErrTokenNotFound
	}
	return t, err
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return service.ErrTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1 AND NOT revoked`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}
