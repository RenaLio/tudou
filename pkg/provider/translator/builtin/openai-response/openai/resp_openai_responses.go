package openai

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func ConvertOpenAIResponseToResponses(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	if !resp.IsStream {
		data, err := ConvertOpenAINonStreamResponseToResponses(ctx, req, resp)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	return ConvertChatCompletionStreamToResponse(ctx, req, resp), nil
}

func ConvertOpenAINonStreamResponseToResponses(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	completion := new(openai.ChatCompletion)
	err := common.UnmarshalJSON(resp.RawData, completion)
	if err != nil {
		return nil, err
	}
	params := new(openai.CreateResponseRequest)
	err = common.UnmarshalJSON(req.Payload, params)
	if err != nil {
		return nil, err
	}

	response := &openai.Response{
		Id:          "resp_" + RandomStringId(48),
		CompletedAt: completion.Created,
		Model:       completion.Model,
		Object:      "response",
		ServiceTier: completion.ServiceTier,
		//Status:      "completed",
	}
	var inCompleteDetailReason string // "max_output_tokens" "content_filter"
	// usage -> usage
	response.Usage = helpers.ChatUsageToResponseUsage(completion.Usage)
	//if completion.Usage != nil {
	//	response.Usage = ChatUsageToResponseUsage(completion.Usage)
	//}
	// choices -> outputs
	outputs, err := ChatCompletionChoicesToResponseOutput(completion)
	if err != nil {
		return nil, err
	}
	response.Output = outputs
	// other
	response.CompletedAt = time.Now().Unix()
	if inCompleteDetailReason != "" {

		response.IncompleteDetails = &openai.IncompleteDetails{
			Reason: inCompleteDetailReason,
		}
	}
	response.Metadata = params.Metadata
	response.ParallelToolCalls = params.ParallelToolCalls
	response.Temperature = params.Temperature
	response.ToolChoice = params.ToolChoice
	response.Tools = params.Tools
	response.TopP = params.TopP
	response.Background = params.Background
	response.MaxOutputTokens = params.MaxOutputTokens
	response.MaxToolCalls = params.MaxToolCalls
	response.PreviousResponseID = params.PreviousResponseID
	response.Prompt = params.Prompt
	response.PromptCacheKey = params.PromptCacheKey
	response.PromptCacheRetention = params.PromptCacheRetention
	response.Reasoning = params.Reasoning
	response.SafetyIdentifier = params.SafetyIdentifier
	response.ServiceTier = params.ServiceTier
	response.Text = params.Text
	response.TopLogprobs = params.TopLogprobs
	response.Truncation = params.Truncation

	data, err := common.MarshalJSON(response)
	if err != nil {
		return nil, err
	}
	resp.RawData = data
	resp.Format = req.Format
	return resp, nil
}

func ChatUsageToResponseUsage(usage *openai.ChatCompletionUsage) *openai.ResponseUsage {
	if usage == nil {
		return nil
	}

	responseUsage := &openai.ResponseUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}

	if usage.PromptTokensDetails != nil {
		responseUsage.InputTokenDetails = &openai.InputTokenDetails{
			CachedTokens: usage.PromptTokensDetails.CachedTokens,
		}
	}
	if usage.CompletionTokensDetails != nil {
		responseUsage.OutputTokenDetails = &openai.OutputTokenDetails{
			ReasoningTokens: usage.CompletionTokensDetails.ReasoningTokens,
		}
	}

	return responseUsage
}

// responses.ResponseOutputItem
// supported finish_reason: "stop" "length" "content_filter" "tool_calls" "function_call"
// supported output:
// - message
// - function_tool_call
// - custom_tool_call
// - reasoning

func ChatCompletionChoicesToResponseOutput(completion *openai.ChatCompletion) ([]json.RawMessage, error) {
	var outputs []json.RawMessage
	for _, choice := range completion.Choices {
		if choice.FinishReason == "function_call" {
			responseToolCall := openai.ResponseFunctionCall{
				Type:      "function_call",
				Status:    "completed",
				Name:      choice.Message.FunctionCall.Name,
				Arguments: choice.Message.FunctionCall.Arguments,
			}
			output, _ := json.Marshal(responseToolCall)
			outputs = append(outputs, output)
			continue
		}
		if choice.FinishReason == "tool_calls" {
			for _, toolCall := range choice.Message.ToolCalls {
				// 构建outputItem
				if toolCall.Type == "function" {
					responseToolCall := openai.ResponseFunctionCall{
						Type:      "function_call",
						Status:    "completed",
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
						CallID:    toolCall.Id,
					}
					output, _ := json.Marshal(responseToolCall)
					outputs = append(outputs, output)
					continue
				}
				if toolCall.Type == "custom" {
					responseToolCall := openai.ResponseCustomToolCall{
						Type:   "custom_tool_call",
						Name:   toolCall.Custom.Name,
						Input:  toolCall.Custom.Input,
						CallId: toolCall.Id,
					}
					output, _ := json.Marshal(responseToolCall)
					outputs = append(outputs, output)
					continue
				}
			}
		}
		// stop、content_filter、length
		// 处理reasoning
		if len(choice.Message.ReasoningContent) > 0 {
			outputItem := openai.ResponseReasoningItem{
				Type: "reasoning",
				Content: []openai.TypeTextObject{
					{Type: "reasoning_text", Text: choice.Message.ReasoningContent},
				},
				Status: "completed",
			}
			output, _ := json.Marshal(outputItem)
			outputs = append(outputs, output)
		}
		var outputItem openai.ResponseOutputMessage
		outputItem.Type = "message"
		outputItem.Role = "assistant"
		if choice.FinishReason == "length" || choice.FinishReason == "length_text" {
			outputItem.Status = "incomplete"
		}
		// 如果模型拒绝了响应
		if len(choice.Message.Refusal) > 0 {
			content := []openai.ResponseOutputRefusal{
				{Refusal: choice.Message.Refusal, Type: "refusal"},
			}
			contentBytes, _ := json.Marshal(content)
			outputItem.Content = contentBytes
			output, _ := json.Marshal(outputItem)
			outputs = append(outputs, output)
			continue
		}
		var annotations []json.RawMessage
		if choice.Message.Annotations != nil {
			for _, annotation := range choice.Message.Annotations {
				mp := make(map[string]any)
				mp["type"] = annotation.Type // url_citation
				if annotation.UrlCitation != nil {
					mp["start_index"] = annotation.UrlCitation.StartIndex
					mp["end_index"] = annotation.UrlCitation.EndIndex
					mp["url"] = annotation.UrlCitation.Url
				}
				b, _ := json.Marshal(mp)
				annotations = append(annotations, b)
			}

		}
		outputTexts := []openai.ResponseOutputText{
			{
				Annotations: annotations,
				Text:        choice.Message.Content,
				Type:        "output_text",
			},
		}
		contentBytes, _ := json.Marshal(outputTexts)
		outputItem.Content = contentBytes
		output, _ := json.Marshal(outputItem)
		outputs = append(outputs, output)
	}
	return outputs, nil
}
