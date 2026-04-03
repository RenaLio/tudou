package openai

import "encoding/json"

// from https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
// use context7 :OpenAI API Reference
// use context7 ：OpenAI API
// Notes (Context7):
// - `model` + `messages` are required.
// - `tool_choice` controls whether/which tool is called (`none`/`auto`/`required` or named tool).
// - `stream_options.include_usage` controls whether final streaming chunk includes usage.
// - `prompt_cache_key` + `prompt_cache_retention` controls is the cache key for prompt caching.

// CreateChatCompletionRequest is the request body for POST /v1/chat/completions.
// Field intent follows OpenAI official API reference semantics.
type CreateChatCompletionRequest struct {
	// Messages A list of messages comprising the conversation so far.
	// supported text,images,audio input.
	// Each message must be a valid JSON object.
	Messages []json.RawMessage `json:"messages"`
	// Model is the model ID to run.
	Model string `json:"model"`
	// Audio configures audio output behavior when audio modality is used.
	Audio *ChatAudioParam `json:"audio,omitempty"`
	// FrequencyPenalty reduces repetition by penalizing frequent tokens.
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	// Deprecated: Use tool_choice instead.
	// FunctionCall is the legacy function-call control field.
	FunctionCall json.RawMessage `json:"function_call,omitempty"` // Enum: "none" | "auto" | NameObject
	// Deprecated: Use tools instead.
	// Functions is the legacy function definition list.
	Functions []Function `json:"functions,omitempty"`
	// LogitBias applies token-level sampling bias by token id.
	LogitBias json.RawMessage `json:"logit_bias,omitempty"`
	// Logprobs controls whether token log probabilities are returned.
	Logprobs *bool `json:"logprobs,omitempty"`
	// MaxCompletionTokens sets the output token upper bound (recommended field).
	MaxCompletionTokens *int64 `json:"max_completion_tokens,omitempty"`
	// Deprecated: Use max_completion_tokens instead.
	// MaxTokens is the legacy output token upper bound.
	MaxTokens *int64 `json:"max_tokens,omitempty"`
	// Metadata carries caller-defined metadata for this request.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Modalities selects output modalities (for example text/audio).
	Modalities []string `json:"modalities,omitempty"`
	// N sets how many completion choices to generate.
	N *int64 `json:"n,omitempty"`
	// ParallelToolCalls allows the model to issue multiple tool calls in parallel.
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`
	// Prediction provides predicted output hints used by supported models.
	Prediction *ChatPrediction `json:"prediction,omitempty"`
	// PresencePenalty encourages topic diversity by penalizing seen tokens.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"` // binding:"omitempty,min=-2.0,max=2.0"
	// PromptCacheKey is the key used for prompt caching.
	PromptCacheKey string `json:"prompt_cache_key,omitempty"`
	// PromptCacheRetention controls prompt-cache retention strategy.
	PromptCacheRetention string `json:"prompt_cache_retention,omitempty"` // Enum: "24h" | "in-memory"
	// ReasoningEffort controls reasoning depth/compute budget on supported models.
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // Enum: "none" | "xhigh" | "high" | "medium" | "low" | "minimal"
	// ResponseFormat defines output format constraints (text/json_schema/json_object).
	ResponseFormat *ChatResponseFormat `json:"response_format,omitempty"`
	// SafetyIdentifier is a caller-provided safety/audit identifier.
	SafetyIdentifier string `json:"safety_identifier,omitempty"`
	// ServiceTier requests a serving tier (for example auto/default/flex/priority).
	ServiceTier string `json:"service_tier,omitempty"`
	// Deprecated
	// Seed is the legacy deterministic-seeding field.
	Seed int64 `json:"seed,omitempty"`
	// Stop provides one or more stop sequences to terminate generation.
	Stop json.RawMessage `json:"stop,omitempty"` // string | []string
	// Store controls whether output may be stored by the platform.
	Store *bool `json:"store,omitempty"`
	// Stream enables Server-Sent Events (SSE) streaming output.
	Stream *bool `json:"stream"`
	// StreamOptions controls streaming details (for example include_usage in final chunk).
	StreamOptions *ChatStreamOptions `json:"stream_options,omitempty"`
	// Temperature controls randomness in token sampling.
	Temperature *float64 `json:"temperature,omitempty" binding:"omitempty,min=0,max=2.0"`
	// ToolChoice controls if/how tools are called (none/auto/required/named tool).
	ToolChoice json.RawMessage `json:"tool_choice,omitempty"` // string("none" | "auto" | "required") | ChatToolChoice
	// Tools declares available tools that the model can call.
	Tools []ChatTool `json:"tools,omitempty"`
	// TopLogprobs controls how many top token alternatives are returned (0-20).logprobs must be true
	TopLogprobs *int64 `json:"top_logprobs,omitempty" binding:"omitempty,min=0,max=20"`
	// TopP sets nucleus sampling threshold.
	TopP *float64 `json:"top_p,omitempty" binding:"omitempty,min=0,max=1.0"`
	// Deprecated: This field is being replaced by safety_identifier and prompt_cache_key.
	// User is the legacy user identifier field.
	User string `json:"user,omitempty"`
	// Verbosity controls response detail level on supported models.
	Verbosity string `json:"verbosity,omitempty" enum:"high,medium,low"` // Enum: "high" | "medium" | "low"
	// WebSearchOptions configures web-search behavior when web search is enabled.
	WebSearchOptions *ChatWebSearchOptions `json:"web_search_options,omitempty"`
}

