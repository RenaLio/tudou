package openairesponses

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

// ConvertResponsesToClaudeResponse 将 OpenAI Responses 响应转换为 Claude 响应
// 这是从 openai/responses 格式到 claude/openai-responses 格式的转换
func ConvertResponsesToClaudeResponse(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	if resp.IsStream {
		return ConvertResponsesStreamToClaude(ctx, req, resp)
	}
	var in oaimodel.Response
	if err := common.UnmarshalJSON(resp.RawData, &in); err != nil {
		return nil, err
	}
	out := &claudemodel.MessageResponse{
		ID:         firstNonEmpty(in.Id, "msg_"+time.Now().Format("150405")),
		Content:    responseOutputToClaudeContent(in.Output),
		Model:      in.Model,
		Role:       "assistant",
		StopReason: responseStatusToClaudeStopReason(in.Status, in.IncompleteDetails),
		Type:       "message",
		Usage:      helpers.ResponseUsageToClaudeUsage(in.Usage),

		Container:    nil,
		StopSequence: "",
	}
	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	cp := common.CloneResponse(resp)
	cp.RawData = data
	cp.Format = types.FormatClaudeMessages
	return cp, nil
}

// responseOutputToClaudeContent 将 OpenAI Responses Output 转换为 Claude Content
func responseOutputToClaudeContent(items []json.RawMessage) []json.RawMessage {
	out := make([]json.RawMessage, 0)
	for _, raw := range items {
		switch getType(raw) {
		case "message":
			block := &claudemodel.TextBlock{
				Type:      "text",
				Citations: nil,
				Text:      "",
			}
			if gjson.GetBytes(raw, "content").IsArray() {
				gjson.GetBytes(raw, "content").ForEach(func(_, value gjson.Result) bool {
					switch value.Get("type").String() {
					case "output_text":
						block.Text = value.Get("text").String()
						out = append(out, mustMarshal(block))
					case "refusal":
					}
					return true
				})
			}
		case "function_call":
			blockJson := helpers.ResponseToolCallToClaudeToolUse(raw)
			if blockJson != nil {
				out = append(out, blockJson)
			}

		case "function_call_output", "custom_tool_call_output":
			blockJson := helpers.ResponseToolCallOutputToClaude(raw)
			if blockJson != nil {
				out = append(out, blockJson)
			}

		case "reasoning":
			var rr oaimodel.ResponseReasoningItem
			if err := json.Unmarshal(raw, &rr); err == nil {
				for _, c := range rr.Summary {
					if c.Text != "" {
						out = append(out, mustMarshal(claudemodel.ThinkingBlock{
							Type:     "thinking",
							Thinking: c.Text,
						}))
					}
				}
			}
		}
	}
	return out
}

// responseStatusToClaudeStopReason 将 OpenAI Responses Status 转换为 Claude StopReason
func responseStatusToClaudeStopReason(status string, incomplete *oaimodel.IncompleteDetails) string {
	if incomplete != nil && incomplete.Reason == "max_output_tokens" {
		return "max_tokens"
	}
	if status == "incomplete" {
		return "max_tokens"
	}
	return "end_turn"
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
