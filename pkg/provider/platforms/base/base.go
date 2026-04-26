package base

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/perrors"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator"
	_ "github.com/RenaLio/tudou/pkg/provider/translator/builtin"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
)

const defaultAnthropicVersion = "2023-06-01"

type Client struct {
	httpC     *http.Client
	baseURL   string
	apiKey    string
	Id        string
	abilities []types.Ability
	abMap     map[types.Ability]struct{}
}

func NewClient(
	httpC *http.Client,
	baseURL, apiKey string,
	Id string,
	abilities []types.Ability,
) *Client {
	abMap := make(map[types.Ability]struct{})
	for _, ab := range abilities {
		abMap[ab] = struct{}{}
	}
	return &Client{
		httpC:     httpC,
		baseURL:   baseURL,
		apiKey:    apiKey,
		Id:        Id,
		abilities: abilities,
		abMap:     abMap,
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
	workReq := cloneRequest(req)
	switch req.Format {
	case types.FormatChatCompletion, types.FormatOpenAIResponses, types.FormatClaudeMessages:
		return c.Chat(ctx, workReq, cb)
	default:
		return nil, perrors.New(perrors.KindUnsupportedFormat, "client.execute.start", c.Identifier(), "", "", nil)
	}
}

func cloneRequest(req *types.Request) *types.Request {
	if req == nil {
		return nil
	}
	cp := *req
	if req.Payload != nil {
		cp.Payload = append([]byte(nil), req.Payload...)
	}

	if req.Headers != nil {
		cp.Headers = req.Headers.Clone()
	} else {
		cp.Headers = make(http.Header)
	}

	if req.FormPayload != nil {
		fp := *req.FormPayload
		if req.FormPayload.Fields != nil {
			fp.Fields = make(map[string]string, len(req.FormPayload.Fields))
			for k, v := range req.FormPayload.Fields {
				fp.Fields[k] = v
			}
		}
		if req.FormPayload.Files != nil {
			fp.Files = make(map[string]*multipart.FileHeader, len(req.FormPayload.Files))
			for k, v := range req.FormPayload.Files {
				fp.Files[k] = v
			}
		}
		cp.FormPayload = &fp
	}
	return &cp
}

func (c *Client) Chat(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	originReq := cloneRequest(req)

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

	plog.Debug("Chat", "origin-format", originReq.Format, "req-format:", req.Format)

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
		target, ok := formatFromAbility(ability)
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
	url := c.GetURLBase(req.Format)
	url = c.baseURL + url
	if req.IsStream {
		req.Headers.Set("Accept", "text/event-stream")
		req.Headers.Set("Cache-Control", "no-cache")
		req.Headers.Set("Connection", "keep-alive")
	}

	switch req.Format {
	case types.FormatChatCompletion:
		req.Headers.Set("Authorization", "Bearer "+c.apiKey)
		req.Headers.Set("Content-Type", "application/json")
		return c.ChatCompletion(ctx, url, originReq, req, cb)
	case types.FormatOpenAIResponses:
		req.Headers.Set("Authorization", "Bearer "+c.apiKey)
		req.Headers.Set("Content-Type", "application/json")
		return c.Responses(ctx, url, originReq, req, cb)
	case types.FormatClaudeMessages:
		req.Headers.Set("x-api-key", c.apiKey)
		if req.Headers.Get("anthropic-version") == "" {
			req.Headers.Set("anthropic-version", defaultAnthropicVersion)
		}
		req.Headers.Set("Content-Type", "application/json")
		return c.ClaudeMessages(ctx, url, originReq, req, cb)
	default:
		return nil, errors.New("unsupported format")
	}
}

func (c *Client) supportsFormat(format types.Format) bool {

	ability, ok := abilityFromFormat(format)
	if !ok {
		return false
	}
	return c.HasAbility(ability)
}

func abilityFromFormat(format types.Format) (types.Ability, bool) {
	switch format {
	case types.FormatChatCompletion:
		return types.AbilityChatCompletions, true
	case types.FormatOpenAIResponses:
		return types.AbilityResponses, true
	case types.FormatClaudeMessages:
		return types.AbilityClaudeMessages, true
	default:
		return "", false
	}
}

func formatFromAbility(ability types.Ability) (types.Format, bool) {
	switch ability {
	case types.AbilityChatCompletions:
		return types.FormatChatCompletion, true
	case types.AbilityResponses:
		return types.FormatOpenAIResponses, true
	case types.AbilityClaudeMessages:
		return types.FormatClaudeMessages, true
	default:
		return "", false
	}
}

func (c *Client) supportsChat() bool {
	if c.HasAbility(types.AbilityChat) {
		return true
	}
	return c.HasAbility(types.AbilityChatCompletions) ||
		c.HasAbility(types.AbilityResponses) ||
		c.HasAbility(types.AbilityClaudeMessages)
}

func (c *Client) Models() ([]string, error) {
	return c.FetchModels(context.Background(), c.baseURL+"/v1/models")
}

func (c *Client) SetOpenAIAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
	c.SetOpenAIAuth(request)
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

func (c *Client) GetURLBase(format types.Format) string {
	return urlBaseMap[format]
}

var urlBaseMap = map[types.Format]string{
	types.FormatChatCompletion: "/v1/chat/completions",
	//types.FormatChatCompletion:   "/chat/completions",
	types.FormatOpenAIResponses:  "/v1/responses",
	types.FormatClaudeMessages:   "/v1/messages",
	types.FormatOpenAIEmbeddings: "/v1/embeddings",
}
