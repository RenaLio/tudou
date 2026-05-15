package alibabacodingplancn

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "alibaba-coding-plan-cn"

const DefaultBaseURL = "https://coding.dashscope.aliyuncs.com"

var SupportedModelList = modelcatalog.MustLoad(PlatformId)

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatClaudeMessages: "/apps/anthropic/v1/messages",
	types.FormatChatCompletion: "/v1/chat/completions",
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
				types.AbilityClaudeMessages,
				types.AbilityChatCompletions,
			},
			DefaultFormatPathMap,
		),
	}
}

// Models returns a static docs-defined model list because this platform
// does not expose a direct public models endpoint for fetching.
func (c *Client) Models() ([]string, error) {
	return append([]string(nil), SupportedModelList...), nil
}
