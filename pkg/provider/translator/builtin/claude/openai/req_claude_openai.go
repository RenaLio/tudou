package openai

import (
	"context"
	"encoding/json"

	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

func getType(in json.RawMessage) string {
	return gjson.GetBytes(in, "type").String()
}

func ConvertClaudeRequestToChatCompletionRequest(_ context.Context, req *types.Request) (*types.Request, error) {
	outReq := common.CloneRequest(req)
	var in claudemodel.CreateMessageRequest
	if err := common.UnmarshalJSON(req.Payload, &in); err != nil {
		return nil, err
	}
	// 通用参数
	out := &oaimodel.CreateChatCompletionRequest{
		Model:               in.Model,
		Stream:              in.Stream,
		MaxCompletionTokens: in.MaxTokens,
		Metadata:            in.MetaData,
		Temperature:         in.Temperature,
		TopP:                in.TopP,
		Messages:            make([]json.RawMessage, 0),
		StreamOptions:       &oaimodel.ChatStreamOptions{IncludeUsage: true},
	}
	// serviceTier,outputConfig，thinking,Metadata-cache_key
	// 系统提示词
	// tool_choice,tools,
	if in.ServiceTier != nil && *in.ServiceTier == "auto" {
		out.ServiceTier = "auto"
	}
	// outputConfig -> responseFormat,verbosity
	// thinking
	if in.OutputConfig != nil {
		mp := map[string]string{
			"low":    "low",
			"medium": "medium",
			"high":   "high",
			"max":    "high",
		}
		out.ReasoningEffort = in.OutputConfig.Effort
		out.Verbosity = mp[in.OutputConfig.Effort]
		if in.OutputConfig.Format != nil {
			out.ResponseFormat = &oaimodel.ChatResponseFormat{
				Type:       "json_schema",
				JsonSchema: &oaimodel.JSONSchema{Name: "trans", Schema: in.OutputConfig.Format.Schema},
			}
		}
	}
	// Metadata-cache_key
	if gjson.GetBytes(in.MetaData, "user_id").String() != "" {
		out.PromptCacheKey = gjson.GetBytes(in.MetaData, "user_id").String()
		out.SafetyIdentifier = gjson.GetBytes(in.MetaData, "user_id").String()
	}

	// 系统提示词
	// messages
	msgS := helpers.ClaudeMessageToOpenAI(in.System, in.Messages)
	out.Messages = msgS

	// tools
	// tool_choice
	tools := make([]oaimodel.ChatTool, 0, len(in.Tools))
	for _, t := range in.Tools {
		tool := helpers.ClaudeToolToOpenAITool(t)
		if tool != nil {
			tools = append(tools, *tool)
		}
	}
	out.Tools = tools
	toolChoice, disableParallelToolUse := helpers.ClaudeToolChoiceToOpenAIChoice(in.ToolChoice)
	out.ToolChoice = toolChoice
	if disableParallelToolUse != nil {
		out.ParallelToolCalls = new(!(*disableParallelToolUse))
	}
	outReq.Payload = mustMarshal(out)

	plog.Info("req.payload:", string(req.Payload))
	plog.Info("out.payload:", string(outReq.Payload))

	return outReq, nil
}

func mustMarshal(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
