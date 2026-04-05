package models

import (
	"database/sql/driver"

	"time"

	"github.com/goccy/go-json"
)

// RequestStatus 请求状态
type RequestStatus string

const (
	RequestStatusSuccess RequestStatus = "success" // 成功
	RequestStatusFail    RequestStatus = "fail"    // 失败
)

// RequestExtra 请求额外信息
type RequestExtra struct {
	RequestBody  string            `json:"requestBody,omitempty"`  // 请求体（脱敏）
	ResponseBody string            `json:"responseBody,omitempty"` // 响应体（脱敏）
	Headers      map[string]string `json:"headers,omitempty"`      // 请求头
	IP           string            `json:"ip,omitempty"`           // 客户端IP
	UserAgent    string            `json:"userAgent,omitempty"`    // 客户端UA
	RequestPath  string            `json:"requestPath,omitempty"`  // 请求路径
}

// Value 实现 driver.Valuer 接口
func (e *RequestExtra) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan 实现 sql.Scanner 接口
func (e *RequestExtra) Scan(value interface{}) error {
	if value == nil {
		*e = RequestExtra{}
		return nil
	}
	return unmarshalJSONValue(value, e)
}

type Pricing struct {
	InputPrice       float64 `json:"inputPrice,omitempty"`       // 输入价格 (per 1M tokens)
	OutputPrice      float64 `json:"outputPrice,omitempty"`      // 输出价格 (per 1M tokens)
	CacheCreatePrice float64 `json:"cacheCreatePrice,omitempty"` // 缓存创建价格 (per 1M tokens)
	CacheReadPrice   float64 `json:"cacheReadPrice,omitempty"`   // 缓存读取价格 (per 1M tokens)
	PerRequestPrice  float64 `json:"perRequestPrice,omitempty"`  // 按次计费价格 (per request)
}

func (p *Pricing) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Pricing) Scan(value interface{}) error {
	if value == nil {
		*p = Pricing{}
		return nil
	}
	return unmarshalJSONValue(value, p)
}

type ProviderDetail struct {
	Provider      string `json:"provider,omitempty"`      // 提供商
	RequestFormat string `json:"requestFormat,omitempty"` // 请求格式
	TransFormat   string `json:"transFormat,omitempty"`   // 转换格式
	// 其他字段...
}

func (p *ProviderDetail) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *ProviderDetail) Scan(value interface{}) error {
	if value == nil {
		*p = ProviderDetail{}
		return nil
	}
	return unmarshalJSONValue(value, p)
}

// RequestLog 请求日志模型
type RequestLog struct {
	ID                        int64          `gorm:"primaryKey;column:id;type:bigint;autoIncrement:false" json:"id,string"`
	RequestID                 string         `gorm:"column:request_id;type:varchar(128);index:idx_reqlog_request_id" json:"requestID"`
	UserID                    int64          `gorm:"column:user_id;type:bigint;not null;index:idx_reqlog_user" json:"userID,string"`
	TokenID                   int64          `gorm:"column:token_id;type:bigint;not null;index:idx_reqlog_token" json:"tokenID,string"`
	GroupID                   int64          `gorm:"column:group_id;type:bigint;index:idx_reqlog_group" json:"groupID,string"`
	ChannelID                 int64          `gorm:"column:channel_id;type:bigint;index:idx_reqlog_channel" json:"channelID,string"`
	ChannelName               string         `gorm:"column:channel_name;type:varchar(128)" json:"channelName"`                       // 渠道名称，冗余字段，只是为了展示时，不再查询channel
	ChannelPriceRate          float64        `gorm:"column:channel_price_rate;type:decimal(20,6);default:0" json:"channelPriceRate"` // 渠道价格比例
	Model                     string         `gorm:"column:model;type:varchar(128);index:idx_reqlog_model" json:"model"`
	UpstreamModel             string         `gorm:"column:upstream_model;type:varchar(128)" json:"upstreamModel"`
	InputToken                int64          `gorm:"column:input_token;type:bigint;default:0;comment:输入token数" json:"inputToken"`
	OutputToken               int64          `gorm:"column:output_token;type:bigint;default:0;comment:输出token数" json:"outputToken"`
	CachedCreationInputTokens int64          `gorm:"column:cached_creation_input_tokens;type:bigint;default:0;comment:缓存创建输入token数" json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64          `gorm:"column:cached_read_input_tokens;type:bigint;default:0;comment:缓存读取输入token数" json:"cachedReadInputTokens"`
	Pricing                   Pricing        `gorm:"column:pricing;type:json" json:"pricing"`
	CostMicros                int64          `gorm:"column:cost_micros;type:bigint;default:0;comment:成本，单位 micros" json:"costMicros"` // 成本，单位 micros
	Status                    RequestStatus  `gorm:"column:status;type:varchar(32);not null;index:idx_reqlog_status" json:"status"`
	TTFT                      int64          `gorm:"column:ttft;type:bigint;default:0" json:"ttft"`                  // 首字时间(ms)
	TransferTime              int64          `gorm:"column:transfer_time;type:bigint;default:0" json:"transferTime"` // 传输时间(ms)
	ErrorCode                 string         `gorm:"column:error_code;type:varchar(64)" json:"errorCode,omitempty"`
	ErrorMsg                  string         `gorm:"column:error_msg;type:text" json:"errorMsg,omitempty"`
	IsStream                  bool           `gorm:"column:is_stream;type:boolean;default:false" json:"isStream"`
	Extra                     RequestExtra   `gorm:"column:extra;type:json" json:"extra"`
	ProviderDetail            ProviderDetail `gorm:"column:provider_detail;type:json" json:"providerDetail"`
	CreatedAt                 time.Time      `gorm:"column:created_at;type:timestamp;not null;index:idx_reqlog_created" json:"createdAt"`
}

// TableName 指定表名
func (*RequestLog) TableName() string {
	return "request_logs"
}
