package handler

import (
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type userMembershipPublic struct {
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	Role          string    `json:"role"`
	WorkspaceName string    `json:"workspace_name"`
}

type userPublic struct {
	ID          uuid.UUID              `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Phone       string                 `json:"phone"`
	Role        string                 `json:"role"`
	Status      string                 `json:"status"`
	Memberships []userMembershipPublic `json:"memberships"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func toUserPublic(u *model.User) userPublic {
	if u == nil {
		return userPublic{}
	}
	return userPublic{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Phone:       u.Phone,
		Role:        u.Role,
		Status:      u.Status,
		Memberships: []userMembershipPublic{},
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func toUserPublicWithMemberships(u *model.User, memberships []service.UserWorkspaceMembership) userPublic {
	out := toUserPublic(u)
	out.Memberships = make([]userMembershipPublic, 0, len(memberships))
	for _, m := range memberships {
		out.Memberships = append(out.Memberships, userMembershipPublic{
			WorkspaceID:   m.WorkspaceID,
			Role:          m.Role,
			WorkspaceName: m.WorkspaceName,
		})
	}
	return out
}

func enrichUserPublic(c *gin.Context, memberSvc *service.WorkspaceMemberService, u *model.User) userPublic {
	if memberSvc == nil || u == nil {
		return toUserPublic(u)
	}
	memberships, err := memberSvc.ListUserWorkspaces(c.Request.Context(), u.ID)
	if err != nil {
		return toUserPublic(u)
	}
	return toUserPublicWithMemberships(u, memberships)
}
