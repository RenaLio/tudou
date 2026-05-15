package kimiforcoding

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/modelcatalog"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
)

const PlatformId = "kimi-for-coding"
const defaultCLIUserAgent = "claude-code/2.1.116 (cli)"

const DefaultBaseURL = "https://api.kimi.com/coding"

var SupportedModelList = modelcatalog.MustLoad(PlatformId)

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion: "/v1/chat/completions",
	types.FormatClaudeMessages: "/v1/messages",
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
			[]types.Ability{
				types.AbilityChatCompletions,
				types.AbilityClaudeMessages,
			},
			DefaultFormatPathMap,
		),
		httpC: httpC,
	}
}

func (c *Client) Execute(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	if req == nil {
		return c.Client.Execute(ctx, req, cb)
	}
	workReq := common.CloneRequest(req)
	if workReq.Headers == nil {
		workReq.Headers = http.Header{}
	}
	normalizeKimiUserAgent(workReq.Headers)
	return c.Client.Execute(ctx, workReq, cb)
}

// Models fetches remote models (for API key availability verification) and
// returns a union with docs-defined static models.
func (c *Client) Models() ([]string, error) {
	httpc := c.httpC
	if httpc == nil {
		httpc = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	c.SetAuthHeader(request)
	normalizeKimiUserAgent(request.Header)

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

	remoteSet := make(map[string]struct{}, len(decoded.Models))
	remoteModels := make([]string, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		if _, exists := remoteSet[model.ID]; exists {
			continue
		}
		remoteSet[model.ID] = struct{}{}
		remoteModels = append(remoteModels, model.ID)
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

func normalizeKimiUserAgent(headers http.Header) {
	if headers == nil {
		return
	}
	ua := strings.TrimSpace(headers.Get("User-Agent"))
	if ua == "" || strings.HasPrefix(ua, "Mozilla/") {
		headers.Set("User-Agent", defaultCLIUserAgent)
	}
}
