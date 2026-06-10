package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ZoneHandler 可用区 HTTP handler。
type ZoneHandler struct {
	svc *service.ZoneService
}

func NewZoneHandler(svc *service.ZoneService) *ZoneHandler {
	return &ZoneHandler{svc: svc}
}

// List GET /api/v1/zones
func (h *ZoneHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	list, total, err := h.svc.List(c.Request.Context(), c.Query("status"), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/zones/:id
func (h *ZoneHandler) Get(c *gin.Context) {
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
	stats, _ := h.svc.GetStats(c.Request.Context(), id)
	response.JSON(c, gin.H{"zone": m, "stats": stats})
}

// Create POST /api/v1/zones (admin)
func (h *ZoneHandler) Create(c *gin.Context) {
	var body service.CreateZoneRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Update PUT /api/v1/zones/:id (admin)
func (h *ZoneHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body service.UpdateZoneRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// InitShared POST /api/v1/zones/:id/init-shared (admin)
// 在 Zone 的监控集群中部署共享 VMCluster。
func (h *ZoneHandler) InitShared(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body service.ZoneInitSharedRequest
	_ = c.ShouldBindJSON(&body) // values 可选，忽略 bind 错误
	plan, err := h.svc.InitShared(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, plan)
}

// InitGrafana POST /api/v1/zones/:id/init-grafana (admin)
// 在 Zone 的监控集群中部署 Zone 级 Grafana 并预配数据源。
func (h *ZoneHandler) InitGrafana(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	plan, err := h.svc.InitGrafana(c.Request.Context(), id, &service.ZoneInitSharedRequest{DryRun: false})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, plan)
}

// Delete DELETE /api/v1/zones/:id (admin)
func (h *ZoneHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *ZoneHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrZoneNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeZoneNotFound, err.Error())
	case errors.Is(err, service.ErrZoneSlugConflict):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeZoneSlugConflict, err.Error())
	case errors.Is(err, service.ErrZoneHasInstances):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeZoneHasInstances, err.Error())
	case errors.Is(err, service.ErrZoneOffline):
		response.Error(c, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, response.ErrCodeZoneOffline, err.Error())
	case errors.Is(err, service.ErrZoneCapacityExhausted):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeZoneCapacityExhausted, err.Error())
	case errors.Is(err, service.ErrZoneClusterNotReady):
		response.Error(c, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, response.ErrCodeClusterInvalid, err.Error())
	case errors.Is(err, service.ErrHelmOperatorNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, response.ErrCodeServiceUnavail, "helm operator not configured")
	case errors.Is(err, service.ErrInvalidNamespace),
		errors.Is(err, service.ErrInvalidReleaseName):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	default:
		// 未识别的错误返回 500 前记录日志，方便排查。
		zap.L().Error("zone_unhandled_error", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
