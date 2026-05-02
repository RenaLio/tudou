package minimaxcoding

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "minimax-coding-plan"

const DefaultBaseURL = "https://api.minimaxi.com"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatClaudeMessages: "/anthropic/v1/messages",
	types.FormatChatCompletion: "/v1/chat/completions",
}

var ModelList = []string{"MiniMax-M2.7", "MiniMax-M2.7-highspeed"}

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
			[]types.Ability{types.AbilityClaudeMessages, types.AbilityChatCompletions},
			DefaultFormatPathMap,
		),
	}
}

func (c *Client) Models() ([]string, error) {
	return ModelList, nil
}
