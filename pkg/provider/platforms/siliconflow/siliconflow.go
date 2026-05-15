package siliconflow

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "siliconflow"

const DefaultBaseURL = "https://api.siliconflow.cn"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion:   "/v1/chat/completions",
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
				types.AbilityClaudeMessages,
				types.AbilityEmbeddings,
			},
			DefaultFormatPathMap,
		),
	}
}

func (c *Client) Models() ([]string, error) {
	return c.FetchModels(context.Background(), c.BaseURL+"/v1/models")
}
