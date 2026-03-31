package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler 用户 HTTP。
type UserHandler struct {
	userSvc   *service.UserService
	jwtSecret string
}

func NewUserHandler(userSvc *service.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{userSvc: userSvc, jwtSecret: jwtSecret}
}

type bootstrapBody struct {
	Username string     `json:"username" binding:"required"`
	Password string     `json:"password" binding:"required"`
	Email    string     `json:"email"`
	Phone    string     `json:"phone"`
	DeptID   *uuid.UUID `json:"dept_id"`
	TenantID *uuid.UUID `json:"tenant_id"`
}

type createUserBody struct {
	Username string     `json:"username" binding:"required"`
	Password string     `json:"password" binding:"required"`
	Email    string     `json:"email"`
	Phone    string     `json:"phone"`
	DeptID   *uuid.UUID `json:"dept_id"`
	TenantID *uuid.UUID `json:"tenant_id"`
	Role     string     `json:"role"`
	Status   string     `json:"status"`
}

type updateUserBody struct {
	Email    *string    `json:"email"`
	Phone    *string    `json:"phone"`
	DeptID   *uuid.UUID `json:"dept_id"`
	TenantID *uuid.UUID `json:"tenant_id"`
	Role     *string    `json:"role"`
	Status   *string    `json:"status"`
	Password *string    `json:"password"`
}

// Bootstrap POST /api/v1/users/bootstrap
func (h *UserHandler) Bootstrap(c *gin.Context) {
	var body bootstrapBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.userSvc.Bootstrap(c.Request.Context(), &service.CreateUserRequest{
		Username: body.Username,
		Password: body.Password,
		Email:    body.Email,
		Phone:    body.Phone,
		DeptID:   body.DeptID,
		TenantID: body.TenantID,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toUserPublic(u))
}

// List GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	if !isAdmin(c) {
		callerID, ok := userIDFromContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
			return
		}
		u, err := h.userSvc.Get(c.Request.Context(), callerID)
		if err != nil {
			h.handleErr(c, err)
			return
		}
		response.JSON(c, gin.H{
			"items":     []userPublic{toUserPublic(u)},
			"total":     1,
			"page":      page,
			"page_size": ps,
		})
		return
	}
	var deptID, tenantID *uuid.UUID
	if s := c.Query("dept_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid dept_id")
			return
		}
		deptID = &id
	}
	if s := c.Query("tenant_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return
		}
		tenantID = &id
	}
	role := c.Query("role")
	status := c.Query("status")
	keyword := c.Query("keyword")

	list, total, err := h.userSvc.List(c.Request.Context(), page, ps, deptID, tenantID, role, status, keyword)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	items := make([]userPublic, 0, len(list))
	for i := range list {
		items = append(items, toUserPublic(&list[i]))
	}
	response.JSON(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

// Create POST /api/v1/users（需管理员；未配置 jwt.secret 时不校验角色）
func (h *UserHandler) Create(c *gin.Context) {
	if h.jwtSecret != "" && !isAdmin(c) {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, "admin only")
		return
	}
	var body createUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.userSvc.Create(c.Request.Context(), &service.CreateUserRequest{
		Username: body.Username,
		Password: body.Password,
		Email:    body.Email,
		Phone:    body.Phone,
		DeptID:   body.DeptID,
		TenantID: body.TenantID,
		Role:     body.Role,
		Status:   body.Status,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toUserPublic(u))
}

// Get GET /api/v1/users/:id
func (h *UserHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if h.jwtSecret != "" && !isAdmin(c) {
		caller, ok := userIDFromContext(c)
		if !ok || caller != id {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return
		}
	}
	u, err := h.userSvc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toUserPublic(u))
}

// Update PUT /api/v1/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if h.jwtSecret != "" {
		caller, ok := userIDFromContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(c) && caller != id {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return
		}
	}
	var body updateUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.userSvc.Update(c.Request.Context(), id, &service.UpdateUserRequest{
		Email:    body.Email,
		Phone:    body.Phone,
		DeptID:   body.DeptID,
		TenantID: body.TenantID,
		Role:     body.Role,
		Status:   body.Status,
		Password: body.Password,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toUserPublic(u))
}

// Delete DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if h.jwtSecret != "" && !isAdmin(c) {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, "admin only")
		return
	}
	if err := h.userSvc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *UserHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrUsernameExists),
		errors.Is(err, service.ErrUsernameRequired),
		errors.Is(err, service.ErrPasswordTooShort):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrBootstrapNotAllowed):
		response.Error(c, http.StatusForbidden, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
