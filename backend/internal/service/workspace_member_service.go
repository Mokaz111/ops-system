package service

import (
	"context"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrWorkspaceMemberNotFound = errors.New("workspace member not found")
	ErrWorkspaceMemberExists   = errors.New("workspace member already exists")
	ErrInvalidWorkspaceRole    = errors.New("invalid workspace role")
)

var allowedWorkspaceRoles = map[string]struct{}{
	"admin":  {},
	"member": {},
	"viewer": {},
}

// UserWorkspaceMembership 用户所属工作空间及角色。
type UserWorkspaceMembership struct {
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	Role          string    `json:"role"`
	WorkspaceName string    `json:"workspace_name"`
}

// WorkspaceMemberService 工作空间成员业务。
type WorkspaceMemberService struct {
	members    *repository.WorkspaceMemberRepository
	workspaces *repository.WorkspaceRepository
}

func NewWorkspaceMemberService(
	members *repository.WorkspaceMemberRepository,
	workspaces *repository.WorkspaceRepository,
) *WorkspaceMemberService {
	return &WorkspaceMemberService{members: members, workspaces: workspaces}
}

func validWorkspaceRole(role string) bool {
	_, ok := allowedWorkspaceRoles[strings.ToLower(strings.TrimSpace(role))]
	return ok
}

// AddMember 添加工作空间成员。
func (s *WorkspaceMemberService) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) (*model.WorkspaceMember, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = "member"
	}
	if !validWorkspaceRole(role) {
		return nil, ErrInvalidWorkspaceRole
	}
	if s.workspaces != nil {
		w, err := s.workspaces.GetByID(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if w == nil {
			return nil, ErrWorkspaceNotFound
		}
	}
	existing, err := s.members.GetByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrWorkspaceMemberExists
	}
	m := &model.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	}
	if err := s.members.Create(ctx, m); err != nil {
		return nil, errors.Wrap(err, "create workspace member")
	}
	return m, nil
}

// RemoveMember 移除工作空间成员。
func (s *WorkspaceMemberService) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	m, err := s.members.GetByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrWorkspaceMemberNotFound
	}
	return s.members.Delete(ctx, m.ID)
}

// UpdateRole 更新成员角色。
func (s *WorkspaceMemberService) UpdateRole(ctx context.Context, workspaceID, userID uuid.UUID, role string) (*model.WorkspaceMember, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if !validWorkspaceRole(role) {
		return nil, ErrInvalidWorkspaceRole
	}
	m, err := s.members.GetByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrWorkspaceMemberNotFound
	}
	m.Role = role
	if err := s.members.Update(ctx, m); err != nil {
		return nil, errors.Wrap(err, "update workspace member")
	}
	return m, nil
}

// ListMembers 列出工作空间成员。
func (s *WorkspaceMemberService) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]model.WorkspaceMember, error) {
	return s.members.ListByWorkspace(ctx, workspaceID)
}

// ListUserWorkspaces 列出用户所属工作空间及角色。
func (s *WorkspaceMemberService) ListUserWorkspaces(ctx context.Context, userID uuid.UUID) ([]UserWorkspaceMembership, error) {
	memberships, err := s.members.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]UserWorkspaceMembership, 0, len(memberships))
	for _, m := range memberships {
		item := UserWorkspaceMembership{
			WorkspaceID: m.WorkspaceID,
			Role:        m.Role,
		}
		if s.workspaces != nil {
			w, wErr := s.workspaces.GetByID(ctx, m.WorkspaceID)
			if wErr == nil && w != nil {
				item.WorkspaceName = w.WorkspaceName
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// GetMembership 查询用户在指定工作空间的成员关系。
func (s *WorkspaceMemberService) GetMembership(ctx context.Context, userID, workspaceID uuid.UUID) (*model.WorkspaceMember, error) {
	return s.members.GetByUserAndWorkspace(ctx, userID, workspaceID)
}
