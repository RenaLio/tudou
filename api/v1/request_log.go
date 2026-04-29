package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type ListRequestLogsRequest struct {
	Page          int    `form:"page" binding:"omitempty,min=1"`
	PageSize      int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
	OrderBy       string `form:"orderBy"`
	Keyword       string `form:"keyword"`
	RequestID     string `form:"requestId"`
	UserID        int64  `form:"userId"`
	TokenID       int64  `form:"tokenId"`
	GroupID       int64  `form:"groupId"`
	ChannelID     int64  `form:"channelId"`
	Model         string `form:"model"`
	UpstreamModel string `form:"upstreamModel"`
	Status        string `form:"status"`
	IsStream      *bool  `form:"isStream"`
	DateFrom      string `form:"dateFrom"`
	DateTo        string `form:"dateTo"`
}

type RequestLogResponse struct {
	ID                        int64                 `json:"id,string"`
	RequestID                 string                `json:"requestId"`
	UserID                    int64                 `json:"userId,string"`
	TokenID                   int64                 `json:"tokenId,string"`
	TokenName                 string                `json:"tokenName"`
	GroupID                   int64                 `json:"groupId,string"`
	GroupName                 string                `json:"groupName"`
	ChannelID                 int64                 `json:"channelId,string"`
	ChannelName               string                `json:"channelName"`
	ChannelPriceRate          float64               `json:"channelPriceRate"`
	Model                     string                `json:"model"`
	UpstreamModel             string                `json:"upstreamModel"`
	InputToken                int64                 `json:"inputToken"`
	OutputToken               int64                 `json:"outputToken"`
	CachedCreationInputTokens int64                 `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64                 `json:"cachedReadInputTokens"`
	Pricing                   models.ModelPricing   `json:"pricing"`
	CostMicros                int64                 `json:"costMicros"`
	Status                    models.RequestStatus  `json:"status"`
	TTFT                      int64                 `json:"ttft"`
	TransferTime              int64                 `json:"transferTime"`
	ErrorCode                 string                `json:"errorCode,omitempty"`
	ErrorMsg                  string                `json:"errorMsg,omitempty"`
	IsStream                  bool                  `json:"isStream"`
	Extra                     models.RequestExtra   `json:"extra"`
	ProviderDetail            models.ProviderDetail `json:"providerDetail"`
	CreatedAt                 time.Time             `json:"createdAt"`
}
