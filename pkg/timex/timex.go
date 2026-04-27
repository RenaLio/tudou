package timex

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

// TimeX 表示一个带时区信息的时间，持久化时以 Unix 毫秒（int64）存储。
type TimeX struct {
	time.Time // 匿名字段，直接继承 time.Time 的所有方法（Format, Add, In 等）
}

func New(t time.Time) TimeX {
	return FromTime(t)
}

func Now() TimeX {
	return FromTime(time.Now())
}

func NowUTC() TimeX {
	return FromTime(time.Now().UTC())
}

func (t TimeX) IsNil() bool {
	return t.IsZero()
}

// FromTime 从 time.Time 构造，原样保留时区。
func FromTime(t time.Time) TimeX {
	if t.IsZero() {
		return TimeX{}
	}
	return TimeX{Time: t}
}

// FromUnixMilli 用 Unix 毫秒构造
func FromUnixMilli(milli int64) TimeX {
	return TimeX{Time: time.UnixMilli(milli)}
}

// ================= GORM & SQL 驱动接口 =================

// GormDataType 告诉 GORM，建表时使用 INTEGER 类型 (SQLite, MySQL通用)
func (TimeX) GormDataType() string {
	return "INTEGER"
}

// Value 实现 driver.Valuer，Go -> 数据库
func (t TimeX) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil // 完美：零值存为 SQL NULL，避免 1970 幽灵数据
	}
	return t.UnixMilli(), nil // 推荐使用毫秒
}

// Scan 实现 sql.Scanner，数据库 -> Go
func (t *TimeX) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	var milli int64
	var err error
	switch v := value.(type) {
	case int64:
		milli = v
	case float64:
		milli = int64(v) // SQLite 驱动有时候会把数字解析为 float64，你做了兼容，非常赞！
	case int:
		milli = int64(v)
	case uint64:
		milli = int64(v)
	case []byte:
		milli, err = strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("timex: cannot scan %T into TimeX: %w", value, err)
		}
	default:
		return fmt.Errorf("timex: cannot scan %T into TimeX", value)
	}

	t.Time = time.UnixMilli(milli)
	return nil
}

// ================= JSON 序列化接口 =================

// MarshalJSON 序列化为前端可见的格式
func (t TimeX) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	// 输出纯数字的毫秒时间戳，例如 1700000000000
	return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
}

func (t *TimeX) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	milli, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(milli)
	return nil
}
