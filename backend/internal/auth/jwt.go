package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserClaims JWT 载荷（Subject 为用户 ID）。
type UserClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// SignUserToken 签发 HS256 Token。
func SignUserToken(secret string, userID uuid.UUID, username, role string, expireHours int) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is empty")
	}
	if expireHours <= 0 {
		expireHours = 24
	}
	now := time.Now()
	claims := UserClaims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseUserToken 解析并校验 Token。
func ParseUserToken(secret, raw string) (*UserClaims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	t, err := jwt.ParseWithClaims(raw, &UserClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := t.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
