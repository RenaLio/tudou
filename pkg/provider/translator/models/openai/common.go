package openai

import "encoding/json"

type JSONSchema struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

type (
	IdObject struct {
		Id string `json:"id"`
	}
	TypeObject struct {
		Type string `json:"type"`
	}
	NameObject struct {
		Name string `json:"name"`
	}
	TypeTextObject struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	NameTypeObject struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
)

type (
	Function struct {
		Type         string          `json:"type,omitempty"`
		Name         string          `json:"name"`
		Description  string          `json:"description,omitempty"`
		Parameters   json.RawMessage `json:"parameters,omitempty"`
		Strict       *bool           `json:"strict,omitempty"`
		DeferLoading *bool           `json:"defer_loading,omitempty"`
	}
	Custom struct {
		Type        string          `json:"type,omitempty"`
		Name        string          `json:"name"`
		Format      json.RawMessage `json:"format,omitempty"`
		Description string          `json:"description,omitempty"`

		DeferLoading *bool `json:"defer_loading,omitempty"`
	}
)

type (
	UserLocation struct {
		Type        string       `json:"type,omitempty" default:"approximate"`
		Approximate *Approximate `json:"approximate,omitempty"`
	}
	Approximate struct {
		City     string `json:"city,omitempty"`
		Country  string `json:"country,omitempty"`
		Region   string `json:"region,omitempty"`
		Timezone string `json:"timezone,omitempty"`
	}
)
