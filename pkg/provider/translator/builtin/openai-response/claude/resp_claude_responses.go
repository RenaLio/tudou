package claude

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

func ConvertClaudeResponseToOpenAIResponses(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	if resp.IsStream {
		return ConvertClaudeStreamToOpenAIResponses(ctx, req, resp)
	}

	var in claudemodel.MessageResponse
	if err := common.UnmarshalJSON(resp.RawData, &in); err != nil {
		return nil, err
	}

	var srcReq oaimodel.CreateResponseRequest
	_ = common.UnmarshalJSON(req.Payload, &srcReq)

	out := oaimodel.Response{
		Id:                   firstNonEmpty(in.ID, "resp_"+RandomId()),
		CreatedAt:            time.Now().Unix(),
		Instructions:         json.RawMessage(srcReq.Instructions),
		Metadata:             srcReq.Metadata,
		Model:                srcReq.Model,
		Object:               "response",
		ParallelToolCalls:    srcReq.ParallelToolCalls,
		Temperature:          srcReq.Temperature,
		ToolChoice:           srcReq.ToolChoice,
		Tools:                srcReq.Tools,
		TopP:                 srcReq.TopP,
		Background:           srcReq.Background,
		CompletedAt:          time.Now().Unix(),
		MaxOutputTokens:      srcReq.MaxOutputTokens,
		MaxToolCalls:         srcReq.MaxToolCalls,
		PreviousResponseID:   srcReq.PreviousResponseID,
		Prompt:               srcReq.Prompt,
		PromptCacheKey:       srcReq.PromptCacheKey,
		PromptCacheRetention: srcReq.PromptCacheRetention,
		Reasoning:            srcReq.Reasoning,
		SafetyIdentifier:     srcReq.SafetyIdentifier,
		ServiceTier:          srcReq.ServiceTier,
		Text:                 srcReq.Text,
		TopLogprobs:          srcReq.TopLogprobs,
		Truncation:           srcReq.Truncation,

		Error:             nil,
		IncompleteDetails: nil,

		Status: "completed",
		Output: nil,
		Usage:  helpers.ClaudeUsageToResponseUsage(in.Usage),
	}

	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	cp := common.CloneResponse(resp)

	cp.RawData = data
	cp.Format = types.FormatOpenAIResponses

	return cp, nil
}

func getType(b json.RawMessage) string {
	return gjson.GetBytes(b, "type").String()
}

func claudeContentBlocksToResponseOutput(blocks []json.RawMessage) []json.RawMessage {
	var out []json.RawMessage

	for _, block := range blocks {
		switch getType(block) {
		case "text":
			// claudemodel.TextBlock
			outputItem := oaimodel.ResponseOutputMessage{
				Id:      "",
				Type:    "message",
				Content: mustMarshal([]oaimodel.ResponseOutputText{{Type: "output_text", Text: gjson.GetBytes(block, "text").String()}}),
				Role:    "assistant",
				Status:  "completed",
			}
			out = append(out, mustMarshal(outputItem))
		case "thinking":
			//claudemodel.ThinkingBlock{}
			outputItem := oaimodel.ResponseReasoningItem{
				Id:      "",
				Type:    "reasoning",
				Summary: []oaimodel.TypeTextObject{{Type: "summary_text", Text: gjson.GetBytes(block, "thinking").String()}},
				Status:  "completed",
			}
			out = append(out, mustMarshal(outputItem))
		case "tool_use":
			//claudemodel.ToolUseBlock{}
			item := helpers.ClaudeToolUseToResponseToolCall(block)
			out = append(out, item)
		}
	}

	return out
}

func RandomId() string {
	return RandomString(48)
}

func RandomString(length int) string {
	//const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}
