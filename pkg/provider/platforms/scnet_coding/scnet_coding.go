package scnetcoding

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
)

const PlatformId = "scnet-coding"

const DefaultBaseURL = "https://api.scnet.cn/api/llm"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/v1/chat/completions",
	types.FormatClaudeMessages: "/anthropic/v1/messages",
}

// LocalModelList is the fixed set of models available under the Coding Plan.
var LocalModelList = []string{
	"MiniMax-M2.5",
	"Qwen3-235B-A22B",
}

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
			[]types.Ability{types.AbilityChatCompletions, types.AbilityClaudeMessages},
			DefaultFormatPathMap,
		),
		httpC: httpC,
	}
}

// Models fetches the remote model list and returns the intersection with
// LocalModelList. Only models that are both remotely available and in the
// local allowlist are returned.
func (c *Client) Models() ([]string, error) {
	httpc := c.httpC
	if httpc == nil {
		httpc = http.DefaultClient
	}

	remoteModels, err := c.fetchRemoteModels(httpc)
	if err != nil {
		plog.Error("scnet.remote.models", "err", err)
		// Fall back to local list when remote is unreachable.
		return LocalModelList, nil
	}

	remoteSet := make(map[string]struct{}, len(remoteModels))
	for _, m := range remoteModels {
		remoteSet[m] = struct{}{}
	}

	// Intersection: keep only local models that also appear remotely.
	result := make([]string, 0, len(LocalModelList))
	for _, m := range LocalModelList {
		if _, ok := remoteSet[m]; ok {
			result = append(result, m)
		}
	}

	// If intersection is empty (e.g. remote list format mismatch), fall back
	// to the local list so the channel is still usable.
	if len(result) == 0 {
		return LocalModelList, nil
	}

	return result, nil
}

func (c *Client) fetchRemoteModels(httpc *http.Client) ([]string, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	c.SetAuthHeader(request)

	response, err := httpc.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s", response.StatusCode, string(data))
	}

	type modelResp struct {
		Models []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	var decoded modelResp
	if err = json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(decoded.Models))
	models := make([]string, 0, len(decoded.Models))
	for _, m := range decoded.Models {
		if _, dup := seen[m.ID]; dup {
			continue
		}
		seen[m.ID] = struct{}{}
		models = append(models, m.ID)
	}
	return models, nil
}
