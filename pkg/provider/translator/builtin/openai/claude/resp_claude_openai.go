package claude

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

func ConvertClaudeResponseToChatCompletion(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	if resp.IsStream {
		return ConvertClaudeStreamToChatCompletion(ctx, req, resp)
	}
	var in claudemodel.MessageResponse
	if err := common.UnmarshalJSON(resp.RawData, &in); err != nil {
		return nil, err
	}
	out := &oaimodel.ChatCompletion{
		Id:      firstNonEmpty(in.ID, "chat-cmpl_"+time.Now().Format("150405")),
		Created: time.Now().Unix(),
		Model:   req.Model,
		Object:  "chat.completion",
		Choices: nil,
		//ServiceTier:       "",
		//Usage:             nil,
	}
	out.Usage = helpers.ClaudeUsageToChatUsage(in.Usage)
	out.Choices = claudeContentToChatMessage(in.Content)
	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	cp := common.CloneResponse(resp)
	cp.RawData = data
	cp.Format = types.FormatChatCompletion
	return cp, nil
}

func getType(in json.RawMessage) string {
	return gjson.GetBytes(in, "type").String()
}

func claudeContentToChatMessage(blocks []json.RawMessage) []oaimodel.ChatCompletionChoice {
	result := make([]oaimodel.ChatCompletionChoice, 0, len(blocks))
	index := 0
	for _, b := range blocks {
		choice := oaimodel.ChatCompletionChoice{
			FinishReason: "",
			Index:        int64(index),
			Logprobs:     nil,
			Message: oaimodel.ChatCompletionChoiceMessage{
				Role: "assistant",
				//Content:          string(b),
				ToolCalls:        nil,
				ReasoningContent: "",
			},
		}
		switch getType(b) {
		case "text":
			choice.Message.Content = gjson.GetBytes(b, "text").String()
		case "thinking":
			choice.Message.ReasoningContent = gjson.GetBytes(b, "thinking").String()
		case "tool_use":
			data := helpers.ClaudeToolUseToChatMessageToolCall(b)
			if data != nil {
				choice.Message.ToolCalls = append(choice.Message.ToolCalls, *data)
			}
		}
	}
	return result
}

func claudeUsageToChatUsage(in *claudemodel.Usage) *oaimodel.ChatCompletionUsage {
	if in == nil {
		return nil
	}
	return &oaimodel.ChatCompletionUsage{PromptTokens: in.InputTokens, CompletionTokens: in.OutputTokens, TotalTokens: in.InputTokens + in.OutputTokens}
}
func claudeStopReasonToChatFinishReason(in string) string {
	switch in {
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	default:
		return "stop"
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
