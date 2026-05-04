package base

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/perrors"
	"github.com/RenaLio/tudou/pkg/provider/phelpers"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator"
	_ "github.com/RenaLio/tudou/pkg/provider/translator/builtin"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
)

const defaultAnthropicVersion = "2023-06-01"

type Client struct {
	httpC         *http.Client
	BaseURL       string
	ApiKey        string
	Id            string
	abilities     []types.Ability
	abMap         map[types.Ability]struct{}
	formatPathMap map[types.Format]string
}

func NewClient(
	httpC *http.Client,
	baseURL, apiKey string,
	Id string,
	abilities []types.Ability,
	pathMap map[types.Format]string,
) *Client {
	abMap := make(map[types.Ability]struct{})
	for _, ab := range abilities {
		abMap[ab] = struct{}{}
	}
	if pathMap == nil {
		pathMap = defaultFormatPathMap
	}
	return &Client{
		httpC:         httpC,
		BaseURL:       baseURL,
		ApiKey:        apiKey,
		Id:            Id,
		abilities:     abilities,
		abMap:         abMap,
		formatPathMap: pathMap,
	}
}

func (c *Client) Identifier() string {
	return c.Id
}

func (c *Client) Execute(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	if req == nil {
		return nil, &perrors.Error{
			Kind:        perrors.KindInvalidRequest,
			Op:          "client.execute.start",
			Provider:    c.Identifier(),
			SafeMessage: "request is nil",
			Cause:       nil,
		}
	}
	workReq := common.CloneRequest(req)
	switch req.Format {
	case types.FormatChatCompletion, types.FormatOpenAIResponses, types.FormatClaudeMessages:
		return c.Chat(ctx, workReq, cb)
	case types.FormatOpenAIEmbeddings:
		if !c.supportsFormat(types.FormatOpenAIEmbeddings) {
			return nil, perrors.New(perrors.KindUnsupportedFormat, "client.execute.start", c.Identifier(), "", "", nil)
		}
		return c.dispatchByFormat(ctx, workReq, workReq, cb)
	default:
		return nil, perrors.New(perrors.KindUnsupportedFormat, "client.execute.start", c.Identifier(), "", "", nil)
	}
}

func (c *Client) Chat(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	originReq := common.CloneRequest(req)

	if req.Headers == nil {
		req.Headers = http.Header{}
	}

	originFormat := req.Format

	if !c.supportsChat() {
		return nil, errors.New("provider does not support chat")
	}

	targetFormat, needTransform, err := c.resolveExecutionFormat(req.Format)
	if err != nil {
		return nil, err
	}

	if needTransform {
		req, err = translator.TransformRequest(ctx, req, targetFormat)
		if err != nil {
			return nil, fmt.Errorf("transform request %s -> %s: %w", originReq.Format, targetFormat, err)
		}
	}

	req.Format = targetFormat

	resp, err := c.dispatchByFormat(ctx, originReq, req, cb)
	if err != nil {
		return nil, err
	}

	if needTransform {
		resp, err = translator.TransformResponse(ctx, originReq, resp, originFormat)
		if err != nil {
			return nil, fmt.Errorf("transform response %s -> %s: %w", targetFormat, originFormat, err)
		}
	}

	return resp, nil
}

func (c *Client) resolveExecutionFormat(source types.Format) (types.Format, bool, error) {
	if c.supportsFormat(source) {
		return source, false, nil
	}

	priority := []types.Format{
		types.FormatOpenAIResponses,
		types.FormatClaudeMessages,
		types.FormatChatCompletion,
	}
	for _, target := range priority {
		if !c.supportsFormat(target) {
			continue
		}
		if translator.CanTransform(source, target) {
			return target, true, nil
		}
	}

	for _, ability := range c.abilities {
		target, ok := phelpers.AbilityToFormat(ability)
		if !ok {
			continue
		}
		if translator.CanTransform(source, target) {
			return target, true, nil
		}
	}

	return "", false, fmt.Errorf("unsupported format %s for provider %s", source, c.Identifier())
}

