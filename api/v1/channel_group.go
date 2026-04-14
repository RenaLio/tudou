package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateChannelGroupRequest struct {
	Name                string                      `json:"name" binding:"required"`
	NameRemark          string                      `json:"nameRemark"`
	Description         string                      `json:"description"`
	PermissionNum       *int32                      `json:"permissionNum,omitempty"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
	ChannelIDs          []int64                     `json:"channelIDs,omitempty"`
}

type UpdateChannelGroupRequest struct {
	Name                *string                     `json:"name,omitempty"`
	NameRemark          *string                     `json:"nameRemark,omitempty"`
	Description         *string                     `json:"description,omitempty"`
	PermissionNum       *int32                      `json:"permissionNum,omitempty"`
	LoadBalanceStrategy *models.LoadBalanceStrategy `json:"loadBalanceStrategy,omitempty"`
	ChannelIDs          []int64                     `json:"channelIDs,omitempty"`
	ReplaceChannels     bool                        `json:"replaceChannels"`
}

type ReplaceGroupChannelsRequest struct {
	ChannelIDs []int64 `json:"channelIDs"`
}

type ListChannelGroupsRequest struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
	OrderBy         string `form:"orderBy"`
	Keyword         string `form:"keyword"`
	ChannelID       int64  `form:"channelID"`
	PermissionNumGE *int32 `form:"permissionNumGE"`
	PreloadChannels bool   `form:"preloadChannels"`
}

type ChannelGroupResponse struct {
	ID                  int64                      `json:"id,string"`
	Name                string                     `json:"name"`
	NameRemark          string                     `json:"nameRemark"`
	Description         string                     `json:"description"`
	PermissionNum       int32                      `json:"permissionNum"`
	LoadBalanceStrategy models.LoadBalanceStrategy `json:"loadBalanceStrategy"`
	CreatedAt           time.Time                  `json:"createdAt"`
	UpdatedAt           time.Time                  `json:"updatedAt"`
	ChannelIDs          []int64                    `json:"channelIDs,omitempty"`
}
