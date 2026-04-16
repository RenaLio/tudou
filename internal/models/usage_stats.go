package models

import (
	"time"
)

// UserUsageDailyStats 用量统计模型（按天汇总）
type UserUsageDailyStats struct {
	ID                        int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	UserID                    int64  `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uidx_stats_user_date" json:"userID,string"`
	Date                      string `gorm:"column:date;type:varchar(10);not null;uniqueIndex:uidx_stats_user_date" json:"date"` // 格式：2024-01-01
	InputToken                int64  `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64  `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64  `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64  `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64  `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64  `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64  `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (*UserUsageDailyStats) TableName() string {
	return "user_usage_daily_stats"
}

type UserUsageHourlyStats struct {
	ID                        int64  `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	UserID                    int64  `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uidx_stats_user_date_hour" json:"userID,string"`
	Date                      string `gorm:"column:date;type:varchar(10);not null;uniqueIndex:uidx_stats_user_date_hour" json:"date"` // 格式：2024-01-01
	Hour                      int    `gorm:"column:hour;type:smallint;not null;uniqueIndex:uidx_stats_user_date_hour" json:"hour"`    // 格式：0-23
	InputToken                int64  `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64  `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64  `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64  `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	RequestSuccess            int64  `gorm:"column:request_success;type:bigint;default:0;comment:请求成功数" json:"requestSuccess"`
	RequestFailed             int64  `gorm:"column:request_failed;type:bigint;default:0;comment:请求失败数" json:"requestFailed"`
	TotalCostMicros           int64  `gorm:"column:total_cost_micros;type:bigint;default:0;comment:总成本，单位 micros" json:"totalCostMicros"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (*UserUsageHourlyStats) TableName() string {
	return "user_usage_hourly_stats"
}
