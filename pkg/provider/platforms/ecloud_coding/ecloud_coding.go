package ecloudcoding

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "eCloud_coding"

const DefaultBaseURL = "https://zhenze-huhehaote.cmecloud.cn"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatClaudeMessages: "/api/coding/v1/message",
	types.FormatChatCompletion: "/api/coding/v1/chat/completions",
}

var ModelList = []string{"cm-code-latest", "minimax-m2.5", "glm-5.1"}

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
