package service

import (
	"context"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrAlertRuleNotFound = errors.New("alert rule not found")
	ErrRuleNameRequired  = errors.New("rule_name required")
	ErrInvalidRuleType   = errors.New("invalid rule_type")
	ErrInvalidAlertLevel = errors.New("invalid alert level")
	ErrQueryRequired     = errors.New("query required")
)

var allowedRuleTypes = map[string]struct{}{
	"metrics": {},
	"logs":    {},
}

var allowedAlertLevels = map[string]struct{}{
	"critical": {},
	"warning":  {},
	"info":     {},
}

// CreateAlertRuleRequest 创建告警规则请求。
type CreateAlertRuleRequest struct {
	TenantID    uuid.UUID
	RuleName    string
	RuleType    string
	Query       string
	Condition   string
	Level       string
	Channels    string
	Annotations string
	Enabled     bool
}

// UpdateAlertRuleRequest 更新告警规则请求。
type UpdateAlertRuleRequest struct {
	RuleName    string
	RuleType    string
	Query       string
	Condition   string
	Level       string
	Channels    string
	Annotations string
	Enabled     *bool
}

// AlertService 告警规则业务。
type AlertService struct {
	ruleRepo      *repository.AlertRuleRepository
	workspaceRepo *repository.WorkspaceRepository
	vmRules       *vm.VMOperatorClient
	log           *zap.Logger
}

func NewAlertService(
	ruleRepo *repository.AlertRuleRepository,
	workspaceRepo *repository.WorkspaceRepository,
	vmRules *vm.VMOperatorClient,
	log *zap.Logger,
) *AlertService {
	return &AlertService{ruleRepo: ruleRepo, workspaceRepo: workspaceRepo, vmRules: vmRules, log: log}
}

// CreateRule 创建告警规则。
func (s *AlertService) CreateRule(ctx context.Context, req *CreateAlertRuleRequest) (*model.AlertRule, error) {
	if req.RuleName == "" {
		return nil, ErrRuleNameRequired
	}
	if !validRuleType(req.RuleType) {
		return nil, ErrInvalidRuleType
	}
	if !validAlertLevel(req.Level) {
		return nil, ErrInvalidAlertLevel
	}
	if req.Query == "" {
		return nil, ErrQueryRequired
	}

	t, err := s.workspaceRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrWorkspaceNotFound
	}

	rule := &model.AlertRule{
		TenantID:    req.TenantID,
		RuleName:    strings.TrimSpace(req.RuleName),
		RuleType:    req.RuleType,
		Query:       req.Query,
		Condition:   req.Condition,
		Level:       req.Level,
		Channels:    req.Channels,
		Annotations: req.Annotations,
		Enabled:     req.Enabled,
	}
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}

	if err := s.applyVMRule(ctx, t, rule); err != nil {
		s.log.Warn("vmrule_apply_failed", zap.Error(err), zap.String("rule_id", rule.ID.String()))
	}

	return rule, nil
}

// GetRule 获取告警规则。
func (s *AlertService) GetRule(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrAlertRuleNotFound
	}
	return rule, nil
}

// ListRules 分页列表。
func (s *AlertService) ListRules(ctx context.Context, page, pageSize int, tenantID *uuid.UUID, ruleType, level, keyword string) ([]model.AlertRule, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.ruleRepo.List(ctx, repository.AlertRuleListFilter{
		TenantID: tenantID,
		RuleType: ruleType,
		Level:    level,
		Keyword:  keyword,
		Offset:   offset,
		Limit:    pageSize,
	})
}

// UpdateRule 更新告警规则。
func (s *AlertService) UpdateRule(ctx context.Context, id uuid.UUID, req *UpdateAlertRuleRequest) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrAlertRuleNotFound
	}

	if req.RuleName != "" {
		rule.RuleName = strings.TrimSpace(req.RuleName)
	}
	if req.RuleType != "" {
		if !validRuleType(req.RuleType) {
			return nil, ErrInvalidRuleType
		}
		rule.RuleType = req.RuleType
	}
	if req.Level != "" {
		if !validAlertLevel(req.Level) {
			return nil, ErrInvalidAlertLevel
		}
		rule.Level = req.Level
	}
	if req.Query != "" {
		rule.Query = req.Query
	}
	if req.Condition != "" {
		rule.Condition = req.Condition
	}
	if req.Channels != "" {
		rule.Channels = req.Channels
	}
	if req.Annotations != "" {
		rule.Annotations = req.Annotations
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	if t, err := s.workspaceRepo.GetByID(ctx, rule.TenantID); err == nil && t != nil {
		if err := s.applyVMRule(ctx, t, rule); err != nil {
			s.log.Warn("vmrule_update_failed", zap.Error(err), zap.String("rule_id", rule.ID.String()))
		}
	}

	return rule, nil
}

func (s *AlertService) applyVMRule(ctx context.Context, t *model.Workspace, rule *model.AlertRule) error {
	if s == nil || s.vmRules == nil || t == nil || rule == nil || rule.RuleType != "metrics" {
		return nil
	}
	ns := strings.TrimSpace(t.VMNamespace)
	if ns == "" {
		ns = "ops-tenant-" + strings.ToLower(strings.ReplaceAll(t.VMUserID, "_", "-"))
	}
	name := strings.ToLower(strings.ReplaceAll(rule.RuleName, " ", "-"))
	if name == "" {
		name = "rule-" + rule.ID.String()
	}
	rule.VMNamespace = ns
	rule.VMRuleName = name
	err := s.vmRules.ApplyAlertRule(ctx, vm.AlertRuleSpec{
		Name:        name,
		Namespace:   ns,
		TenantID:    t.ID.String(),
		Expr:        rule.Query,
		For:         "1m",
		Severity:    rule.Level,
		Annotations: rule.Annotations,
		Enabled:     rule.Enabled,
	})
	if err != nil {
		return err
	}
	return s.ruleRepo.Update(ctx, rule)
}

// DeleteRule 删除告警规则。
func (s *AlertService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if rule == nil {
		return ErrAlertRuleNotFound
	}
	return s.ruleRepo.Delete(ctx, id)
}

// ToggleRule 启用/禁用告警规则。
func (s *AlertService) ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrAlertRuleNotFound
	}

	rule.Enabled = enabled
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	if t, err := s.workspaceRepo.GetByID(ctx, rule.TenantID); err == nil && t != nil {
		if err := s.applyVMRule(ctx, t, rule); err != nil {
			s.log.Warn("vmrule_toggle_failed", zap.Error(err), zap.String("rule_id", rule.ID.String()))
		}
	}

	return rule, nil
}

func validRuleType(s string) bool {
	_, ok := allowedRuleTypes[s]
	return ok
}

func validAlertLevel(s string) bool {
	_, ok := allowedAlertLevels[s]
	return ok
}
