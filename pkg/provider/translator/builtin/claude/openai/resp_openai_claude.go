package openai

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func ConvertChatCompletionResponseToClaudeResponse(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	if resp.IsStream {
		return ConvertChatCompletionStreamToClaude(ctx, req, resp)
	}
	var in oaimodel.ChatCompletion
	if err := common.UnmarshalJSON(resp.RawData, &in); err != nil {
		return nil, err
	}
	out := &claudemodel.MessageResponse{
		ID: firstNonEmpty(in.Id, "msg_"+strconv.FormatInt(time.Now().Unix(), 10)),
		//Content:    []json.RawMessage{},
		Model:      in.Model,
		Role:       "assistant",
		StopReason: "end_turn",
		Type:       "message",
		Usage:      helpers.ChatUsageToClaudeUsage(in.Usage),
	}
	if len(in.Choices) > 0 {
		// stop reason
		out.StopReason = chatFinishReasonToClaudeStopReason(in.Choices[0].FinishReason)
		// content
		out.Content = ChatChoicesToClaudeContent(in.Choices)
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

func ChatChoicesToClaudeContent(choices []oaimodel.ChatCompletionChoice) []json.RawMessage {
	var out []json.RawMessage
	for _, choice := range choices {
		// tool_calls
		if len(choice.Message.ToolCalls) > 0 {
			for _, tc := range choice.Message.ToolCalls {
				data := helpers.ChatMessageToolCallToClaudeToolUse(&tc)
				if data != nil {
					out = append(out, data)
				}
			}
		}
		// reasoning
		if choice.Message.ReasoningContent != "" {
			block := new(claudemodel.ThinkingBlock)
			block.Type = "thinking"
			block.Thinking = choice.Message.ReasoningContent
			out = append(out, mustMarshal(block))
		}
		// text
		if choice.Message.Content != "" {
			block := new(claudemodel.TextBlock)
			block.Type = "text"
			block.Text = choice.Message.Content
			out = append(out, mustMarshal(block))
		}
		// refusal
		// function_call: Deprecated
	}
	return out
}

func chatFinishReasonToClaudeStopReason(in string) string {
	switch in {
	case "tool_calls", "function_call":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "refusal"
	default:
		return "end_turn"
	}
}
func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
