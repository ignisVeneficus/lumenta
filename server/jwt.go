package server

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
)

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   uint64 `json:"uid"`
	Role     string `json:"role"`
	UserName string `json:"username"`
}

type JWTService struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

type Option func(*JWTService)

// WithTTL configures token lifetime.
func WithTTL(ttl time.Duration) Option {
	return func(s *JWTService) { s.ttl = ttl }
}

// WithIssuer configures token issuer (iss).
func WithIssuer(issuer string) Option {
	return func(s *JWTService) { s.issuer = issuer }
}

func NewJWTService(secret string, opts ...Option) *JWTService {
	if secret == "" {
		return nil
	}
	s := &JWTService{
		secret: []byte(secret),
		ttl:    24 * time.Hour,
		issuer: "",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *JWTService) Verify(tokenStr string) (*JWTClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	token, err := parser.ParseWithClaims(
		tokenStr,
		&JWTClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return s.secret, nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *JWTService) Issue(acl authData.ACLContext) (string, error) {
	now := time.Now()

	rc := jwt.RegisteredClaims{
		Issuer:    s.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	claims := JWTClaims{
		RegisteredClaims: rc,
		UserName:         *acl.UserName,
		Role:             string(acl.Role),
	}
	if acl.UserID != nil {
		claims.UserID = *acl.UserID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}
