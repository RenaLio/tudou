package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateChannelRequest struct {
	Type           models.ChannelType      `json:"type" binding:"required"`
	Name           string                  `json:"name" binding:"required"`
	BaseURL        string                  `json:"baseURL" binding:"required"`
	APIKey         string                  `json:"apiKey" binding:"required"`
	Weight         *int                    `json:"weight,omitempty"`
	Remark         string                  `json:"remark"`
	Tag            string                  `json:"tag"`
	Model          string                  `json:"model"`
	CustomModel    string                  `json:"customModel"`
	Settings       *models.ChannelSettings `json:"settings,omitempty"`
	Extra          *models.ChannelExtra    `json:"extra,omitempty"`
	PriceRate      *float64                `json:"priceRate,omitempty"`
	ExpiredAt      *time.Time              `json:"expiredAt,omitempty"`
	GroupIDs       []int64                 `json:"-"`
	GroupStringIDs []string                `json:"groupIDs,omitempty"`
}

type UpdateChannelRequest struct {
	Type           *models.ChannelType     `json:"type,omitempty"`
	Name           *string                 `json:"name,omitempty"`
	BaseURL        *string                 `json:"baseURL,omitempty"`
	APIKey         *string                 `json:"apiKey,omitempty"`
	Weight         *int                    `json:"weight,omitempty"`
	Status         *models.ChannelStatus   `json:"status,omitempty"`
	Remark         *string                 `json:"remark,omitempty"`
	Tag            *string                 `json:"tag,omitempty"`
	Model          *string                 `json:"model,omitempty"`
	CustomModel    *string                 `json:"customModel,omitempty"`
	Settings       *models.ChannelSettings `json:"settings,omitempty"`
	Extra          *models.ChannelExtra    `json:"extra,omitempty"`
	PriceRate      *float64                `json:"priceRate,omitempty"`
	ExpiredAt      *time.Time              `json:"expiredAt,omitempty"`
	GroupIDs       []int64                 `json:"-"`
	GroupStringIDs []string                `json:"groupIDs,omitempty"`
}

type SetChannelStatusRequest struct {
	Status *models.ChannelStatus `json:"status" binding:"required"`
}

type ReplaceChannelGroupsRequest struct {
	GroupIDs []int64 `json:"groupIDs"`
}

type ListChannelsRequest struct {
	Page          int                   `form:"page"`
	PageSize      int                   `form:"pageSize"`
	OrderBy       string                `form:"orderBy"`
	Keyword       string                `form:"keyword"`
	GroupID       int64                 `form:"-"`
	GroupStringID string                `form:"groupID"`
	Type          string                `form:"type"`
	Status        *models.ChannelStatus `form:"status"`
	OnlyAvailable bool                  `form:"onlyAvailable"`
	PreloadGroups bool                  `form:"preloadGroups"`
	PreloadStats  bool                  `form:"preloadStats"`
}

type ChannelResponse struct {
	ID          int64                  `json:"id,string"`
	Type        models.ChannelType     `json:"type"`
	Name        string                 `json:"name"`
	BaseURL     string                 `json:"baseURL"`
	APIKey      string                 `json:"apiKey"`
	Weight      int                    `json:"weight"`
	Status      models.ChannelStatus   `json:"status"`
	Remark      string                 `json:"remark"`
	Tag         string                 `json:"tag"`
	Model       string                 `json:"model"`
	CustomModel string                 `json:"customModel"`
	Settings    models.ChannelSettings `json:"settings"`
	Extra       models.ChannelExtra    `json:"extra"`
	PriceRate   float64                `json:"priceRate"`
	ExpiredAt   *time.Time             `json:"expiredAt,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	GroupIDs    []string               `json:"groupIDs,omitempty"`
	Stats       *models.ChannelStats   `json:"stats,omitempty"`
}
