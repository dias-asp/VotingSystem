package postgres

import (
	"context"
	"database/sql"
	"errors"

	"voting_service/internal/domain"
	"voting_service/internal/service"
)

var _ service.PollRepository = (*PollRepository)(nil)

type PollRepository struct {
	db *sql.DB
}

func NewPollRepository(db *sql.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) CreatePoll(ctx context.Context, p domain.Poll) error {
	const q = `INSERT INTO polls (id, title, description, status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Title, p.Description, string(p.Status), p.CreatedAt)
	if err != nil && isUniqueViolation(err) {
		return service.ErrPollExists
	}
	return err
}

func (r *PollRepository) GetPoll(ctx context.Context, id string) (domain.Poll, error) {
	const q = `SELECT id, title, description, status, created_at FROM polls WHERE id = $1`
	var (
		p  domain.Poll
		st string
	)
	err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Title, &p.Description, &st, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Poll{}, service.ErrPollNotFound
	}
	if err != nil {
		return domain.Poll{}, err
	}
	p.Status = domain.PollStatus(st)
	return p, nil
}

func (r *PollRepository) ListPolls(ctx context.Context) ([]domain.Poll, error) {
	const q = `SELECT id, title, description, status, created_at FROM polls ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Poll
	for rows.Next() {
		var (
			p  domain.Poll
			st string
		)
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &st, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Status = domain.PollStatus(st)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PollRepository) UpdatePollStatus(ctx context.Context, id string, status domain.PollStatus) error {
	const q = `UPDATE polls SET status = $2 WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, string(status))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrPollNotFound
	}
	return nil
}

func (r *PollRepository) AddCandidate(ctx context.Context, c domain.Candidate) error {
	const q = `INSERT INTO candidates (poll_id, id, name) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, c.PollID, c.ID, c.Name)
	if err != nil && isUniqueViolation(err) {
		return service.ErrCandidateExists
	}
	return err
}

func (r *PollRepository) ListCandidates(ctx context.Context, pollID string) ([]domain.Candidate, error) {
	const q = `SELECT poll_id, id, name FROM candidates WHERE poll_id = $1 ORDER BY name`
	rows, err := r.db.QueryContext(ctx, q, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Candidate
	for rows.Next() {
		var c domain.Candidate
		if err := rows.Scan(&c.PollID, &c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
