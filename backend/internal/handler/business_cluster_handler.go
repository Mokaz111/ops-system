package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BusinessClusterHandler 业务集群 HTTP handler。
type BusinessClusterHandler struct {
	svc     *service.BusinessClusterService
	userSvc *service.UserService
}

func NewBusinessClusterHandler(svc *service.BusinessClusterService, userSvc *service.UserService) *BusinessClusterHandler {
	return &BusinessClusterHandler{svc: svc, userSvc: userSvc}
}

// List GET /api/v1/business-clusters
func (h *BusinessClusterHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	tenantID := c.Query("workspace_id")
	instanceID := c.Query("instance_id")
	list, total, err := h.svc.List(c.Request.Context(), tenantID, instanceID, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/business-clusters/:id
func (h *BusinessClusterHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Create POST /api/v1/business-clusters (tenant_admin)
func (h *BusinessClusterHandler) Create(c *gin.Context) {
	var body service.CreateBusinessClusterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	tenantID, ok := resolveWorkspaceID(c, h.userSvc)
	if !ok {
		return
	}
	m, err := h.svc.Create(c.Request.Context(), tenantID, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Delete DELETE /api/v1/business-clusters/:id (tenant_admin)
func (h *BusinessClusterHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	force := c.Query("force") == "true"
	if err := h.svc.Delete(c.Request.Context(), id, force); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *BusinessClusterHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBusinessClusterNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeBusinessClusterNotFound, err.Error())
	case errors.Is(err, service.ErrBusinessClusterNameConflict):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeBusinessClusterNameConflict, err.Error())
	case errors.Is(err, service.ErrInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeInstanceNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
