package helpers

import (
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

func getType(in json.RawMessage) string {
	if gjson.GetBytes(in, "type").Exists() {
		return gjson.GetBytes(in, "type").String()
	}
	return ""
}

func getName(in json.RawMessage) string {
	if gjson.GetBytes(in, "name").Exists() {
		return gjson.GetBytes(in, "name").String()
	}
	return ""
}

func mustMarshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
