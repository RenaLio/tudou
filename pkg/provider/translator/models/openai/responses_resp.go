package openai

import "encoding/json"

type Response struct {
	Id                   string                `json:"id"`
	CreatedAt            int64                 `json:"created_at"`
	Error                *ResponseError        `json:"error,omitempty"`
	IncompleteDetails    *IncompleteDetails    `json:"incomplete_details,omitempty"`
	Instructions         json.RawMessage       `json:"instructions,omitempty"` // string or [] ResponseInputItem
	Metadata             json.RawMessage       `json:"metadata,omitempty"`
	Model                string                `json:"model,omitempty"`
	Object               string                `json:"object"`
	Output               []json.RawMessage     `json:"output,omitempty"` //  ResponseOutputItem
	ParallelToolCalls    *bool                 `json:"parallel_tool_calls,omitempty"`
	Temperature          *float64              `json:"temperature,omitempty"`
	ToolChoice           json.RawMessage       `json:"tool_choice,omitempty"` // string or ResponseToolChoice
	Tools                []json.RawMessage     `json:"tools,omitempty"`       // Tool
	TopP                 *float64              `json:"top_p,omitempty"`
	Background           *bool                 `json:"background,omitempty"`
	CompletedAt          int64                 `json:"completed_at,omitempty"`
	Conversation         *ResponseConversation `json:"conversation,omitempty"`
	MaxOutputTokens      *int64                `json:"max_output_tokens,omitempty"`
	MaxToolCalls         *int64                `json:"max_tool_calls,omitempty"`
	OutputText           string                `json:"output_text,omitempty"`
	PreviousResponseID   string                `json:"previous_response_id,omitempty"`
	Prompt               *ResponsePrompt       `json:"prompt,omitempty"`
	PromptCacheKey       string                `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string                `json:"prompt_cache_retention,omitempty"`
	Reasoning            *Reasoning            `json:"reasoning,omitempty"`
	SafetyIdentifier     string                `json:"safety_identifier,omitempty"`
	ServiceTier          string                `json:"service_tier,omitempty"`
	Status               string                `json:"status,omitempty"`
	Text                 *ResponseTextConfig   `json:"text,omitempty"`
	TopLogprobs          *int64                `json:"top_logprobs,omitempty"`
	Truncation           string                `json:"truncation,omitempty"`
	Usage                *ResponseUsage        `json:"usage,omitempty"`
	// Deprecated: This field is being replaced by safety_identifier and prompt_cache_key
	User string `json:"user,omitempty"`
}

type (
	ResponseError struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	IncompleteDetails struct {
		Reason string `json:"reason,omitempty"`
	}
	ResponseConversation struct {
		ID string `json:"id"`
	}
	ResponseUsage struct {
		InputTokens        int64               `json:"input_tokens,omitempty"`
		OutputTokens       int64               `json:"output_tokens,omitempty"`
		InputTokenDetails  *InputTokenDetails  `json:"input_token_details,omitempty"`
		OutputTokenDetails *OutputTokenDetails `json:"output_token_details,omitempty"`
		TotalTokens        int64               `json:"total_tokens,omitempty"`
	}
	InputTokenDetails struct {
		CachedTokens int64 `json:"cached_tokens"`
	}
	OutputTokenDetails struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	}
)

type (
	ResponseOutputText struct {
		Annotations []json.RawMessage `json:"annotations,omitempty"`
		LogProbs    json.RawMessage   `json:"log_probs,omitempty"`
		Text        string            `json:"text"`
		Type        string            `json:"type"` // "output_text"
	}
	ResponseOutputRefusal struct {
		Refusal string `json:"refusal"`
		Type    string `json:"type"` // "refusal"
	}
)

type (
	ResponseOutputItem     = ResponseInputItem // 几乎是差不多的
	ResponseCompactionItem struct {
		ResponseCompactionItemParam
		CreatedBy string `json:"created_by,omitempty"`
	}
	ResponseFunctionShellToolCall struct {
		ShellCall
		CreatedBy string `json:"created_by,omitempty"`
	}
	ResponseFunctionShellToolCallOutput struct {
		ShellCallOutput
		CreatedBy string `json:"created_by,omitempty"`
	}
	ResponseApplyPatchToolCall struct {
		ApplyPatchCall
		CreatedBy string `json:"created_by,omitempty"`
	}
	ResponseApplyPatchToolCallOutput struct {
		ApplyPatchCallOutput
		CreatedBy string `json:"created_by,omitempty"`
	}
)
