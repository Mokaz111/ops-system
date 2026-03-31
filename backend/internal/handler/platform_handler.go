package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ops-system/backend/internal/idempotency"
	"ops-system/backend/internal/middleware"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PlatformHandler struct {
	scaleSvc *service.PlatformScaleService
	log      *zap.Logger
	idem     idempotency.Store
	audit    *repository.PlatformScaleAuditRepository
}

func NewPlatformHandler(
	scaleSvc *service.PlatformScaleService,
	log *zap.Logger,
	idemStore idempotency.Store,
	auditRepo *repository.PlatformScaleAuditRepository,
) *PlatformHandler {
	return &PlatformHandler{scaleSvc: scaleSvc, log: log, idem: idemStore, audit: auditRepo}
}

type scaleVMClusterBody struct {
	TargetID          string `json:"target_id" binding:"required"`
	DryRun            bool   `json:"dry_run"`
	VMSelectReplicas  *int32 `json:"vmselect_replicas"`
	VMInsertReplicas  *int32 `json:"vminsert_replicas"`
	VMStorageReplicas *int32 `json:"vmstorage_replicas"`
	StorageSize       string `json:"storage_size"`
}

// ListAudits GET /api/v1/platform/scaling/audits
func (h *PlatformHandler) ListAudits(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	if h.audit == nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "audit repository not configured")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	switch status {
	case "", "success", "failed", "replayed":
	default:
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid status")
		return
	}
	var startTime *time.Time
	if raw := strings.TrimSpace(c.Query("start_time")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid start_time, expect RFC3339")
			return
		}
		startTime = &t
	}
	var endTime *time.Time
	if raw := strings.TrimSpace(c.Query("end_time")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid end_time, expect RFC3339")
			return
		}
		endTime = &t
	}
	offset := (page - 1) * ps
	rows, total, err := h.audit.List(c.Request.Context(), repository.PlatformScaleAuditListFilter{
		TargetID:  strings.TrimSpace(c.Query("target_id")),
		Status:    status,
		Operator:  strings.TrimSpace(c.Query("operator")),
		StartTime: startTime,
		EndTime:   endTime,
		Offset:    offset,
		Limit:     ps,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(c, gin.H{
		"items":     rows,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

// ListVMClusterTargets GET /api/v1/platform/scaling/vmcluster/targets
func (h *PlatformHandler) ListVMClusterTargets(c *gin.Context) {
	response.JSON(c, h.scaleSvc.ListVMClusterTargets())
}

// ScaleVMCluster POST /api/v1/platform/scaling/vmcluster
func (h *PlatformHandler) ScaleVMCluster(c *gin.Context) {
	var body scaleVMClusterBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	var idemKey string
	if !body.DryRun {
		rawKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if rawKey == "" {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "Idempotency-Key is required when dry_run=false")
			return
		}
		idemKey = c.GetString(middleware.ContextUserIDKey) + ":platform_scale_vmcluster:" + rawKey
		if replayPlan, ok, err := h.getReplayPlan(c.Request.Context(), idemKey); err == nil && ok {
			h.writeAudit(c, replayPlan.TargetID, body.DryRun, "replayed", replayPlan.SpecPatch, "")
			c.Header("X-Idempotency-Replayed", "true")
			response.JSON(c, replayPlan)
			return
		} else if err != nil && h.log != nil {
			h.log.Warn("platform_scale_idempotency_get_failed", zap.Error(err), zap.String("key", idemKey))
		}
	}
	plan, err := h.scaleSvc.ScaleVMCluster(c.Request.Context(), &service.ScaleVMClusterRequest{
		TargetID:          body.TargetID,
		DryRun:            body.DryRun,
		VMSelectReplicas:  body.VMSelectReplicas,
		VMInsertReplicas:  body.VMInsertReplicas,
		VMStorageReplicas: body.VMStorageReplicas,
		StorageSize:       body.StorageSize,
	})
	if err != nil {
		if h.log != nil {
			h.log.Warn("platform_scale_vmcluster_failed",
				zap.String("user_id", c.GetString(middleware.ContextUserIDKey)),
				zap.String("username", c.GetString(middleware.ContextUsernameKey)),
				zap.String("role", c.GetString(middleware.ContextRoleKey)),
				zap.String("client_ip", c.ClientIP()),
				zap.String("target_id", body.TargetID),
				zap.Bool("dry_run", body.DryRun),
				zap.Error(err),
			)
		}
		h.writeAudit(c, body.TargetID, body.DryRun, "failed", nil, err.Error())
		h.handleErr(c, err)
		return
	}
	if !body.DryRun {
		if err := h.saveReplayPlan(c.Request.Context(), idemKey, plan); err != nil && h.log != nil {
			h.log.Warn("platform_scale_idempotency_set_failed", zap.Error(err), zap.String("key", idemKey))
		}
	}
	h.writeAudit(c, plan.TargetID, body.DryRun, "success", plan.SpecPatch, "")
	if h.log != nil {
		h.log.Info("platform_scale_vmcluster_audit",
			zap.String("user_id", c.GetString(middleware.ContextUserIDKey)),
			zap.String("username", c.GetString(middleware.ContextUsernameKey)),
			zap.String("role", c.GetString(middleware.ContextRoleKey)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("target_id", plan.TargetID),
			zap.Bool("dry_run", body.DryRun),
			zap.Any("spec_patch", plan.SpecPatch),
		)
	}
	response.JSON(c, plan)
}

func (h *PlatformHandler) getReplayPlan(ctx context.Context, idemKey string) (*service.ScaleVMClusterPlan, bool, error) {
	if h.idem == nil {
		return nil, false, nil
	}
	raw, err := h.idem.Get(ctx, idemKey)
	if errors.Is(err, idempotency.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var plan service.ScaleVMClusterPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, false, err
	}
	return &plan, true, nil
}

func (h *PlatformHandler) saveReplayPlan(ctx context.Context, idemKey string, plan *service.ScaleVMClusterPlan) error {
	if h.idem == nil || plan == nil {
		return nil
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	_, err = h.idem.SetNX(ctx, idemKey, string(raw), 10*time.Minute)
	return err
}

func (h *PlatformHandler) writeAudit(
	c *gin.Context,
	targetID string,
	dryRun bool,
	status string,
	specPatch map[string]interface{},
	errMsg string,
) {
	if h.audit == nil {
		return
	}
	rawSpec := "{}"
	if len(specPatch) > 0 {
		if b, err := json.Marshal(specPatch); err == nil {
			rawSpec = string(b)
		}
	}
	row := &model.PlatformScaleAudit{
		UserID:       c.GetString(middleware.ContextUserIDKey),
		Username:     c.GetString(middleware.ContextUsernameKey),
		Role:         c.GetString(middleware.ContextRoleKey),
		ClientIP:     c.ClientIP(),
		TargetID:     targetID,
		DryRun:       dryRun,
		Status:       status,
		SpecPatch:    rawSpec,
		ErrorMessage: errMsg,
	}
	if err := h.audit.Create(c.Request.Context(), row); err != nil && h.log != nil {
		h.log.Warn("platform_scale_audit_persist_failed", zap.Error(err))
	}
}

func (h *PlatformHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrK8sOperatorNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, service.ErrInvalidPlatformScope),
		errors.Is(err, service.ErrPlatformTargetRequired),
		errors.Is(err, service.ErrPlatformTargetNotFound),
		errors.Is(err, service.ErrPlatformScaleNoop),
		errors.Is(err, service.ErrInvalidReplicas),
		errors.Is(err, service.ErrInvalidStorageSize):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
