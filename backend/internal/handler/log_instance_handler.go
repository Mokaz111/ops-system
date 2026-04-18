package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogInstanceHandler 日志实例 HTTP。
type LogInstanceHandler struct {
	svc *service.LogInstanceService
}

func NewLogInstanceHandler(svc *service.LogInstanceService) *LogInstanceHandler {
	return &LogInstanceHandler{svc: svc}
}

// List GET /api/v1/log-instances
func (h *LogInstanceHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if raw := c.Query("tenant_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return
		}
		tenantID = &id
	}
	keyword := c.Query("keyword")
	list, total, err := h.svc.List(c.Request.Context(), tenantID, keyword, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/log-instances/:id
func (h *LogInstanceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Create POST /api/v1/log-instances
func (h *LogInstanceHandler) Create(c *gin.Context) {
	var body service.CreateLogInstanceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Update PUT /api/v1/log-instances/:id
func (h *LogInstanceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.UpdateLogInstanceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Delete DELETE /api/v1/log-instances/:id
func (h *LogInstanceHandler) Delete(c *gin.Context) {
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

// Query POST /api/v1/log-instances/:id/query
// M1 占位：返回空结果；M4 接入 VictoriaLogs LogsQL。
func (h *LogInstanceHandler) Query(c *gin.Context) {
	response.JSON(c, gin.H{
		"note":    "LogsQL query endpoint placeholder; to be implemented in M4",
		"results": []any{},
	})
}

func (h *LogInstanceHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLogInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrLogInstanceName),
		errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