func (c *Client) dispatchByFormat(ctx context.Context, originReq *types.Request, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	urlPath, ok := c.formatPathMap[req.Format]
	if !ok {
		return nil, errors.New("unsupported format")
	}
	reqUrl, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	reqUrl.Path = path.Join(reqUrl.Path, urlPath)

	if req.IsStream {
		req.Headers.Set("Accept", "text/event-stream")
		req.Headers.Set("Cache-Control", "no-cache")
		req.Headers.Set("Connection", "keep-alive")
	}

	switch req.Format {
	case types.FormatChatCompletion:
		req.Headers.Set("Authorization", "Bearer "+c.ApiKey)
		req.Headers.Set("Content-Type", "application/json")
		return c.ChatCompletion(ctx, reqUrl.String(), originReq, req, cb)
	case types.FormatOpenAIResponses:
		req.Headers.Set("Authorization", "Bearer "+c.ApiKey)
		req.Headers.Set("Content-Type", "application/json")
		return c.Responses(ctx, reqUrl.String(), originReq, req, cb)
	case types.FormatOpenAIEmbeddings:
		req.Headers.Set("Authorization", "Bearer "+c.ApiKey)
		req.Headers.Set("Content-Type", "application/json")
		return c.ChatCompletion(ctx, reqUrl.String(), originReq, req, cb)
	case types.FormatClaudeMessages:
		req.Headers.Set("X-API-Key", c.ApiKey)
		req.Headers.Set("Authorization", "Bearer "+c.ApiKey)
		if req.Headers.Get("Anthropic-Version") == "" {
			req.Headers.Set("Anthropic-Version", defaultAnthropicVersion)
		}
		req.Headers.Set("Content-Type", "application/json")
		return c.ClaudeMessages(ctx, reqUrl.String(), originReq, req, cb)
	default:
		return nil, errors.New("unsupported format")
	}
}

func (c *Client) supportsFormat(format types.Format) bool {
	ability, ok := phelpers.FormatToAbility(format)
	if !ok {
		return false
	}
	return c.HasAbility(ability)
}

func (c *Client) supportsChat() bool {
	return c.HasAbility(types.AbilityChatCompletions) ||
		c.HasAbility(types.AbilityResponses) ||
		c.HasAbility(types.AbilityClaudeMessages)
}

func (c *Client) Models() ([]string, error) {
	return c.FetchModels(context.Background(), c.BaseURL+"/v1/models")
}

func (c *Client) SetAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("X-API-Key", c.ApiKey)
}

func (c *Client) FetchModels(ctx context.Context, reqURL string) ([]string, error) {
	var models []string
	var err error
	modelSet := make(map[string]struct{})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		plog.Error("get.models.request", "err", err)
		return models, perrors.New(perrors.KindBuildRequest, "get.models.request", c.Identifier(), "", "", err)
	}
	c.SetAuthHeader(request)
	response, err := c.httpC.Do(request)
	if err != nil {
		plog.Error("get.models.response", "err", err)
		return models, perrors.New(perrors.KindFetchResponse, "get.models.response", c.Identifier(), "", "", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(response.Body)
		plog.Error("get.models.response", "data", string(data))
		return models, &perrors.Error{
			Kind:        perrors.KindBadResponse,
			Op:          "get.models.response",
			Provider:    c.Identifier(),
			HTTPStatus:  response.StatusCode,
			SafeMessage: fmt.Sprintf("unexpected status code: %d: %s", response.StatusCode, string(data)),
			Cause:       nil,
		}
	}
	type ModelResp struct {
		Models []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	var modelResp ModelResp
	if err = json.NewDecoder(response.Body).Decode(&modelResp); err != nil {
		plog.Error("get.models.decode", "err", err)
		return models, perrors.New(perrors.KindInternal, "get.models.decode", c.Identifier(), "", "", err)
	}
	for _, model := range modelResp.Models {
		if _, exists := modelSet[model.ID]; exists {
			continue
		}
		modelSet[model.ID] = struct{}{}
		models = append(models, model.ID)
	}
	return models, nil
}

func (c *Client) Abilities() []types.Ability {
	return c.abilities
}

func (c *Client) HasAbility(ability types.Ability) bool {
	_, ok := c.abMap[ability]
	return ok
}
