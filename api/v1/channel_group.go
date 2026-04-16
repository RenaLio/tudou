package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateChannelGroupRequest struct {
	Name                string                      `json:"name" binding:"required"`
	NameRemark          string                      `json:"nameRemark"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
}

type UpdateChannelGroupRequest struct {
	Name                *string                     `json:"name,omitempty"`
	NameRemark          *string                     `json:"nameRemark,omitempty"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
}

type ListChannelGroupsRequest struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
	OrderBy         string `form:"orderBy"`
	Keyword         string `form:"keyword"`
	PreloadChannels bool   `form:"preloadChannels"`
}

type ChannelGroupResponse struct {
	ID                  int64                      `json:"id,string"`
	Name                string                     `json:"name"`
	NameRemark          string                     `json:"nameRemark"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"loadBalanceStrategy"`
	CreatedAt           time.Time                  `json:"createdAt"`
	UpdatedAt           time.Time                  `json:"updatedAt"`
}
