package claude

import (
	"context"

	"strings"

	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// ConvertResponsesRequestToClaude 将 OpenAI Responses 请求转换为 Claude 请求
func ConvertResponsesRequestToClaude(_ context.Context, req *types.Request) (*types.Request, error) {
	outReq := common.CloneRequest(req)

	var in oaimodel.CreateResponseRequest
	if err := common.UnmarshalJSON(req.Payload, &in); err != nil {
		return nil, err
	}

	out := &claudemodel.CreateMessageRequest{
		MaxTokens: in.MaxOutputTokens,
		Messages:  make([]claudemodel.MessageParam, 0),
		Model:     in.Model,
		CacheControl: &claudemodel.CacheControlEphemeral{
			Type: "ephemeral",
		},
		//MetaData:      nil,
		//OutputConfig:  nil,
		//ServiceTier:   nil,
		Stream: in.Stream,
		//System:      nil,
		Temperature: in.Temperature,
		//Thinking:    nil,
		//ToolChoice:  nil,
		//Tools:       nil,
		TopP: in.TopP,
	}
	// cache - metadata
	if in.PromptCacheKey != "" {
		out.MetaData = mustMarshal(map[string]any{"user_id": string(mustMarshal(map[string]string{"prompt_cache_key": in.PromptCacheKey, "safety_identifier": in.SafetyIdentifier}))})
	}
	// outputConfig thinking
	// 处理 Text/Reasoning -> OutputConfig
	if in.Text != nil {
		out.OutputConfig = &claudemodel.OutputConfig{}
		if in.Reasoning != nil {
			out.OutputConfig.Effort = in.Reasoning.Effort
		}
		if in.Text.Format != nil && in.Text.Format.Type == "json_schema" {
			out.OutputConfig.Format = &claudemodel.JSONOutputFormat{
				Type:   "json_schema",
				Schema: in.Text.Format.Schema,
			}
		}
	}
	if out.OutputConfig == nil && in.Reasoning != nil && in.Reasoning.Effort != "" {
		out.OutputConfig = &claudemodel.OutputConfig{Effort: in.Reasoning.Effort}
	}

	// serviceTier
	if in.ServiceTier != "" {
		out.ServiceTier = &in.ServiceTier
	}
	// system，messages
	// 处理 Input -> Messages 和 System
	messages, system := responsesInputToClaudeMessages(in.Instructions, in.Input)
	out.Messages = append(out.Messages, messages...)
	out.System = system
	// tools
	// 处理 Tools
	for _, tool := range in.Tools {
		item := helpers.ResponsesToolToClaudeTool(tool)
		if item != nil {
			out.Tools = append(out.Tools, item)
		}
	}

	// toolChoice
	// 处理 ToolChoice
	out.ToolChoice = helpers.ResponsesChoiceToClaudeToolChoice(in.ToolChoice, new(true))

	if out.MaxTokens == nil {
		out.MaxTokens = new(int64)
		*out.MaxTokens = 1024 * 8
	}

	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}

	plog.Debug("req.payload:", string(req.Payload))
	plog.Debug("out.payload:", string(data))

	outReq.Payload = data
	return outReq, nil
}

func responsesInputToClaudeMessages(systemStr string, respInput json.RawMessage) ([]claudemodel.MessageParam, json.RawMessage) {
	systemBlocks := make([]claudemodel.TextBlockParam, 0)
	if systemStr != "" {
		systemBlocks = append(systemBlocks, claudemodel.TextBlockParam{Type: "text", Text: systemStr})
	}
	messages := make([]claudemodel.MessageParam, 0)
	if gjson.ParseBytes(respInput).Type == gjson.String {
		messages = append(messages, claudemodel.MessageParam{
			Role:    "user",
			Content: respInput,
		})
		return messages, mustMarshal(systemBlocks)
	}
	if gjson.ParseBytes(respInput).IsArray() {
		gjson.ParseBytes(respInput).ForEach(func(_, value gjson.Result) bool {
			switch gjson.Get(value.Raw, "type").String() {
			case "message", "":
				var messageItem oaimodel.Message
				if err := json.Unmarshal([]byte(value.Raw), &messageItem); err != nil {
					return true
				}
				if gjson.ParseBytes(messageItem.Content).Type == gjson.String {
					if messageItem.Role == "system" || messageItem.Role == "developer" {
						systemBlocks = append(systemBlocks, claudemodel.TextBlockParam{Type: "text", Text: gjson.ParseBytes(messageItem.Content).String()})
					} else {
						messages = append(messages, claudemodel.MessageParam{
							Role:    messageItem.Role,
							Content: messageItem.Content,
						})
					}
				}
				if gjson.ParseBytes(messageItem.Content).IsArray() {
					gjson.ParseBytes(messageItem.Content).ForEach(func(_, content gjson.Result) bool {
						switch gjson.Get(content.Raw, "type").String() {
						case "input_text":
							text := content.Get("text").String()
							if messageItem.Role == "system" || messageItem.Role == "developer" {
								systemBlocks = append(systemBlocks, claudemodel.TextBlockParam{Type: "text", Text: text})
							} else {
								messages = append(messages, claudemodel.MessageParam{
									Role:    messageItem.Role,
									Content: mustMarshal([]claudemodel.TextBlockParam{{Type: "text", Text: text}}),
								})
							}
						case "input_image":
							// Convert image_url to Claude ImageBlockParam
							url := content.Get("image_url").String()
							imageBlockParam := claudemodel.ImageBlockParam{
								Type: "image_url",
								Source: claudemodel.ImageSourceParam{
									Type: "url",
									Url:  url,
								},
							}
							messages = append(messages, claudemodel.MessageParam{
								Role:    messageItem.Role,
								Content: mustMarshal([]claudemodel.ImageBlockParam{imageBlockParam}),
							})
						}
						return true
					})
				}
			case "function_call":
				block := helpers.ResponseToolCallToClaudeToolUse(json.RawMessage(value.Raw))
				message := claudemodel.MessageParam{
					Role:    "assistant",
					Content: mustMarshal([]json.RawMessage{block}),
				}
				messages = append(messages, message)
			case "function_call_output":
				blockParam := helpers.ResponseToolCallOutputToClaude(json.RawMessage(value.Raw))
				message := claudemodel.MessageParam{
					Role:    "user",
					Content: mustMarshal([]json.RawMessage{blockParam}),
				}
				messages = append(messages, message)
			case "reasoning":
				var reasoningItem oaimodel.ResponseReasoningItem
				if err := json.Unmarshal([]byte(value.Raw), &reasoningItem); err != nil {
					return true
				}
				var summary string
				for _, s := range reasoningItem.Summary {
					summary += s.Text + "\n"
				}
				thinkingBlockParam := claudemodel.ThinkingBlockParam{
					Type:      "thinking",
					Thinking:  summary,
					Signature: "",
				}
				message := claudemodel.MessageParam{
					Role:    "assistant",
					Content: mustMarshal([]claudemodel.ThinkingBlockParam{thinkingBlockParam}),
				}
				messages = append(messages, message)
			}
			return true
		})
	}
	return messages, mustMarshal(systemBlocks)
}

