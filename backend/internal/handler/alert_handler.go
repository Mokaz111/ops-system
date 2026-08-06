package handler

import (
	"errors"
	"net/http"
	"time"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AlertHandler 告警相关 HTTP。
type AlertHandler struct {
	alertSvc   *service.AlertService
	eventSvc   *service.AlertEventService
	channelSvc *service.NotificationChannelService
	userSvc    *service.UserService
}

func NewAlertHandler(
	alertSvc *service.AlertService,
	eventSvc *service.AlertEventService,
	channelSvc *service.NotificationChannelService,
	userSvc *service.UserService,
) *AlertHandler {
	return &AlertHandler{alertSvc: alertSvc, eventSvc: eventSvc, channelSvc: channelSvc, userSvc: userSvc}
}

// ── Alert Rules ──

type createAlertRuleBody struct {
	TenantID    uuid.UUID `json:"tenant_id" binding:"required"`
	RuleName    string    `json:"rule_name" binding:"required"`
	RuleType    string    `json:"rule_type" binding:"required"`
	Query       string    `json:"query" binding:"required"`
	Condition   string    `json:"condition"`
	Level       string    `json:"level" binding:"required"`
	Channels    string    `json:"channels"`
	Annotations string    `json:"annotations"`
	Enabled     bool      `json:"enabled"`
}

type updateAlertRuleBody struct {
	RuleName    string `json:"rule_name"`
	RuleType    string `json:"rule_type"`
	Query       string `json:"query"`
	Condition   string `json:"condition"`
	Level       string `json:"level"`
	Channels    string `json:"channels"`
	Annotations string `json:"annotations"`
	Enabled     *bool  `json:"enabled"`
}

// ListRules GET /api/v1/alerts/rules
func (h *AlertHandler) ListRules(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if !isAdmin(c) {
		scope, ok := resolveWorkspaceScope(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = scope
	} else if s := c.Query("workspace_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = &id
	}
	ruleType := c.Query("rule_type")
	level := c.Query("level")
	keyword := c.Query("keyword")

	list, total, err := h.alertSvc.ListRules(c.Request.Context(), page, ps, tenantID, ruleType, level, keyword)
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

// CreateRule POST /api/v1/alerts/rules
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var body createAlertRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	rule, err := h.alertSvc.CreateRule(c.Request.Context(), &service.CreateAlertRuleRequest{
		TenantID:    body.TenantID,
		RuleName:    body.RuleName,
		RuleType:    body.RuleType,
		Query:       body.Query,
		Condition:   body.Condition,
		Level:       body.Level,
		Channels:    body.Channels,
		Annotations: body.Annotations,
		Enabled:     body.Enabled,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, rule)
}

// UpdateRule PUT /api/v1/alerts/rules/:id
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body updateAlertRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	rule, err := h.alertSvc.UpdateRule(c.Request.Context(), id, &service.UpdateAlertRuleRequest{
		RuleName:    body.RuleName,
		RuleType:    body.RuleType,
		Query:       body.Query,
		Condition:   body.Condition,
		Level:       body.Level,
		Channels:    body.Channels,
		Annotations: body.Annotations,
		Enabled:     body.Enabled,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, rule)
}

// DeleteRule DELETE /api/v1/alerts/rules/:id
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.alertSvc.DeleteRule(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Alert Events ──

// ListEvents GET /api/v1/alerts/events
func (h *AlertHandler) ListEvents(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if !isAdmin(c) {
		scope, ok := resolveWorkspaceScope(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = scope
	} else if s := c.Query("workspace_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = &id
	}
	var ruleID *uuid.UUID
	if s := c.Query("rule_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid rule_id")
			return
		}
		ruleID = &id
	}
	level := c.Query("level")
	status := c.Query("status")
	var startTime, endTime *time.Time
	if s := c.Query("start_time"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid start_time")
			return
		}
		startTime = &t
	}
	if s := c.Query("end_time"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid end_time")
			return
		}
		endTime = &t
	}

	list, total, err := h.eventSvc.ListEvents(c.Request.Context(), page, ps, tenantID, ruleID, level, status, startTime, endTime)
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

// GetEvent GET /api/v1/alerts/events/:id
func (h *AlertHandler) GetEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	event, err := h.eventSvc.GetEvent(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !isAdmin(c) {
		if !assertWorkspaceAccess(c, h.userSvc, event.TenantID) {
			return
		}
	}
	response.JSON(c, event)
}

// AckEvent PUT /api/v1/alerts/events/:id/ack
func (h *AlertHandler) AckEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return
	}
	if !isAdmin(c) {
		event, err := h.eventSvc.GetEvent(c.Request.Context(), eventID)
		if err != nil {
			h.handleErr(c, err)
			return
		}
		if !assertWorkspaceAccess(c, h.userSvc, event.TenantID) {
			return
		}
	}

	event, err := h.eventSvc.AckEvent(c.Request.Context(), eventID, userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, event)
}

// ── Notification Channels ──

