package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/auth"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/pkg/utils"

	"github.com/google/uuid"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// AuthService 登录与 Token。
type AuthService struct {
	user        *repository.UserRepository
	audit       *AuditService
	jwtSecret   string
	expireHours int
}

func NewAuthService(user *repository.UserRepository, audit *AuditService, jwtSecret string, expireHours int) *AuthService {
	return &AuthService{user: user, audit: audit, jwtSecret: jwtSecret, expireHours: expireHours}
}

// Login 校验用户名密码并签发 JWT。
func (s *AuthService) Login(ctx context.Context, username, password, ip, userAgent string) (token string, u *model.User, err error) {
	u, err = s.user.GetByUsername(ctx, username)
	if err != nil {
		s.recordLoginFailed(ctx, nil, username, ip, userAgent, err)
		return "", nil, ErrInvalidCredentials
	}
	if u == nil || !utils.CheckPassword(u.PasswordHash, password) {
		s.recordLoginFailed(ctx, u, username, ip, userAgent, ErrInvalidCredentials)
		return "", nil, ErrInvalidCredentials
	}
	if u.Status != "" && u.Status != "active" {
		s.recordLoginFailed(ctx, u, username, ip, userAgent, ErrInvalidCredentials)
		return "", nil, ErrInvalidCredentials
	}
	if s.jwtSecret == "" {
		return "", u, errors.New("JWT secret not configured; set OPS_JWT_SECRET or jwt.secret")
	}
	token, err = auth.SignUserToken(s.jwtSecret, u.ID, u.Username, u.Role, s.expireHours)
	if err != nil {
		return "", nil, err
	}
	s.recordLoginSuccess(ctx, u, ip, userAgent)
	return token, u, nil
}

func (s *AuthService) recordLoginSuccess(ctx context.Context, u *model.User, ip, userAgent string) {
	if s == nil || s.audit == nil || u == nil {
		return
	}
	actorID := u.ID
	_ = s.audit.Record(ctx, AuditEntry{
		ActorID:   &actorID,
		ActorType: "user",
		Action:    "user.login",
		Resource:  "user",
		ResourceID: u.ID.String(),
		IP:        ip,
		UserAgent: userAgent,
		Status:    "success",
	})
}

func (s *AuthService) recordLoginFailed(ctx context.Context, u *model.User, username, ip, userAgent string, cause error) {
	if s == nil || s.audit == nil {
		return
	}
	var actorID *uuid.UUID
	if u != nil {
		actorID = &u.ID
	}
	details := map[string]string{"username": username}
	if cause != nil {
		details["reason"] = cause.Error()
	}
	_ = s.audit.Record(ctx, AuditEntry{
		ActorID:   actorID,
		ActorType: "user",
		Action:    "user.login_failed",
		Resource:  "user",
		Details:   details,
		IP:        ip,
		UserAgent: userAgent,
		Status:    "failed",
	})
}
