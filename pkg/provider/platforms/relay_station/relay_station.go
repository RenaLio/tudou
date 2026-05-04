package relaystation

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "relay_station"

const DefaultBaseURL = "https://api.example.com"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion:   "/v1/chat/completions",
	types.FormatOpenAIResponses:  "/v1/responses",
	types.FormatClaudeMessages:   "/v1/messages",
	types.FormatOpenAIEmbeddings: "/v1/embeddings",
}

type Client struct {
	*base.Client
}

func NewClient(httpC *http.Client, baseURL string, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		Client: base.NewClient(
			httpC,
			baseURL,
			apiKey,
			PlatformId,
			[]types.Ability{
				types.AbilityChatCompletions,
				types.AbilityResponses,
				types.AbilityClaudeMessages,
				types.AbilityEmbeddings,
			},
			DefaultFormatPathMap,
		),
	}
}
