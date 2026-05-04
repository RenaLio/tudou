package openai

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "openai"

const DefaultBaseURL = "https://api.openai.com"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion:         "/v1/chat/completions",
	types.FormatOpenAIResponses:        "/v1/responses",
	types.FormatOpenAIResponsesCompact: "/v1/responses/compact",
	types.FormatOpenAIEmbeddings:       "/v1/embeddings",
}

type Client struct {
	*base.Client
}

func NewClient(httpC *http.Client, baseURL string, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	client := base.NewClient(
		httpC,
		baseURL,
		apiKey,
		PlatformId,
		[]types.Ability{
			types.AbilityChatCompletions,
			types.AbilityResponses,
			types.AbilityResponsesCompact,
			types.AbilityEmbeddings,
		},
		DefaultFormatPathMap,
	)
	return &Client{Client: client}
}