type (
	// ChatDeveloperMessage 是 chat 消息的基础结构（developer/system/user 复用）。
	ChatDeveloperMessage struct {
		Role    string          `json:"role"`    // role = “developer”
		Content json.RawMessage `json:"content"` // string or [] ChatContentPartText
		Name    string          `json:"name,omitempty"`
	}
	ChatSystemMessage = ChatDeveloperMessage // role = "system"
	ChatUserMessage   = ChatDeveloperMessage // role = "user" content = string | [] ChatContentPartText or ChatContentPartImage or ChatContentPartInputAudio or ChatContentPartFile
	// ChatAssistantMessage 表示 assistant 侧返回或回填的消息结构，支持 tool_calls。
	ChatAssistantMessage struct {
		Role    string          `json:"role"` // role = "assistant"
		Audio   *IdObject       `json:"audio,omitempty"`
		Content json.RawMessage `json:"content"` // string | [] ChatContentPartText | ChatContentPartRefusal
		// Deprecated: Use tool_calls instead.
		FunctionCall *ChatMessageFunctionCall `json:"function_call,omitempty"`
		Name         string                   `json:"name,omitempty"`
		Refusal      string                   `json:"refusal,omitempty"`
		ToolCalls    []ChatMessageToolCall    `json:"tool_calls,omitempty"`
	}
	// ChatToolMessage 表示 tool 角色消息，通常用于回传工具执行结果。
	ChatToolMessage struct {
		Role       string          `json:"role"`    // role = "tool"
		Content    json.RawMessage `json:"content"` // string | [] ChatContentPartText
		ToolCallId string          `json:"tool_call_id"`
	}
	ChatFunctionMessage = ChatDeveloperMessage // role = "function"

)

type (
	ChatContentPartText  = TypeTextObject // type = "text"
	ChatContentPartImage struct {
		Type     string `json:"type" default:"image_url"`
		ImageUrl struct {
			Url    string `json:"url"`
			Detail string `json:"detail,omitempty"`
		} `json:"image_url"`
	}
	ChatContentPartInputAudio struct {
		Type       string `json:"type" default:"input_audio"`
		InputAudio struct {
			Data   string `json:"data"`
			Format string `json:"format"`
		} `json:"input_audio"`
	}
	ChatContentPartFile struct {
		Type string `json:"type" default:"file"`
		File struct {
			FileData string `json:"file_data,omitempty"`
			FileId   string `json:"file_id,omitempty"`
			Filename string `json:"filename,omitempty"`
		} `json:"file"`
	}
	ChatContentPartRefusal struct {
		Type    string `json:"type" default:"refusal"`
		Refusal string `json:"refusal"`
	}
)
