package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/pkg/utils"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const apiTokenPrefix = "ops_"

var (
	ErrAPITokenNotFound   = errors.New("api token not found")
	ErrAPITokenNameReq    = errors.New("api token name required")
	ErrAPITokenInvalid    = errors.New("invalid api token")
	ErrAPITokenExpired    = errors.New("api token expired")
	ErrAPITokenScope      = errors.New("invalid api token scope")
)

var allowedAPITokenScopes = map[string]struct{}{
	"read":       {},
	"read_write": {},
}

// CreateAPITokenRequest 创建 Token 请求。
type CreateAPITokenRequest struct {
	Name      string
	Scope     string
	ExpiresAt *time.Time
}

// CreateAPITokenResult 创建 Token 结果（明文仅返回一次）。
type CreateAPITokenResult struct {
	Token *model.APIToken `json:"token"`
	Plain string          `json:"plain_text"`
}

// APITokenService API Token 业务。
type APITokenService struct {
	repo *repository.APITokenRepository
	user *repository.UserRepository
}

func NewAPITokenService(repo *repository.APITokenRepository, user *repository.UserRepository) *APITokenService {
	return &APITokenService{repo: repo, user: user}
}

func hashAPIToken(plain string) (string, error) {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:]), nil
}

func tokenPrefixFromPlain(plain string) string {
	if len(plain) <= 16 {
		return plain
	}
	return plain[:16]
}

// Create 创建 Token，返回明文（仅一次）。
func (s *APITokenService) Create(ctx context.Context, userID uuid.UUID, req *CreateAPITokenRequest) (*CreateAPITokenResult, error) {
	if req == nil || strings.TrimSpace(req.Name) == "" {
		return nil, ErrAPITokenNameReq
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "read_write"
	}
	if _, ok := allowedAPITokenScopes[scope]; !ok {
		return nil, ErrAPITokenScope
	}
	if s.user != nil {
		u, err := s.user.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, ErrUserNotFound
		}
	}
	randPart, err := utils.RandomHex(24)
	if err != nil {
		return nil, errors.Wrap(err, "generate api token")
	}
	plain := apiTokenPrefix + randPart
	hash, err := hashAPIToken(plain)
	if err != nil {
		return nil, err
	}
	t := &model.APIToken{
		UserID:      userID,
		Name:        strings.TrimSpace(req.Name),
		TokenPrefix: tokenPrefixFromPlain(plain),
		TokenHash:   hash,
		Scope:       scope,
		ExpiresAt:   req.ExpiresAt,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, errors.Wrap(err, "create api token")
	}
	return &CreateAPITokenResult{Token: t, Plain: plain}, nil
}

// List 列出用户的 Token。
func (s *APITokenService) List(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Revoke 撤销 Token。
func (s *APITokenService) Revoke(ctx context.Context, userID, tokenID uuid.UUID) error {
	t, err := s.repo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrAPITokenNotFound
	}
	if t.UserID != userID {
		return ErrAPITokenNotFound
	}
	return s.repo.Delete(ctx, tokenID)
}

// Authenticate 校验 Bearer Token 并返回关联用户。
func (s *APITokenService) Authenticate(ctx context.Context, plain string) (*model.User, *model.APIToken, error) {
	plain = strings.TrimSpace(plain)
	if !strings.HasPrefix(plain, apiTokenPrefix) {
		return nil, nil, ErrAPITokenInvalid
	}
	prefix := tokenPrefixFromPlain(plain)
	t, err := s.repo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, ErrAPITokenInvalid
	}
	hash, err := hashAPIToken(plain)
	if err != nil {
		return nil, nil, err
	}
	if t.TokenHash != hash {
		return nil, nil, ErrAPITokenInvalid
	}
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return nil, nil, ErrAPITokenExpired
	}
	u, err := s.user.GetByID(ctx, t.UserID)
	if err != nil {
		return nil, nil, err
	}
	if u == nil || u.Status != "" && u.Status != "active" {
		return nil, nil, ErrAPITokenInvalid
	}
	_ = s.repo.UpdateLastUsed(ctx, t.ID, time.Now())
	return u, t, nil
}
