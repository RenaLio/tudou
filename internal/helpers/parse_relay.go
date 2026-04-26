package helpers

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func GetModelName(body []byte) string {
	model := gjson.GetBytes(body, "model")
	return model.String()
}

func GetStream(body []byte) bool {
	stream := gjson.GetBytes(body, "stream")
	return stream.Bool()
}

func SetModelName(body []byte, model string) []byte {
	body, _ = sjson.SetBytes(body, "model", model)
	return body
}
