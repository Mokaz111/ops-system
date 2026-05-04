package service

import (
	"context"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

type AuthzService struct {
	users   *repository.UserRepository
	members *repository.TenantMemberRepository
}

func NewAuthzService(users *repository.UserRepository, members *repository.TenantMemberRepository) *AuthzService {
	return &AuthzService{users: users, members: members}
}

func (s *AuthzService) CanAccessTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, action string) (bool, error) {
	if s == nil || s.users == nil {
		return false, nil
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil || u == nil {
		return false, err
	}
	if u.Role == "admin" || u.Role == model.PlatformRoleAdmin {
		return true, nil
	}
	if u.TenantID != nil && *u.TenantID == tenantID {
		return true, nil
	}
	if s.members == nil {
		return false, nil
	}
	m, err := s.members.GetActive(ctx, tenantID, userID)
	if err != nil || m == nil {
		return false, err
	}
	switch action {
	case "read":
		return true, nil
	case "write":
		return m.Role == model.TenantRoleAdmin || m.Role == model.TenantRoleEditor || m.Role == model.TenantRoleAlert, nil
	case "admin":
		return m.Role == model.TenantRoleAdmin, nil
	default:
		return true, nil
	}
}
