package models

import (
	"database/sql/driver"
	"time"

	"github.com/goccy/go-json"
)

const (
	ObservationWindow3HMinutes  = 180
	ObservationBucket15MMinutes = 15
	ObservationBucketCount      = ObservationWindow3HMinutes / ObservationBucket15MMinutes
)

type ObservationWindow3H struct {
	WindowMinutes int                    `json:"windowMinutes"`
	BucketMinutes int                    `json:"bucketMinutes"`
	Buckets       []ObservationBucket15M `json:"buckets"`
}

type ObservationBucket15M struct {
	StartAt                   time.Time `json:"startAt"`
	EndAt                     time.Time `json:"endAt"`
	InputToken                int64     `json:"inputToken"`
	OutputToken               int64     `json:"outputToken"`
	CachedCreationInputTokens int64     `json:"cachedCreationInputTokens"`
	CachedReadInputTokens     int64     `json:"cachedReadInputTokens"`
	RequestSuccess            int64     `json:"requestSuccess"`
	RequestFailed             int64     `json:"requestFailed"`
	TotalCostMicros           int64     `json:"totalCostMicros"`
	AvgTTFT                   int       `json:"avgTTFT"`
	AvgTPS                    float64   `json:"avgTPS"`
}

func NewObservationWindow3H() ObservationWindow3H {
	return ObservationWindow3H{
		WindowMinutes: ObservationWindow3HMinutes,
		BucketMinutes: ObservationBucket15MMinutes,
		Buckets:       make([]ObservationBucket15M, 0, ObservationBucketCount),
	}
}

func (w ObservationWindow3H) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func (w *ObservationWindow3H) Scan(value interface{}) error {
	if value == nil {
		*w = NewObservationWindow3H()
		return nil
	}
	if err := unmarshalJSONValue(value, w); err != nil {
		return err
	}
	if w.WindowMinutes == 0 {
		w.WindowMinutes = ObservationWindow3HMinutes
	}
	if w.BucketMinutes == 0 {
		w.BucketMinutes = ObservationBucket15MMinutes
	}
	if w.Buckets == nil {
		w.Buckets = make([]ObservationBucket15M, 0, ObservationBucketCount)
	}
	return nil
}