type createChannelBody struct {
	TenantID    uuid.UUID `json:"tenant_id" binding:"required"`
	ChannelName string    `json:"channel_name" binding:"required"`
	ChannelType string    `json:"channel_type" binding:"required"`
	Config      string    `json:"config"`
	Enabled     bool      `json:"enabled"`
}

type updateChannelBody struct {
	ChannelName string `json:"channel_name"`
	ChannelType string `json:"channel_type"`
	Config      string `json:"config"`
	Enabled     *bool  `json:"enabled"`
}

// ListChannels GET /api/v1/alerts/channels
func (h *AlertHandler) ListChannels(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if !isAdmin(c) {
		scope, ok := resolveWorkspaceScope(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = scope
	} else if s := c.Query("workspace_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = &id
	}
	channelType := c.Query("channel_type")
	keyword := c.Query("keyword")

	list, total, err := h.channelSvc.List(c.Request.Context(), page, ps, tenantID, channelType, keyword)
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

// CreateChannel POST /api/v1/alerts/channels
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	var body createChannelBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	ch, err := h.channelSvc.Create(c.Request.Context(), &service.CreateChannelRequest{
		TenantID:    body.TenantID,
		ChannelName: body.ChannelName,
		ChannelType: body.ChannelType,
		Config:      body.Config,
		Enabled:     body.Enabled,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, ch)
}

// UpdateChannel PUT /api/v1/alerts/channels/:id
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body updateChannelBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	ch, err := h.channelSvc.Update(c.Request.Context(), id, &service.UpdateChannelRequest{
		ChannelName: body.ChannelName,
		ChannelType: body.ChannelType,
		Config:      body.Config,
		Enabled:     body.Enabled,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, ch)
}

// DeleteChannel DELETE /api/v1/alerts/channels/:id
func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.channelSvc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Stats ──

// Summary GET /api/v1/alerts/stats/summary
func (h *AlertHandler) Summary(c *gin.Context) {
	var tenantID uuid.UUID
	if !isAdmin(c) {
		id, ok := resolveWorkspaceID(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = id
	} else {
		id, err := uuid.Parse(c.Query("workspace_id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = id
	}
	summary, err := h.eventSvc.Summary(c.Request.Context(), tenantID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, summary)
}

// Trend GET /api/v1/alerts/stats/trend
func (h *AlertHandler) Trend(c *gin.Context) {
	var tenantID uuid.UUID
	if !isAdmin(c) {
		id, ok := resolveWorkspaceID(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = id
	} else {
		id, err := uuid.Parse(c.Query("workspace_id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = id
	}
	start, end, ok := h.parseTimeRange(c)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "hour")

	data, err := h.eventSvc.Trend(c.Request.Context(), tenantID, start, end, interval)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, data)
}

// StatsByLevel GET /api/v1/alerts/stats/by-level
func (h *AlertHandler) StatsByLevel(c *gin.Context) {
	var tenantID uuid.UUID
	if !isAdmin(c) {
		id, ok := resolveWorkspaceID(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = id
	} else {
		id, err := uuid.Parse(c.Query("workspace_id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = id
	}
	start, end, ok := h.parseTimeRange(c)
	if !ok {
		return
	}
	data, err := h.eventSvc.StatsByLevel(c.Request.Context(), tenantID, start, end)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, data)
}

// StatsByRule GET /api/v1/alerts/stats/by-rule
func (h *AlertHandler) StatsByRule(c *gin.Context) {
	var tenantID uuid.UUID
	if !isAdmin(c) {
		id, ok := resolveWorkspaceID(c, h.userSvc)
		if !ok {
			return
		}
		tenantID = id
	} else {
		id, err := uuid.Parse(c.Query("workspace_id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return
		}
		tenantID = id
	}
	start, end, ok := h.parseTimeRange(c)
	if !ok {
		return
	}
	limit, ok := parsePositiveIntQuery(c, "limit", "10")
	if !ok {
		return
	}
	data, err := h.eventSvc.StatsByRule(c.Request.Context(), tenantID, start, end, limit)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, data)
}

func (h *AlertHandler) parseTimeRange(c *gin.Context) (time.Time, time.Time, bool) {
	start, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid start")
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid end")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func (h *AlertHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAlertRuleNotFound),
		errors.Is(err, service.ErrAlertEventNotFound),
		errors.Is(err, service.ErrChannelNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeNotFound, err.Error())
	case errors.Is(err, service.ErrWorkspaceNotFound):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeTenantNotFound, err.Error())
	case errors.Is(err, service.ErrRuleNameRequired),
		errors.Is(err, service.ErrInvalidRuleType),
		errors.Is(err, service.ErrInvalidAlertLevel),
		errors.Is(err, service.ErrQueryRequired),
		errors.Is(err, service.ErrChannelNameRequired),
		errors.Is(err, service.ErrInvalidChannelType),
		errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	case errors.Is(err, service.ErrEventAlreadyAcked):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeEventAlreadyAcked, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
