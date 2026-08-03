package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"anonym_service/internal/service"
)

var _ service.MappingRepository = (*MappingRepository)(nil)

type MappingRepository struct {
	db *sql.DB
}

func NewMappingRepository(db *sql.DB) *MappingRepository {
	return &MappingRepository{db: db}
}

func (r *MappingRepository) GetOrCreate(ctx context.Context, userID string) (string, error) {
	const q = `
        INSERT INTO user_mdm_mapping (user_id, mdm_id)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
        RETURNING mdm_id`

	candidate := newUUID()
	var mdmID string
	if err := r.db.QueryRowContext(ctx, q, userID, candidate).Scan(&mdmID); err != nil {
		return "", err
	}
	return mdmID, nil
}

func (r *MappingRepository) Get(ctx context.Context, userID string) (string, bool, error) {
	const q = `SELECT mdm_id FROM user_mdm_mapping WHERE user_id = $1`
	var mdmID string
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&mdmID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return mdmID, true, nil
}

func newUUID() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("uuid: read random: %w", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
