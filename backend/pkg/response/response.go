package response

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ── Machine-readable error codes ──────────────────────────────────────

const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeServiceUnavail = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeBadGateway     = "BAD_GATEWAY"

	ErrCodeTenantNotFound          = "TENANT_NOT_FOUND"
	ErrCodeInstanceNotFound        = "INSTANCE_NOT_FOUND"
	ErrCodeDepartmentNotFound      = "DEPARTMENT_NOT_FOUND"
	ErrCodeUserNotFound            = "USER_NOT_FOUND"
	ErrCodeClusterNotFound         = "CLUSTER_NOT_FOUND"
	ErrCodeMetricNotFound          = "METRIC_NOT_FOUND"
	ErrCodeTemplateNotFound        = "TEMPLATE_NOT_FOUND"
	ErrCodeVersionNotFound         = "VERSION_NOT_FOUND"
	ErrCodeInstallationNotFound    = "INSTALLATION_NOT_FOUND"
	ErrCodeAlertRuleNotFound       = "ALERT_RULE_NOT_FOUND"
	ErrCodeAlertEventNotFound      = "ALERT_EVENT_NOT_FOUND"
	ErrCodeChannelNotFound         = "CHANNEL_NOT_FOUND"
	ErrCodeLogInstanceNotFound     = "LOG_INSTANCE_NOT_FOUND"
	ErrCodeGrafanaInstanceNotFound = "GRAFANA_INSTANCE_NOT_FOUND"
	ErrCodePlatformTargetNotFound  = "PLATFORM_TARGET_NOT_FOUND"
	ErrCodeParentNotFound          = "PARENT_NOT_FOUND"

	ErrCodeTenantNameRequired     = "TENANT_NAME_REQUIRED"
	ErrCodeInvalidTemplateType    = "INVALID_TEMPLATE_TYPE"
	ErrCodeInvalidInstanceType    = "INVALID_INSTANCE_TYPE"
	ErrCodeInvalidInstanceStatus  = "INVALID_INSTANCE_STATUS"
	ErrCodeInvalidPagination      = "INVALID_PAGINATION"
	ErrCodeInvalidPlatformScope   = "INVALID_PLATFORM_SCOPE"
	ErrCodePlatformTargetRequired = "PLATFORM_TARGET_REQUIRED"
	ErrCodeInvalidReplicas        = "INVALID_REPLICAS"
	ErrCodeInvalidStorageSize     = "INVALID_STORAGE_SIZE"
	ErrCodeInvalidNamespace       = "INVALID_NAMESPACE"
	ErrCodeInvalidReleaseName     = "INVALID_RELEASE_NAME"
	ErrCodePlatformScaleNoop      = "PLATFORM_SCALE_NOOP"
	ErrCodeQuotaConfigNotJSON     = "QUOTA_CONFIG_NOT_JSON"
	ErrCodeDeptNameRequired       = "DEPT_NAME_REQUIRED"
	ErrCodeInvalidParentID        = "INVALID_PARENT_ID"
	ErrCodeParentSelf             = "PARENT_SELF_REFERENCE"
	ErrCodeParentCycle            = "PARENT_CYCLE_DETECTED"
	ErrCodeUsernameExists         = "USERNAME_EXISTS"
	ErrCodeUsernameRequired       = "USERNAME_REQUIRED"
	ErrCodePasswordTooShort       = "PASSWORD_TOO_SHORT"
	ErrCodeBootstrapNotAllowed    = "BOOTSTRAP_NOT_ALLOWED"
	ErrCodeInstanceNameRequired   = "INSTANCE_NAME_REQUIRED"
	ErrCodeInvalidCredentials     = "INVALID_CREDENTIALS"
	ErrCodeOrgNameRequired        = "ORG_NAME_REQUIRED"
	ErrCodeTemplateNameExists     = "TEMPLATE_NAME_EXISTS"
	ErrCodeTemplateInUse          = "TEMPLATE_IN_USE"
	ErrCodeVersionExists          = "VERSION_EXISTS"
	ErrCodeVersionInUse           = "VERSION_IN_USE"
	ErrCodeVersionLastOne         = "VERSION_LAST_ONE"
	ErrCodeTemplateNameRequired   = "TEMPLATE_NAME_REQUIRED"
	ErrCodeTenantMismatch         = "TENANT_MISMATCH"

	ErrCodeDeptHasTenant            = "DEPARTMENT_HAS_TENANT"
	ErrCodeDeptHasChild             = "DEPARTMENT_HAS_CHILD"
	ErrCodeInstanceHasInstallations = "INSTANCE_HAS_INSTALLATIONS"
	ErrCodeTenantHasInstances       = "TENANT_HAS_INSTANCES"
	ErrCodeEventAlreadyAcked        = "EVENT_ALREADY_ACKED"
	ErrCodeMetricNameExists         = "METRIC_NAME_EXISTS"
	ErrCodeTenantProvisionFailed    = "TENANT_PROVISION_FAILED"
	ErrCodeTenantDeprovisionFailed  = "TENANT_DEPROVISION_FAILED"

	ErrCodeK8sNotConfigured          = "K8S_NOT_CONFIGURED"
	ErrCodeHelmNotConfigured         = "HELM_NOT_CONFIGURED"
	ErrCodeGrafanaDisabled           = "GRAFANA_DISABLED"
	ErrCodeClusterInvalid            = "CLUSTER_INVALID"
	ErrCodeLogInstanceName           = "LOG_INSTANCE_NAME"
	ErrCodeRuleNameRequired          = "RULE_NAME_REQUIRED"
	ErrCodeInvalidRuleType           = "INVALID_RULE_TYPE"
	ErrCodeInvalidAlertLevel         = "INVALID_ALERT_LEVEL"
	ErrCodeQueryRequired             = "QUERY_REQUIRED"
	ErrCodeChannelNameRequired       = "CHANNEL_NAME_REQUIRED"
	ErrCodeInvalidChannelType        = "INVALID_CHANNEL_TYPE"
	ErrCodeTenantNotFoundForInstance = "TENANT_NOT_FOUND_FOR_INSTANCE"

		ErrCodeZoneNotFound           = "ZONE_NOT_FOUND"
		ErrCodeZoneSlugConflict       = "ZONE_SLUG_CONFLICT"
		ErrCodeZoneHasInstances       = "ZONE_HAS_ACTIVE_INSTANCES"
		ErrCodeZoneOffline            = "ZONE_OFFLINE"
		ErrCodeZoneCapacityExhausted  = "ZONE_CAPACITY_EXHAUSTED"
		ErrCodeZoneSharedNotReady     = "ZONE_SHARED_NOT_READY"
		ErrCodeZoneLogsNotReady       = "ZONE_LOGS_NOT_READY"

		ErrCodeBusinessClusterNotFound      = "BUSINESS_CLUSTER_NOT_FOUND"
		ErrCodeBusinessClusterNameConflict  = "BUSINESS_CLUSTER_NAME_CONFLICT"
		ErrCodeVMOperatorRequired           = "VM_OPERATOR_REQUIRED"
		ErrCodeInstanceHasBusinessClusters  = "INSTANCE_HAS_BUSINESS_CLUSTERS"
)

// JSON 统一成功响应。
func JSON(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// Error 业务/客户端错误（HTTP 4xx/5xx + 业务 code + 机器可读错误码）。
func Error(c *gin.Context, httpStatus int, code int, errCode string, message string) {
	c.JSON(httpStatus, gin.H{
		"code":    code,
		"error":   errCode,
		"message": message,
	})
}

// TranslateBindingError 将 Gin 的 ShouldBindJSON 错误映射为用户可读消息，
// 避免泄漏 Go 包路径和 validator 内部实现细节。
func TranslateBindingError(err error) string {
	if err == nil {
		return ""
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		var msgs []string
		for _, fe := range ve {
			field := fe.Field()
			switch fe.Tag() {
			case "required":
				msgs = append(msgs, field+" is required")
			default:
				msgs = append(msgs, field+" is invalid")
			}
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	}
	// JSON 语法错误、类型不匹配等
	return "invalid request body"
}
