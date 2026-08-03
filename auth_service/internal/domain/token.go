package domain

import "time"

type Claims struct {
	Subject   string
	Email     string
	Role      string
	TokenID   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type RefreshToken struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}
