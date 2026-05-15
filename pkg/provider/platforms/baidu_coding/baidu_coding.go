package baiducoding

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "baidu_coding"

const DefaultBaseURL = "https://qianfan.baidubce.com"

var SupportedModelList = modelcatalog.MustLoad(PlatformId)

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/v2/coding/chat/completions",
	types.FormatClaudeMessages: "/anthropic/coding/v1/messages",
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
	models, err := c.FetchModels(context.Background(), c.BaseURL+"/v2/coding/models")
	if err != nil {
		return append([]string(nil), SupportedModelList...), nil
	}

	modelSet := make(map[string]struct{}, len(models))
	for _, model := range models {
		modelSet[model] = struct{}{}
	}

	// Preserve docs-defined order while filtering unsupported models.
	filtered := make([]string, 0, len(SupportedModelList))
	for _, model := range SupportedModelList {
		if _, ok := modelSet[model]; ok {
			filtered = append(filtered, model)
		}
	}

	if len(filtered) == 0 {
		return append([]string(nil), SupportedModelList...), nil
	}
	return filtered, nil
}
