package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DepartmentHandler 部门 HTTP。
type DepartmentHandler struct {
	svc *service.DepartmentService
}

func NewDepartmentHandler(svc *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{svc: svc}
}

type createDepartmentBody struct {
	DeptName     string     `json:"dept_name" binding:"required"`
	ParentID     *uuid.UUID `json:"parent_id"`
	LeaderUserID *uuid.UUID `json:"leader_user_id"`
	TenantID     *uuid.UUID `json:"tenant_id"`
	Status       string     `json:"status"`
}

// List GET /api/v1/departments
func (h *DepartmentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if ps < 1 {
		ps = 20
	}
	list, total, err := h.svc.List(c.Request.Context(), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{
		"items":     list,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

// Tree GET /api/v1/departments/tree
func (h *DepartmentHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, tree)
}

// Create POST /api/v1/departments
func (h *DepartmentHandler) Create(c *gin.Context) {
	var body createDepartmentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.svc.Create(c.Request.Context(), &service.CreateDepartmentRequest{
		DeptName:     body.DeptName,
		ParentID:     body.ParentID,
		LeaderUserID: body.LeaderUserID,
		TenantID:     body.TenantID,
		Status:       body.Status,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, d)
}

// Get GET /api/v1/departments/:id
func (h *DepartmentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, d)
}

// Update PUT /api/v1/departments/:id
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.svc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, d)
}

// Delete DELETE /api/v1/departments/:id
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListUsers GET /api/v1/departments/:id/users
func (h *DepartmentHandler) ListUsers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if ps < 1 {
		ps = 20
	}
	list, total, err := h.svc.ListUsers(c.Request.Context(), id, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{
		"items":     list,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

func (h *DepartmentHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrParentNotFound):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDepartmentHasChild):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrDepartmentHasTenant):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDeptNameRequired),
		errors.Is(err, service.ErrInvalidParentID),
		errors.Is(err, service.ErrParentSelf):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
	}
}
