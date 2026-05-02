package codingplan

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "coding-plan-adapter"

const DefaultBaseURL = "https://api.example.com/v1"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion:  "/chat/completions",
	types.AbilityClaudeMessages: "/messages",
}

type Client struct {
	*base.Client
}

func NewClient(httpC *http.Client, baseURL string, apiKey string) *Client {
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
	reqUrl := c.Client.BaseURL + "/models"
	return c.Client.FetchModels(context.Background(), reqUrl)
}
