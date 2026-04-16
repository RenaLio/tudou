package v1

import "time"

type UpsertChannelStatsRequest struct {
	ChannelID                 int64   `json:"channelID,string" binding:"required"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	AvgTTFT                   int     `json:"avgTTFT"`
	AvgTPS                    float64 `json:"avgTPS"`
}

type ChannelStatsResponse struct {
	ChannelID                 int64   `json:"channelID,string"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	AvgTTFT                   int     `json:"avgTTFT"`
	AvgTPS                    float64 `json:"avgTPS"`
}

type UpsertChannelModelStatsRequest struct {
	ChannelID                 int64   `json:"channelID,string" binding:"required"`
	Model                     string  `json:"model" binding:"required"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	AvgTTFT                   int     `json:"avgTTFT"`
	AvgTPS                    float64 `json:"avgTPS"`
}

type ChannelModelStatsResponse struct {
	ChannelID                 int64   `json:"channelID,string"`
	Model                     string  `json:"model"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	AvgTTFT                   int     `json:"avgTTFT"`
	AvgTPS                    float64 `json:"avgTPS"`
}

type UpsertTokenStatsRequest struct {
	TokenID                   int64 `json:"tokenID,string" binding:"required"`
	InputToken                int64 `json:"inputToken"`
	OutputToken               int64 `json:"outputToken"`
	CachedCreationInputTokens int64 `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `json:"cachedReadInputTokens"`
	RequestSuccess            int64 `json:"requestSuccess"`
	RequestFailed             int64 `json:"requestFailed"`
	TotalCostMicros           int64 `json:"totalCostMicros"`
}

type TokenStatsResponse struct {
	TokenID                   int64 `json:"tokenID,string"`
	InputToken                int64 `json:"inputToken"`
	OutputToken               int64 `json:"outputToken"`
	CachedCreationInputTokens int64 `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `json:"cachedReadInputTokens"`
	RequestSuccess            int64 `json:"requestSuccess"`
	RequestFailed             int64 `json:"requestFailed"`
	TotalCostMicros           int64 `json:"totalCostMicros"`
}

type UpsertUserStatsRequest struct {
	UserID                    int64 `json:"userID,string" binding:"required"`
	InputToken                int64 `json:"inputToken"`
	OutputToken               int64 `json:"outputToken"`
	CachedCreationInputTokens int64 `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `json:"cachedReadInputTokens"`
	RequestSuccess            int64 `json:"requestSuccess"`
	RequestFailed             int64 `json:"requestFailed"`
	TotalCostMicros           int64 `json:"totalCostMicros"`
}

type UserStatsResponse struct {
	UserID                    int64 `json:"userID,string"`
	InputToken                int64 `json:"inputToken"`
	OutputToken               int64 `json:"outputToken"`
	CachedCreationInputTokens int64 `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64 `json:"cachedReadInputTokens"`
	RequestSuccess            int64 `json:"requestSuccess"`
	RequestFailed             int64 `json:"requestFailed"`
	TotalCostMicros           int64 `json:"totalCostMicros"`
}

type UpsertUserUsageDailyStatsRequest struct {
	UserID                    int64  `json:"userID,string" binding:"required"`
	Date                      string `json:"date" binding:"required"`
	InputToken                int64  `json:"inputToken"`
	OutputToken               int64  `json:"outputToken"`
	CachedCreationInputTokens int64  `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64  `json:"cachedReadInputTokens"`
	RequestSuccess            int64  `json:"requestSuccess"`
	RequestFailed             int64  `json:"requestFailed"`
	TotalCostMicros           int64  `json:"totalCostMicros"`
}

type ListUserUsageDailyStatsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	OrderBy  string `form:"orderBy"`
	UserID   int64  `form:"userID"`
	DateFrom string `form:"dateFrom"`
	DateTo   string `form:"dateTo"`
}

type UserUsageDailyStatsResponse struct {
	ID                        int64     `json:"id,string"`
	UserID                    int64     `json:"userID,string"`
	Date                      string    `json:"date"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type UpsertUserUsageHourlyStatsRequest struct {
	UserID                    int64  `json:"userID,string" binding:"required"`
	Date                      string `json:"date" binding:"required"`
	Hour                      int    `json:"hour" binding:"required"`
	InputToken                int64  `json:"inputToken"`
	OutputToken               int64  `json:"outputToken"`
	CachedCreationInputTokens int64  `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64  `json:"cachedReadInputTokens"`
	RequestSuccess            int64  `json:"requestSuccess"`
	RequestFailed             int64  `json:"requestFailed"`
	TotalCostMicros           int64  `json:"totalCostMicros"`
}

type ListUserUsageHourlyStatsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	OrderBy  string `form:"orderBy"`
	UserID   int64  `form:"userID"`
	DateFrom string `form:"dateFrom"`
	HourFrom int    `form:"hourFrom"`
	DateTo   string `form:"dateTo"`
	HourTo   int    `form:"hourTo"`
}

type UserUsageHourlyStatsResponse struct {
	ID                        int64     `json:"id,string"`
	UserID                    int64     `json:"userID,string"`
	Date                      string    `json:"date"`
	Hour                      int       `json:"hour"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}
