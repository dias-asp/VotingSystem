package auth

import (
	"context"
	"errors"
)

type Claims struct {
	Subject   string
	Email     string
	Role      string
	TokenID   string
	IssuedAt  int64
	ExpiresAt int64
}

type ctxKey struct{}

var ErrNoClaims = errors.New("no claims in context")

func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

func FromContext(ctx context.Context) (Claims, error) {
	v, ok := ctx.Value(ctxKey{}).(Claims)
	if !ok {
		return Claims{}, ErrNoClaims
	}
	return v, nil
}
