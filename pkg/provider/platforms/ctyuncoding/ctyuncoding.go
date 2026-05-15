package ctyuncoding

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "ctyuncoding"

const DefaultBaseURL = "https://wishub-x6.ctyun.cn"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/coding/v1/chat/completions",
	types.FormatClaudeMessages: "/coding/v1/messages",
}

var ModelList = modelcatalog.MustLoad(PlatformId)

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
	return append([]string(nil), ModelList...), nil
}