// responseInputToClaudeMessages 将 OpenAI Responses Input 转换为 Claude Messages
func responseInputToClaudeMessages(raw json.RawMessage) ([]claudemodel.MessageParam, json.RawMessage) {
	if len(raw) == 0 {
		return nil, nil
	}

	// 尝试作为字符串解析
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []claudemodel.MessageParam{
			{Role: "user", Content: mustMarshal([]claudemodel.TextBlockParam{{Type: "text", Text: s}})},
		}, nil
	}

	// 作为数组解析
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, nil
	}

	messages := make([]claudemodel.MessageParam, 0, len(items))
	var systemParts []string

	for _, item := range items {
		var base struct {
			Type string `json:"type"`
			Role string `json:"role"`
		}
		_ = json.Unmarshal(item, &base)

		typ := base.Type
		if typ == "" {
			typ = "message"
		}

		switch typ {
		case "message":
			var msg oaimodel.Message
			if err := json.Unmarshal(item, &msg); err != nil {
				continue
			}
			role := normalizeClaudeRole(msg.Role)
			text := openAIMessageContentToText(msg.Content)
			if role == "system" {
				if text != "" {
					systemParts = append(systemParts, text)
				}
				continue
			}
			messages = append(messages, claudemodel.MessageParam{
				Role:    role,
				Content: mustMarshal([]claudemodel.TextBlockParam{{Type: "text", Text: text}}),
			})

		case "function_call":
			var call oaimodel.ResponseFunctionCall
			if err := json.Unmarshal(item, &call); err != nil {
				continue
			}
			var inputMap map[string]json.RawMessage
			if err := json.Unmarshal([]byte(call.Arguments), &inputMap); err != nil {
				inputMap = make(map[string]json.RawMessage)
			}
			messages = append(messages, claudemodel.MessageParam{
				Role: "assistant",
				Content: mustMarshal([]claudemodel.ToolUseBlockParam{{
					Type:  "tool_use",
					ID:    firstNonEmpty(call.CallID, call.Id),
					Name:  call.Name,
					Input: inputMap,
				}}),
			})

		case "function_call_output":
			var out oaimodel.FunctionCallOutput
			if err := json.Unmarshal(item, &out); err != nil {
				continue
			}
			messages = append(messages, claudemodel.MessageParam{
				Role: "user",
				Content: mustMarshal([]claudemodel.ToolResultBlockParam{{
					Type:      "tool_result",
					ToolUseId: out.CallId,
					Content:   out.Output,
					IsError:   false,
				}}),
			})
		}
	}

	var system json.RawMessage
	if len(systemParts) > 0 {
		system = mustMarshal(strings.Join(systemParts, "\n\n"))
	}
	return messages, system
}

// normalizeClaudeRole 规范化 Claude 角色
func normalizeClaudeRole(role string) string {
	switch role {
	case "system", "developer":
		return "system"
	case "assistant":
		return "assistant"
	default:
		return "user"
	}
}

// openAIMessageContentToText 从 OpenAI 消息内容中提取文本
func openAIMessageContentToText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if p["type"] == "text" || p["type"] == "input_text" || p["type"] == "output_text" {
				if v, ok := p["text"].(string); ok && v != "" {
					texts = append(texts, v)
				}
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		plog.Error("mustMarshal", "err", err)
	}
	return b
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
