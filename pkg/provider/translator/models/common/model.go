package common

type ModelItem struct {
	ID                   string   `json:"id"`
	Object               string   `json:"object"`
	Created              int64    `json:"created"`
	OwnedBy              string   `json:"owned_by"`
	SupportEndPointTypes []string `json:"support_end_point_types"`
}
