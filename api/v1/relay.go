package v1

import "github.com/RenaLio/tudou/internal/models"

type FetchModelRequest struct {
	Type    models.ChannelType `json:"type" binding:"required"`
	BaseURL string             `json:"baseURL" binding:"required"`
	APIKey  string             `json:"apiKey" binding:"required"`
}
