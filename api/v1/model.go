package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type CreateAIModelRequest struct {
	Name         string                   `json:"name" binding:"required"`
	Type         models.ModelType         `json:"type"`
	Description  string                   `json:"description"`
	Pricing      models.ModelPricing      `json:"pricing"`
	Capabilities models.ModelCapabilities `json:"capabilities"`
	PricingType  models.ModelPricingType  `json:"pricingType"`
	IsEnabled    *bool                    `json:"isEnabled,omitempty"`
	Extra        models.AIModelExtra      `json:"extra"`
}

type UpdateAIModelRequest struct {
	Name         *string                   `json:"name,omitempty"`
	Type         *models.ModelType         `json:"type,omitempty"`
	Description  *string                   `json:"description,omitempty"`
	Pricing      *models.ModelPricing      `json:"pricing,omitempty"`
	Capabilities *models.ModelCapabilities `json:"capabilities,omitempty"`
	PricingType  *models.ModelPricingType  `json:"pricingType,omitempty"`
	IsEnabled    *bool                     `json:"isEnabled,omitempty"`
	Extra        *models.AIModelExtra      `json:"extra,omitempty"`
}

type SetAIModelEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type ListAIModelsRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	OrderBy   string `form:"orderBy"`
	Keyword   string `form:"keyword"`
	Type      string `form:"type"`
	IsEnabled *bool  `form:"isEnabled"`
}

type AIModelResponse struct {
	ID           int64                    `json:"id,string"`
	Name         string                   `json:"name"`
	Type         models.ModelType         `json:"type"`
	Description  string                   `json:"description"`
	Pricing      models.ModelPricing      `json:"pricing"`
	Capabilities models.ModelCapabilities `json:"capabilities"`
	PricingType  models.ModelPricingType  `json:"pricingType"`
	IsEnabled    bool                     `json:"isEnabled"`
	Extra        models.AIModelExtra      `json:"extra"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
}
