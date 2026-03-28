package service

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 常见业务错误（handler 映射 HTTP 状态码）。
var (
	ErrDepartmentNotFound  = errors.New("department not found")
	ErrDepartmentHasChild  = errors.New("department has child departments")
	ErrDepartmentHasTenant = errors.New("department is bound to tenant")
	ErrParentNotFound      = errors.New("parent department not found")
	ErrInvalidPagination   = errors.New("invalid page or page_size")
	ErrDeptNameRequired    = errors.New("dept_name required")
	ErrInvalidParentID     = errors.New("invalid parent_id")
	ErrParentSelf          = errors.New("parent cannot be self")
)

// DepartmentTreeNode 部门树节点。
type DepartmentTreeNode struct {
	ID           uuid.UUID            `json:"id"`
	DeptName     string               `json:"dept_name"`
	ParentID     *uuid.UUID           `json:"parent_id"`
	TenantID     *uuid.UUID           `json:"tenant_id"`
	LeaderUserID *uuid.UUID         `json:"leader_user_id"`
	Status       string               `json:"status"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	Children     []DepartmentTreeNode `json:"children"`
}

// DepartmentService 部门业务。
type DepartmentService struct {
	dept   *repository.DepartmentRepository
	tenant *repository.TenantRepository
	user   *repository.UserRepository
}

func NewDepartmentService(
	dept *repository.DepartmentRepository,
	tenant *repository.TenantRepository,
	user *repository.UserRepository,
) *DepartmentService {
	return &DepartmentService{dept: dept, tenant: tenant, user: user}
}

// CreateDepartmentRequest 创建部门。
type CreateDepartmentRequest struct {
	DeptName     string
	ParentID     *uuid.UUID
	LeaderUserID *uuid.UUID
	TenantID     *uuid.UUID
	Status       string
}

// Create 创建部门。
func (s *DepartmentService) Create(ctx context.Context, req *CreateDepartmentRequest) (*model.Department, error) {
	if req.DeptName == "" {
		return nil, ErrDeptNameRequired
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	if req.ParentID != nil {
		p, err := s.dept.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, ErrParentNotFound
		}
	}
	d := &model.Department{
		DeptName:     req.DeptName,
		ParentID:     req.ParentID,
		LeaderUserID: req.LeaderUserID,
		TenantID:     req.TenantID,
		Status:       status,
	}
	if err := s.dept.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Get 获取详情。
func (s *DepartmentService) Get(ctx context.Context, id uuid.UUID) (*model.Department, error) {
	d, err := s.dept.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrDepartmentNotFound
	}
	return d, nil
}

// UpdateDepartmentRequest 更新部门（整资源 PUT）。parent_id 为空字符串表示无上级；非空则为上级 UUID 字符串。
type UpdateDepartmentRequest struct {
	DeptName     string     `json:"dept_name"`
	ParentID     string     `json:"parent_id"`
	LeaderUserID *uuid.UUID `json:"leader_user_id"`
	TenantID     *uuid.UUID `json:"tenant_id"`
	Status       string     `json:"status"`
}

// Update 更新。
func (s *DepartmentService) Update(ctx context.Context, id uuid.UUID, req *UpdateDepartmentRequest) (*model.Department, error) {
	d, err := s.dept.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrDepartmentNotFound
	}
	if req.DeptName == "" {
		return nil, ErrDeptNameRequired
	}
	d.DeptName = req.DeptName
	var parentPtr *uuid.UUID
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			return nil, ErrInvalidParentID
		}
		if pid == id {
			return nil, ErrParentSelf
		}
		p, err := s.dept.GetByID(ctx, pid)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, ErrParentNotFound
		}
		parentPtr = &pid
	}
	d.ParentID = parentPtr
	d.LeaderUserID = req.LeaderUserID
	d.TenantID = req.TenantID
	if req.Status != "" {
		d.Status = req.Status
	}
	if err := s.dept.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Delete 删除（无子部门、无绑定租户）。
func (s *DepartmentService) Delete(ctx context.Context, id uuid.UUID) error {
	d, err := s.dept.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrDepartmentNotFound
	}
	n, err := s.dept.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrDepartmentHasChild
	}
	tn, err := s.tenant.CountByDeptID(ctx, id)
	if err != nil {
		return err
	}
	if tn > 0 {
		return ErrDepartmentHasTenant
	}
	return s.dept.Delete(ctx, id)
}

// List 分页列表。
func (s *DepartmentService) List(ctx context.Context, page, pageSize int) ([]model.Department, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.dept.List(ctx, offset, pageSize)
}

// Tree 部门树。
func (s *DepartmentService) Tree(ctx context.Context) ([]DepartmentTreeNode, error) {
	all, err := s.dept.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	byParent := make(map[uuid.UUID][]model.Department)
	var roots []model.Department
	for _, d := range all {
		if d.ParentID == nil {
			roots = append(roots, d)
			continue
		}
		pid := *d.ParentID
		byParent[pid] = append(byParent[pid], d)
	}
	var build func(m model.Department) DepartmentTreeNode
	build = func(m model.Department) DepartmentTreeNode {
		n := toTreeNode(m)
		for _, ch := range byParent[m.ID] {
			n.Children = append(n.Children, build(ch))
		}
		return n
	}
	out := make([]DepartmentTreeNode, 0, len(roots))
	for _, r := range roots {
		out = append(out, build(r))
	}
	return out, nil
}

func toTreeNode(d model.Department) DepartmentTreeNode {
	return DepartmentTreeNode{
		ID:           d.ID,
		DeptName:     d.DeptName,
		ParentID:     d.ParentID,
		TenantID:     d.TenantID,
		LeaderUserID: d.LeaderUserID,
		Status:       d.Status,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		Children:     []DepartmentTreeNode{},
	}
}

// ListUsers 部门用户分页。
func (s *DepartmentService) ListUsers(ctx context.Context, deptID uuid.UUID, page, pageSize int) ([]model.User, int64, error) {
	d, err := s.dept.GetByID(ctx, deptID)
	if err != nil {
		return nil, 0, err
	}
	if d == nil {
		return nil, 0, ErrDepartmentNotFound
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.user.ListByDeptID(ctx, deptID, offset, pageSize)
}
