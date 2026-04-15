package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateTokenRequest struct {
	UserID              int64                       `json:"-"`
	GroupID             int64                       `json:"groupID,string" binding:"required"`
	Name                string                      `json:"name"`
	Status              *models.TokenStatus         `json:"status,omitempty"`
	Limit               *float64                    `json:"limit,omitempty"`
	ExpiresAt           *time.Time                  `json:"expiresAt,omitempty"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
	Settings            models.TokenSettings        `json:"settings"`
}

type UpdateTokenRequest struct {
	Name                *string                     `json:"name,omitempty"`
	Status              *models.TokenStatus         `json:"status,omitempty"`
	LimitMicros         *int64                      `json:"limitMicros,omitempty"`
	ExpiresAt           **time.Time                 `json:"expiresAt,omitempty"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
	Settings            *models.TokenSettings       `json:"settings,omitempty"`
}

type SetTokenStatusRequest struct {
	Status models.TokenStatus `json:"status" binding:"required"`
}

type ListTokensRequest struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"pageSize"`
	OrderBy       string `form:"orderBy"`
	Keyword       string `form:"keyword"`
	UserID        int64  `form:"userID"`
	GroupID       int64  `form:"groupID"`
	Status        string `form:"status"`
	OnlyAvailable bool   `form:"onlyAvailable"`
	PreloadUser   bool   `form:"preloadUser"`
	PreloadGroup  bool   `form:"preloadGroup"`
	PreloadStats  bool   `form:"preloadStats"`
}

type TokenResponse struct {
	ID                  int64                      `json:"id,string"`
	UserID              int64                      `json:"userID,string"`
	GroupID             int64                      `json:"groupID,string"`
	Token               string                     `json:"token"`
	Name                string                     `json:"name"`
	Status              models.TokenStatus         `json:"status"`
	LimitMicros         int64                      `json:"limitMicros"`
	ExpiresAt           *time.Time                 `json:"expiresAt,omitempty"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"loadBalanceStrategy"`
	Settings            models.TokenSettings       `json:"settings"`
	CreatedAt           time.Time                  `json:"createdAt"`
	UpdatedAt           time.Time                  `json:"updatedAt"`
}

type TokenWithRelationsResponse struct {
	TokenResponse
	User  *UserResponse         `json:"user,omitempty"`
	Group *ChannelGroupResponse `json:"group,omitempty"`
	Stats *models.TokenStats    `json:"stats,omitempty"`
}
