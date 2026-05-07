package tencentcodingplan

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "tencent-coding-plan"

const DefaultBaseURL = "https://api.lkeap.cloud.tencent.com"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/coding/v3/chat/completions",
	types.FormatClaudeMessages: "/coding/anthropic/v1/messages",
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
			},
			DefaultFormatPathMap,
		),
	}
}

func (c *Client) Models() ([]string, error) {
	return c.FetchModels(context.Background(), c.BaseURL+"/coding/v3/models")
}
