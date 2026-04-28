package v1

import "time"

type UpsertChannelStatsRequest struct {
	ChannelID                 int64   `json:"channelID,string" binding:"required"`
	ChannelName               string  `json:"channelName"`
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
	ChannelID                 int64                       `json:"channelID,string"`
	ChannelName               string                      `json:"channelName"`
	InputToken                int64                       `json:"inputToken"`
	OutputToken               int64                       `json:"outputToken"`
	CachedCreationInputTokens int64                       `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64                       `json:"cachedReadInputTokens"`
	RequestSuccess            int64                       `json:"requestSuccess"`
	RequestFailed             int64                       `json:"requestFailed"`
	TotalCostMicros           int64                       `json:"totalCostMicros"`
	TotalCost                 float64                     `json:"totalCost"`
	AvgTTFT                   int                         `json:"avgTTFT"`
	AvgTPS                    float64                     `json:"avgTPS"`
	Window3H                  ObservationWindow3HResponse `json:"window3h"`
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
	ChannelID                 int64                       `json:"channelID,string"`
	Model                     string                      `json:"model"`
	InputToken                int64                       `json:"inputToken"`
	OutputToken               int64                       `json:"outputToken"`
	CachedCreationInputTokens int64                       `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64                       `json:"cachedReadInputTokens"`
	RequestSuccess            int64                       `json:"requestSuccess"`
	RequestFailed             int64                       `json:"requestFailed"`
	TotalCostMicros           int64                       `json:"totalCostMicros"`
	TotalCost                 float64                     `json:"totalCost"`
	AvgTTFT                   int                         `json:"avgTTFT"`
	AvgTPS                    float64                     `json:"avgTPS"`
	Window3H                  ObservationWindow3HResponse `json:"window3h"`
}

type ObservationWindow3HResponse struct {
	WindowMinutes int                            `json:"windowMinutes"`
	BucketMinutes int                            `json:"bucketMinutes"`
	Buckets       []ObservationBucket15MResponse `json:"buckets"`
}

type ObservationBucket15MResponse struct {
	StartAt                   time.Time `json:"startAt"`
	EndAt                     time.Time `json:"endAt"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	TotalCost                 float64   `json:"totalCost"`
	AvgTTFT                   int       `json:"avgTTFT"`
	AvgTPS                    float64   `json:"avgTPS"`
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
	TokenID                   int64   `json:"tokenID,string"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	TotalCost                 float64 `json:"totalCost"`
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
	UserID                    int64   `json:"userID,string"`
	InputToken                int64   `json:"inputToken"`
	OutputToken               int64   `json:"outputToken"`
	CachedCreationInputTokens int64   `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64   `json:"cachedReadInputTokens"`
	RequestSuccess            int64   `json:"requestSuccess"`
	RequestFailed             int64   `json:"requestFailed"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	TotalCost                 float64 `json:"totalCost"`
}

type UpsertUserUsageDailyStatsRequest struct {
	UserID                    int64     `json:"userID,string" binding:"required"`
	Date                      time.Time `json:"date" binding:"required"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
}

type ListUserUsageDailyStatsRequest struct {
	Page      int        `form:"page"`
	PageSize  int        `form:"pageSize"`
	OrderBy   string     `form:"orderBy"`
	UserID    int64      `form:"userID"`
	StartTime *time.Time `form:"startTime"`
	EndTime   *time.Time `form:"endTime"`
}

type UserUsageDailyStatsResponse struct {
	ID                        int64     `json:"id,string"`
	UserID                    int64     `json:"userID,string"`
	Date                      time.Time `json:"date"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	TotalCost                 float64   `json:"totalCost"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type UpsertUserUsageHourlyStatsRequest struct {
	UserID                    int64     `json:"userID,string" binding:"required"`
	Date                      time.Time `json:"date" binding:"required"`
	Hour                      int       `json:"hour" binding:"required"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
}

type ListUserUsageHourlyStatsRequest struct {
	Page      int        `form:"page"`
	PageSize  int        `form:"pageSize"`
	OrderBy   string     `form:"orderBy"`
	UserID    int64      `form:"userID"`
	StartTime *time.Time `form:"startTime" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime   *time.Time `form:"endTime" time_format:"2006-01-02T15:04:05Z07:00"`
}

type UserUsageHourlyStatsResponse struct {
	ID                        int64     `json:"id,string"`
	UserID                    int64     `json:"userID,string"`
	Date                      time.Time `json:"date"`
	Hour                      int       `json:"hour"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	TotalCost                 float64   `json:"totalCost"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}
