package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateSystemConfigRequest struct {
	Key         string             `json:"key" binding:"required"`
	Value       any                `json:"value"`
	Type        models.ConfigType  `json:"type"`
	Scope       models.ConfigScope `json:"scope"`
	Description string             `json:"description"`
	IsEditable  *bool              `json:"isEditable,omitempty"`
	IsVisible   *bool              `json:"isVisible,omitempty"`
	Sort        *int               `json:"sort,omitempty"`
}

type UpsertSystemConfigRequest struct {
	Key         string             `json:"key" binding:"required"`
	Value       any                `json:"value"`
	Type        models.ConfigType  `json:"type"`
	Scope       models.ConfigScope `json:"scope"`
	Description string             `json:"description"`
	IsEditable  *bool              `json:"isEditable,omitempty"`
	IsVisible   *bool              `json:"isVisible,omitempty"`
	Sort        *int               `json:"sort,omitempty"`
}

type SetSystemConfigValueRequest struct {
	Value any `json:"value"`
}

type ListSystemConfigsRequest struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	OrderBy     string `form:"orderBy"`
	Keyword     string `form:"keyword"`
	Scope       string `form:"scope"`
	OnlyVisible bool   `form:"onlyVisible"`
}

type SystemConfigResponse struct {
	ID          int64              `json:"id,string"`
	Key         string             `json:"key"`
	Value       any                `json:"value"`
	Type        models.ConfigType  `json:"type"`
	Scope       models.ConfigScope `json:"scope"`
	Description string             `json:"description"`
	IsEditable  bool               `json:"isEditable"`
	IsVisible   bool               `json:"isVisible"`
	Sort        int                `json:"sort"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}
