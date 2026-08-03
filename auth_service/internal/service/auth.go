package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"auth_service/internal/domain"
)

var (
	ErrNotImplemented     = errors.New("not implemented")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrTokenRevoked       = errors.New("refresh token revoked or expired")
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, token domain.RefreshToken) error
	Get(ctx context.Context, id string) (domain.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type TokenIssuer interface {
	Issue(claims domain.Claims) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (domain.Claims, error)
}

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) bool
}

type Auth struct {
	users      UserRepository
	refresh    RefreshTokenRepository
	issuer     TokenIssuer
	hasher     PasswordHasher
	accessTTL  time.Duration
	refreshTTL time.Duration
	adminEmail string
	now        func() time.Time
}

func NewAuth(
	users UserRepository,
	refresh RefreshTokenRepository,
	issuer TokenIssuer,
	hasher PasswordHasher,
	accessTTL, refreshTTL time.Duration,
	adminEmail string,
) *Auth {
	return &Auth{
		users:      users,
		refresh:    refresh,
		issuer:     issuer,
		hasher:     hasher,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		adminEmail: strings.ToLower(strings.TrimSpace(adminEmail)),
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (a *Auth) Register(ctx context.Context, email, password string) (domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 {
		return domain.User{}, ErrInvalidInput
	}

	if _, err := a.users.GetByEmail(ctx, email); err == nil {
		return domain.User{}, ErrUserExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return domain.User{}, err
	}

	hash, err := a.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}

	role := domain.RoleUser
	if a.adminEmail != "" && email == a.adminEmail {
		role = domain.RoleAdmin
	}

	user := domain.User{
		ID:           newUUID(),
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    a.now(),
	}
	if err := a.users.Create(ctx, user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (a *Auth) Login(ctx context.Context, email, password string) (domain.TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return domain.TokenPair{}, ErrInvalidCredentials
		}
		return domain.TokenPair{}, err
	}
	if !a.hasher.Verify(password, user.PasswordHash) {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	return a.issuePair(ctx, user)
}

func (a *Auth) Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return domain.TokenPair{}, ErrInvalidInput
	}

	token, err := a.refresh.Get(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return domain.TokenPair{}, ErrTokenRevoked
		}
		return domain.TokenPair{}, err
	}
	if token.Revoked || a.now().After(token.ExpiresAt) {
		return domain.TokenPair{}, ErrTokenRevoked
	}

	user, err := a.users.GetByID(ctx, token.UserID)
	if err != nil {
		return domain.TokenPair{}, err
	}

	if err := a.refresh.Revoke(ctx, token.ID); err != nil {
		return domain.TokenPair{}, err
	}
	return a.issuePair(ctx, user)
}

func (a *Auth) Logout(ctx context.Context, refreshTokenID string) error {
	refreshTokenID = strings.TrimSpace(refreshTokenID)
	if refreshTokenID == "" {
		return ErrInvalidInput
	}
	if err := a.refresh.Revoke(ctx, refreshTokenID); err != nil && !errors.Is(err, ErrTokenNotFound) {
		return err
	}
	return nil
}

func (a *Auth) issuePair(ctx context.Context, user domain.User) (domain.TokenPair, error) {
	now := a.now()
	role := user.Role
	if role == "" {
		role = domain.RoleUser
	}
	accessClaims := domain.Claims{
		Subject:   user.ID,
		Email:     user.Email,
		Role:      role,
		TokenID:   newUUID(),
		IssuedAt:  now,
		ExpiresAt: now.Add(a.accessTTL),
	}
	access, err := a.issuer.Issue(accessClaims)
	if err != nil {
		return domain.TokenPair{}, err
	}

	refresh := domain.RefreshToken{
		ID:        newUUID(),
		UserID:    user.ID,
		ExpiresAt: now.Add(a.refreshTTL),
		CreatedAt: now,
	}
	if err := a.refresh.Save(ctx, refresh); err != nil {
		return domain.TokenPair{}, err
	}

	return domain.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh.ID,
		ExpiresIn:    int(a.accessTTL.Seconds()),
	}, nil
}
