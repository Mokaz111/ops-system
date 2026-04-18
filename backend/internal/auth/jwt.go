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
//
// 显式校验签名算法为 HS256，防止 algorithm confusion 攻击：攻击者把 header
// 中的 alg 改成 none / RS256 后，jwt 库可能用错误的算法（公钥当 HMAC secret）
// 验证，从而伪造 token。jwt/v5 默认拒绝 alg=none，但仍需在 keyFunc 里主动
// 断言算法以防其它变种攻击。
func ParseUserToken(secret, raw string) (*UserClaims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	t, err := jwt.ParseWithClaims(raw, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing algorithm")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := t.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
