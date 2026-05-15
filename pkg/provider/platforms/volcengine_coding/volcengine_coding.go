package volcenginecoding

import (
	"context"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "volcengine-coding"

const DefaultBaseURL = "https://ark.cn-beijing.volces.com/api/coding"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/v3/chat/completions",
	types.FormatClaudeMessages: "/v1/messages",
}

var LocalModelList = modelcatalog.MustLoad(PlatformId)

type Client struct {
	*base.Client
	httpC *http.Client
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
		httpC: httpC,
	}
}

// Models fetches remote models and returns the union with LocalModelList.
func (c *Client) Models() ([]string, error) {
	remoteModels, err := c.FetchModels(context.Background(), c.BaseURL+"/v3/models")
	if err != nil {
		plog.Error("volcengine.remote.models", "err", err)
		return LocalModelList, nil
	}

	modelSet := make(map[string]struct{}, len(remoteModels)+len(LocalModelList))
	merged := make([]string, 0, len(remoteModels)+len(LocalModelList))

	// Keep local order stable.
	for _, m := range LocalModelList {
		if _, ok := modelSet[m]; ok {
			continue
		}
		modelSet[m] = struct{}{}
		merged = append(merged, m)
	}

	// Append remote-only models.
	for _, m := range remoteModels {
		if _, ok := modelSet[m]; ok {
			continue
		}
		modelSet[m] = struct{}{}
		merged = append(merged, m)
	}

	return merged, nil
}
