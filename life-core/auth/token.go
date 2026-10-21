package auth

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuer = "life-core"

var ErrInvalidTokenConfig = errors.New("auth: jwt secret and positive ttl are required")

// TokenManager 负责签发和校验登录 Token。
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type userClaims struct {
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if len(strings.TrimSpace(secret)) < 32 || ttl <= 0 {
		return nil, ErrInvalidTokenConfig
	}
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

// Issue 为用户签发 Token，并返回过期时间。
func (m *TokenManager) Issue(userID uint64) (string, time.Time, error) {
	if m == nil || userID == 0 {
		return "", time.Time{}, ErrInvalidTokenConfig
	}
	now := m.now()
	expiresAt := now.Add(m.ttl)
	claims := userClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		Subject:   strconv.FormatUint(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expiresAt, err
}

// Parse 校验 Token，并返回其中的用户 ID。
func (m *TokenManager) Parse(raw string) (uint64, error) {
	if m == nil {
		return 0, ErrInvalidTokenConfig
	}
	claims := new(userClaims)
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("auth: unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer(tokenIssuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(m.now))
	if err != nil || token == nil || !token.Valid {
		return 0, errors.New("auth: invalid token")
	}
	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || userID == 0 {
		return 0, errors.New("auth: invalid token subject")
	}
	return userID, nil
}
