package v1

type SelectOptions[K, V any] struct {
	Options []SelectOptionItem[K, V] `json:"options"`
}

type SelectOptionItem[K, V any] struct {
	Key   K              `json:"key"`
	Value V              `json:"value"`
	Extra map[string]any `json:"extra"`
}
