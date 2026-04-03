package openairesponses

import (
	"context"
	"encoding/json"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

// ConvertClaudeRequestToResponses 将 Claude 请求转换为 OpenAI Responses 请求
// 这是从 claude/openai-responses 格式到 openai/responses 格式的转换
func ConvertClaudeRequestToResponses(_ context.Context, req *types.Request) (*types.Request, error) {
	outReq := common.CloneRequest(req)
	var in claudemodel.CreateMessageRequest
	if err := common.UnmarshalJSON(req.Payload, &in); err != nil {
		return nil, err
	}

	// 创建 OpenAI Responses 请求
	out := &oaimodel.CreateResponseRequest{
		Conversation: nil,
		Include:      nil,
		//Include:         []string{"reasoning.encrypted_content"},
		MaxOutputTokens: in.MaxTokens,
		//Metadata:        in.MetaData,
		Model:         in.Model,
		Stream:        in.Stream,
		Temperature:   in.Temperature,
		TopP:          in.TopP,
		StreamOptions: nil,
	}
	// serviceTier
	if in.ServiceTier != nil && *in.ServiceTier != "" {
		out.ServiceTier = *in.ServiceTier
	}
	// cacheKey
	if gjson.GetBytes(in.MetaData, "user_id").Exists() {
		out.PromptCacheKey = gjson.GetBytes(in.MetaData, "user_id").String()[:64]
		out.SafetyIdentifier = gjson.GetBytes(in.MetaData, "user_id").String()[:64]
	}
	// Text Reasoning
	// 处理 OutputConfig -> Reasoning, Text
	if in.OutputConfig != nil {
		out.Reasoning = &oaimodel.Reasoning{Effort: in.OutputConfig.Effort}
		if in.OutputConfig.Format != nil && in.OutputConfig.Format.Type == "json_schema" {
			out.Text = &oaimodel.ResponseTextConfig{
				Format: &oaimodel.ResponseTextConfigFormat{
					Type:       "json_schema",
					JSONSchema: oaimodel.JSONSchema{Schema: in.OutputConfig.Format.Schema},
				},
			}
		}
	}
	// 消息和系统消息
	system, messages := ClaudeMessageToResponse(in.System, in.Messages)
	out.Instructions = system
	if len(messages) > 0 {
		out.Input = mustMarshal(messages)
	}
	// Tools
	for _, tool := range in.Tools {
		data := helpers.ClaudeToolToResponsesTool(tool)
		if data != nil {
			out.Tools = append(out.Tools, mustMarshal(data))
		}
	}
	// ToolChoice
	toolChoice, disableParallelToolUse := helpers.ClaudeToolChoiceToOpenAIChoice(in.ToolChoice)
	if disableParallelToolUse != nil {
		out.ParallelToolCalls = new(!(*disableParallelToolUse))
	}
	out.ToolChoice = toolChoice

	// 序列化输出
	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}

	plog.Info("req.payload:", string(req.Payload))
	plog.Info("out.payload:", string(data))

	outReq.Payload = data
	return outReq, nil
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ClaudeMessageToResponse 将 Claude 消息转换为 OpenAI Responses 消息
//
// parameters:
//   - system: string | []textTypeObj
//   - messages: [] claudemodel.MessageParam
//
// return:
//   - json.RawMessage: system -> Instructions (string)
//   - []json.RawMessage: messages -> Input
func ClaudeMessageToResponse(system json.RawMessage, messages []claudemodel.MessageParam) (string, []json.RawMessage) {
	var systemMsg string
	inputs := make([]json.RawMessage, 0, len(messages))
	if gjson.ParseBytes(system).Type == gjson.String {
		systemMsg = gjson.ParseBytes(system).String()
	} else {
		// []typeTextObj
		gjson.ParseBytes(system).ForEach(func(key, value gjson.Result) bool {
			systemMsg += gjson.Get(value.Raw, "text").String() + "\n\n"
			return true
		})
		//developerMsg := oaimodel.Message{
		//	Type:    "message",
		//	Content: system,
		//	Role:    "developer",
		//}
		//inputs = append(inputs, mustMarshal(developerMsg))
	}
	// claude messages to openai responses input
	inputs = append(inputs, claudeMessagesToResponseInput(messages)...)
	return systemMsg, inputs
}

// claudeMessagesToResponseInput 将 Claude 消息转换为 OpenAI Responses Input
func claudeMessagesToResponseInput(msgs []claudemodel.MessageParam) []json.RawMessage {

	items := make([]json.RawMessage, 0, len(msgs))
	for _, msg := range msgs {
		textType := "input_text"
		if msg.Role == "assistant" {
			textType = "output_text"
		}
		// 解析 content 块
		if gjson.ParseBytes(msg.Content).Type == gjson.String {
			item := oaimodel.Message{
				Type: "message",
				//Content: msg.Content,
				Content: mustMarshal([]oaimodel.TypeTextObject{{Type: textType, Text: gjson.ParseBytes(msg.Content).String()}}),
				Role:    msg.Role,
			}
			items = append(items, mustMarshal(item))
			continue
		}
		var blocks []json.RawMessage

		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}

		// 合并消息
		type ContentItem struct {
			Type    string
			Content json.RawMessage
		}

		for _, block := range blocks {
			switch getType(block) {
			case "text":
				//result, _ := sjson.SetBytes(block, "type", textType)
				item := oaimodel.Message{
					Type:    "message",
					Content: mustMarshal([]oaimodel.TypeTextObject{{Type: textType, Text: gjson.GetBytes(block, "text").String()}}),
					Role:    msg.Role,
				}
				items = append(items, mustMarshal(item))
			case "image":
				imageUrl := gjson.ParseBytes(block).Get("source.url").String()
				if imageUrl == "" {
					imageUrl = gjson.GetBytes(block, "source.data").String()
				}
				ResponseInputImage := oaimodel.ResponseInputImage{
					Type:     "image_url",
					Detail:   "high",
					ImageUrl: imageUrl,
				}
				item := oaimodel.Message{
					Type:    "message",
					Content: mustMarshal([]json.RawMessage{mustMarshal(ResponseInputImage)}),
					Role:    msg.Role,
				}
				items = append(items, mustMarshal(item))
			case "thinking":
				item := oaimodel.ResponseReasoningItem{
					//Id:      "rs_" + time.Now().Format("150405"),
					Type:    "reasoning",
					Summary: []oaimodel.TypeTextObject{{Type: "summary_text", Text: gjson.GetBytes(block, "thinking").String()}},
				}
				items = append(items, mustMarshal(item))
			case "tool_use":
				item := helpers.ClaudeToolUseToResponseToolCall(block)
				items = append(items, mustMarshal(item))
			case "tool_result":
				item := helpers.ClaudeToolCallOutputToResponse(block, "")
				items = append(items, mustMarshal(item))
			}
		}
	}
	return items
}

func getType(in json.RawMessage) string {
	return gjson.ParseBytes(in).Get("type").String()
}
