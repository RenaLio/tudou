package models

import (
	"fmt"

	"github.com/goccy/go-json"
)

func unmarshalJSONValue(value interface{}, target interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, target)
	case string:
		return json.Unmarshal([]byte(v), target)
	default:
		return fmt.Errorf("unsupported JSON scan type: %T", value)
	}
}
