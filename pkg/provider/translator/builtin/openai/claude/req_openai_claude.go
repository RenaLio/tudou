package claude

import (
	"context"
	"fmt"

	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
	"github.com/tidwall/sjson"
)

func ConvertChatCompletionRequestToClaudeRequest(_ context.Context, req *types.Request) (*types.Request, error) {
	outReq := common.CloneRequest(req)
	var in oaimodel.CreateChatCompletionRequest
	if err := common.UnmarshalJSON(req.Payload, &in); err != nil {
		return nil, err
	}
	// 通用参数
	out := &claudemodel.CreateMessageRequest{
		MaxTokens:    in.MaxCompletionTokens,
		Messages:     make([]claudemodel.MessageParam, 0),
		Model:        in.Model,
		CacheControl: &claudemodel.CacheControlEphemeral{Type: "ephemeral"},
		Stream:       in.Stream,
		Temperature:  in.Temperature,
		TopP:         in.TopP,
		Thinking:     &claudemodel.ThinkingParam{Type: "adaptive"},
		//ServiceTier:   nil,
		//MetaData:      in.Metadata,
		//OutputConfig:  nil,
		//ToolChoice:    nil,
		//Tools:         nil,
		//System:        nil,
	}
	if out.MaxTokens == nil {
		out.MaxTokens = new(int64)
		*out.MaxTokens = 8092
		//out.MaxTokens = new(int64(8092))
	}
	// serviceTier
	if in.ServiceTier == "auto" {
		out.ServiceTier = new("auto")
	}
	// Metadata-cache_key
	if in.PromptCacheKey != "" {
		mp := map[string]any{
			"user_id": map[string]any{
				"prompt_cache_key":  in.PromptCacheKey,
				"safety_identifier": in.SafetyIdentifier,
			},
		}
		out.MetaData = mustMarshal(mp)
	}
	// outputConfig
	outConfig := claudemodel.OutputConfig{
		Effort: "high",
		Format: nil,
	}
	if in.ReasoningEffort != "" {
		// Verbosity
		outConfig.Effort = in.ReasoningEffort
	}
	if in.ResponseFormat != nil && in.ResponseFormat.Type == "json_schema" {
		outConfig.Format = &claudemodel.JSONOutputFormat{Type: "json_schema", Schema: mustMarshal(in.ResponseFormat.JsonSchema)}
	}
	out.OutputConfig = &outConfig

	// 系统提示词
	// 消息
	system, messages := helpers.OpenAIChatMessageToClaude(in.Messages)
	if system != nil {
		out.System = system
	}
	if len(messages) > 0 {
		out.Messages = messages
	}
	// tools
	tools := make([]json.RawMessage, 0, len(in.Tools))
	for _, t := range in.Tools {
		tool := helpers.OpenAIToolToClaudeTool(&t)
		if tool != nil {
			tools = append(tools, tool)
		}
	}
	out.Tools = tools
	// toolChoice
	toolChoice := helpers.OpenAIChoiceToClaudeToolChoice(in.ToolChoice, new(true))

	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	if toolChoice != nil {
		d, err := sjson.Set(string(data), "tool_choice", toolChoice)
		if err != nil {
			return nil, err
		}
		data = []byte(d)
	}
	fmt.Println("req.payload", string(req.Payload))
	fmt.Println("trans.payload", string(data))
	outReq.Payload = data
	return outReq, nil
}

func mustMarshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
