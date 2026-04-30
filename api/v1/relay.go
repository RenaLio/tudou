package v1

import (
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

// FetchModelRequest 后台"测试渠道，拉上游模型列表"的请求体。
type FetchModelRequest struct {
	Type    models.ChannelType `json:"type" binding:"required"`
	BaseURL string             `json:"baseURL" binding:"required"`
	APIKey  string             `json:"apiKey" binding:"required"`
}

// RelayFormatOf 根据 HTTP 路径返回对应 provider Format；未命中返回空串。
func RelayFormatOf(path string) types.Format {
	switch path {
	case "/v1/chat/completions":
		return types.FormatChatCompletion
	case "/v1/messages":
		return types.FormatClaudeMessages
	case "/v1/responses":
		return types.FormatOpenAIResponses
	default:
		return ""
	}
}

type RelayListResp[T any] struct {
	Object string `json:"object"`
	Data   []T    `json:"data"`
}

type RelayModelItemResp struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
