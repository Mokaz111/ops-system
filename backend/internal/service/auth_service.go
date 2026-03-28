package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/auth"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/pkg/utils"

)

var ErrInvalidCredentials = errors.New("invalid username or password")

// AuthService 登录与 Token。
type AuthService struct {
	user        *repository.UserRepository
	jwtSecret   string
	expireHours int
}

func NewAuthService(user *repository.UserRepository, jwtSecret string, expireHours int) *AuthService {
	return &AuthService{user: user, jwtSecret: jwtSecret, expireHours: expireHours}
}

// Login 校验用户名密码并签发 JWT。
func (s *AuthService) Login(ctx context.Context, username, password string) (token string, u *model.User, err error) {
	u, err = s.user.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, err
	}
	if u == nil || !utils.CheckPassword(u.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}
	if u.Status != "" && u.Status != "active" {
		return "", nil, ErrInvalidCredentials
	}
	if s.jwtSecret == "" {
		return "", u, errors.New("JWT secret not configured; set OPS_JWT_SECRET or jwt.secret")
	}
	token, err = auth.SignUserToken(s.jwtSecret, u.ID, u.Username, u.Role, s.expireHours)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}
