package jdcoding

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "jd_coding"

const DefaultBaseURL = "https://modelservice.jdcloud.com"

var SupportedModelList = modelcatalog.MustLoad(PlatformId)

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/coding/openai/v1/chat/completions",
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
	return append([]string(nil), SupportedModelList...), nil
}
