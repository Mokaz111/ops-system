package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkspaceMemberHandler 工作空间成员 HTTP。
type WorkspaceMemberHandler struct {
	svc     *service.WorkspaceMemberService
	userSvc *service.UserService
}

func NewWorkspaceMemberHandler(svc *service.WorkspaceMemberService, userSvc *service.UserService) *WorkspaceMemberHandler {
	return &WorkspaceMemberHandler{svc: svc, userSvc: userSvc}
}

type addMemberBody struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role"`
}

type updateMemberRoleBody struct {
	Role string `json:"role" binding:"required"`
}

// List GET /api/v1/workspaces/:id/members
func (h *WorkspaceMemberHandler) List(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace id")
		return
	}
	if !assertWorkspaceAccess(c, h.userSvc, workspaceID) {
		return
	}
	list, err := h.svc.ListMembers(c.Request.Context(), workspaceID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list})
}

// Add POST /api/v1/workspaces/:id/members
func (h *WorkspaceMemberHandler) Add(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace id")
		return
	}
	if !isAdmin(c) {
		allowed, aErr := h.userSvc.CanAccessWorkspace(c.Request.Context(), mustUserID(c), workspaceID, "write")
		if aErr != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return
		}
	}
	var body addMemberBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.AddMember(c.Request.Context(), workspaceID, body.UserID, body.Role)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Update PUT /api/v1/workspaces/:id/members/:userId
func (h *WorkspaceMemberHandler) Update(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace id")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid user id")
		return
	}
	if !isAdmin(c) {
		allowed, aErr := h.userSvc.CanAccessWorkspace(c.Request.Context(), mustUserID(c), workspaceID, "write")
		if aErr != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return
		}
	}
	var body updateMemberRoleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.UpdateRole(c.Request.Context(), workspaceID, userID, body.Role)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Remove DELETE /api/v1/workspaces/:id/members/:userId
func (h *WorkspaceMemberHandler) Remove(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace id")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid user id")
		return
	}
	if !isAdmin(c) {
		allowed, aErr := h.userSvc.CanAccessWorkspace(c.Request.Context(), mustUserID(c), workspaceID, "write")
		if aErr != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return
		}
	}
	if err := h.svc.RemoveMember(c.Request.Context(), workspaceID, userID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func mustUserID(c *gin.Context) uuid.UUID {
	id, _ := userIDFromContext(c)
	return id
}

func (h *WorkspaceMemberHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrWorkspaceNotFound),
		errors.Is(err, service.ErrWorkspaceMemberNotFound),
		errors.Is(err, service.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeNotFound, err.Error())
	case errors.Is(err, service.ErrWorkspaceMemberExists):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeConflict, err.Error())
	case errors.Is(err, service.ErrInvalidWorkspaceRole):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
