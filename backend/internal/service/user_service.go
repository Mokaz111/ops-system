package service

import (
	"context"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/pkg/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUsernameRequired    = errors.New("username required")
	ErrUsernameExists      = errors.New("username already exists")
	ErrBootstrapNotAllowed = errors.New("bootstrap only allowed when no users exist")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
)

// CreateUserRequest 创建用户。
type CreateUserRequest struct {
	Username string
	Password string
	Email    string
	Phone    string
	Role     string
	Status   string
}

// UpdateUserRequest 更新用户。
type UpdateUserRequest struct {
	Email    *string
	Phone    *string
	Role     *string
	Status   *string
	Password *string // 明文新密码，非空则更新
}

// UserService 用户业务。
type UserService struct {
	user    *repository.UserRepository
	members *repository.WorkspaceMemberRepository
}

func NewUserService(user *repository.UserRepository, members *repository.WorkspaceMemberRepository) *UserService {
	return &UserService{user: user, members: members}
}

// Bootstrap 首个管理员（仅当用户表为空）。
func (s *UserService) Bootstrap(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return nil, ErrUsernameRequired
	}
	if len(req.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	req.Role = "admin"
	if req.Status == "" {
		req.Status = "active"
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		Email:        strings.TrimSpace(req.Email),
		Phone:        strings.TrimSpace(req.Phone),
		Role:         req.Role,
		Status:       req.Status,
	}
	if err := s.user.CreateFirstUser(ctx, u); err != nil {
		if errors.Is(err, repository.ErrFirstUserAlreadyExists) {
			return nil, ErrBootstrapNotAllowed
		}
		return nil, err
	}
	zap.L().Warn("bootstrap_admin_created",
		zap.String("user_id", u.ID.String()),
		zap.String("username", u.Username),
		zap.String("email", u.Email),
	)
	return u, nil
}

// Create 创建用户（调用方负责鉴权）。
func (s *UserService) Create(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	return s.create(ctx, req)
}

func (s *UserService) create(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return nil, ErrUsernameRequired
	}
	if len(req.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	if exist, err := s.user.GetByUsername(ctx, req.Username); err != nil {
		return nil, err
	} else if exist != nil {
		return nil, ErrUsernameExists
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		Email:        strings.TrimSpace(req.Email),
		Phone:        strings.TrimSpace(req.Phone),
		Role:         req.Role,
		Status:       req.Status,
	}
	if err := s.user.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Get 按 ID。
func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := s.user.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// Update 更新。
func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*model.User, error) {
	u, err := s.user.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	if req.Email != nil {
		u.Email = strings.TrimSpace(*req.Email)
	}
	if req.Phone != nil {
		u.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Role != nil && *req.Role != "" {
		u.Role = *req.Role
	}
	if req.Status != nil && *req.Status != "" {
		u.Status = *req.Status
	}
	if req.Password != nil && *req.Password != "" {
		if len(*req.Password) < 8 {
			return nil, ErrPasswordTooShort
		}
		hash, err := utils.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = hash
	}
	if err := s.user.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Delete 删除。
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	u, err := s.user.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	return s.user.Delete(ctx, id)
}

// CanAccessWorkspace 检查用户是否有权访问指定工作空间。
// action: read / write
func (s *UserService) CanAccessWorkspace(ctx context.Context, userID uuid.UUID, workspaceID uuid.UUID, action string) (bool, error) {
	if s == nil || s.user == nil {
		return false, nil
	}
	u, err := s.user.GetByID(ctx, userID)
	if err != nil || u == nil {
		return false, err
	}
	if u.Role == "admin" {
		return true, nil
	}
	if s.members == nil {
		return false, nil
	}
	m, err := s.members.GetByUserAndWorkspace(ctx, userID, workspaceID)
	if err != nil || m == nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "write":
		return m.Role == "admin", nil
	default:
		return true, nil
	}
}

// ListUserWorkspaces 返回用户所属工作空间 ID 列表。
func (s *UserService) ListUserWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if s == nil || s.members == nil {
		return nil, nil
	}
	memberships, err := s.members.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(memberships))
	for _, m := range memberships {
		out = append(out, m.WorkspaceID)
	}
	return out, nil
}

// List 分页筛选。
func (s *UserService) List(ctx context.Context, page, pageSize int, deptID, tenantID *uuid.UUID, role, status, keyword string) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.user.List(ctx, repository.UserListFilter{
		DeptID:   deptID,
		TenantID: tenantID,
		Role:     role,
		Status:   status,
		Keyword:  keyword,
		Offset:   offset,
		Limit:    pageSize,
	})
}
