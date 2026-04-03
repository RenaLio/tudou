package common

import (
	"github.com/goccy/go-json"
)

func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func UnmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func UnmarshalJSONType[T any](data []byte) (T, error) {
	var v T
	if err := UnmarshalJSON(data, &v); err != nil {
		return v, err
	}
	return v, nil
}

func MustMarshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
