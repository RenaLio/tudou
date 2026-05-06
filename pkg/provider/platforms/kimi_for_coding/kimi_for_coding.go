package kimiforcoding

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "kimi-for-coding"

const DefaultBaseURL = "https://api.kimi.com/coding"

var SupportedModelList = []string{
	"kimi-k2.6",
	"kimi-for-coding",
	"kimi-k2.5",
	"kimi-k2-thinking",
}

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/v1/chat/completions",
	types.FormatClaudeMessages: "/v1/messages",
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

// Models fetches remote models (for API key availability verification) and
// returns a union with docs-defined static models.
func (c *Client) Models() ([]string, error) {
	remoteModels, err := c.FetchModels(context.Background(), c.BaseURL+"/v1/models")
	if err != nil {
		return nil, err
	}

	modelSet := make(map[string]struct{}, len(remoteModels)+len(SupportedModelList))
	merged := make([]string, 0, len(remoteModels)+len(SupportedModelList))

	// Keep docs-defined order stable.
	for _, model := range SupportedModelList {
		if _, ok := modelSet[model]; ok {
			continue
		}
		modelSet[model] = struct{}{}
		merged = append(merged, model)
	}

	// Append remote-only models.
	for _, model := range remoteModels {
		if _, ok := modelSet[model]; ok {
			continue
		}
		modelSet[model] = struct{}{}
		merged = append(merged, model)
	}

	return merged, nil
}
