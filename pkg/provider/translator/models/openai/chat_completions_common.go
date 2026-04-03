package openai

import "encoding/json"

type (
	// ChatTool 定义请求中可用的工具声明（function/custom）。
	ChatTool struct {
		Type     string    `json:"type"`               // "function" | "custom"
		Function *Function `json:"function,omitempty"` // only when type is "function"
		Custom   *Custom   `json:"custom,omitempty"`   // only when type is "custom"
	}

	ChatMessageFunctionCall struct {
		Name      string `json:"name" binding:"required"`
		Arguments string `json:"arguments,omitempty"`
	}
	// ChatMessageToolCall 表示 assistant 在消息中产生的单次工具调用。
	ChatMessageToolCall struct {
		Id       string                       `json:"id"`
		Type     string                       `json:"type"`
		Function *ChatMessageToolCallFunction `json:"function,omitempty"`
		Custom   *ChatMessageToolCallCustom   `json:"custom,omitempty"`
	}
	ChatMessageToolCallFunction struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	ChatMessageToolCallCustom struct {
		Name  string `json:"name"`
		Input string `json:"input"`
	}
	// ChatToolChoice 控制模型是否/如何调用工具（none/auto/required/指定工具）。
	ChatToolChoice struct {
		Type         string          `json:"type,omitempty"`          // "function" | "custom" | "allowed_tools"
		AllowedTools *ChatAllowTools `json:"allowed_tools,omitempty"` // only when type is "allowed_tools"
		Function     *NameObject     `json:"function,omitempty"`      // only when type is "function"
		Custom       *NameObject     `json:"custom,omitempty"`        // only when type is "custom"
	}
	ChatAllowTools struct {
		Mode  string          `json:"mode,omitempty"`
		Tools json.RawMessage `json:",omitempty"`
	}
)

type (
	ChatAudioParam struct {
		Format string `json:"format"`
		Voice  any    `json:"voice,omitempty"`
	}
	ChatPrediction struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"` // string | array(TypeTextObject)
	}
	// ChatResponseFormat 指定输出格式（text/json_schema/json_object）。
	ChatResponseFormat struct {
		Type       string      `json:"type,omitempty"`        // "text" | "json_schema" | "json_object"
		JsonSchema *JSONSchema `json:"json_schema,omitempty"` // only when type is "json_schema"
	}
	ChatStreamOptions struct {
		IncludeObfuscation bool `json:"include_obfuscation,omitempty"`
		IncludeUsage       bool `json:"include_usage,omitempty"`
	}
)

type (
	ChatWebSearchOptions struct {
		SearchContentSize string        `json:"search_content_size,omitempty" enum:"low,medium,high"` // Enum: "low" | "medium" | "high"
		UserLocation      *UserLocation `json:"user_location,omitempty"`
	}
)

type (
	ChatCompletionAudio struct {
		Id         string `json:"id"`
		Data       string `json:"data,omitempty"`
		ExpiresAt  int64  `json:"expires_at,omitempty"`
		Transcript string `json:"transcript,omitempty"`
	}
)

type (
	ChatCompletionLogprobs struct {
		Content []ChatCompletionLogprobsContent `json:"content,omitempty"`
		Refusal []ChatCompletionLogprobsRefusal `json:"refusal,omitempty"`
	}
	ChatCompletionLogprobsContent struct {
		Token       string       `json:"token"`
		Bytes       []byte       `json:"bytes"`
		Logprob     float64      `json:"logprob"`
		TopLogProbs []TopLogProb `json:"top_logprobs"`
	}
	ChatCompletionLogprobsRefusal = ChatCompletionLogprobsContent
	TopLogProb                    struct {
		Token   string  `json:"token"`
		Bytes   []byte  `json:"bytes"`
		Logprob float64 `json:"logprob"`
	}
)

type (
	// ChatCompletionUsage 表示一次 chat completion 的 token 使用统计。
	ChatCompletionUsage struct {
		CompletionTokens        int64                    `json:"completion_tokens"`
		PromptTokens            int64                    `json:"prompt_tokens"`
		TotalTokens             int64                    `json:"total_tokens"`
		CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
		PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	}
	CompletionTokensDetails struct {
		AcceptedPredictionTokens int64 `json:"accepted_prediction_tokens,omitempty"`
		AudioTokens              int64 `json:"audio_tokens,omitempty"`
		ReasoningTokens          int64 `json:"reasoning_tokens,omitempty"`
		RejectedPredictionTokens int64 `json:"rejected_prediction_tokens,omitempty"`
	}
	PromptTokensDetails struct {
		AudioTokens  int64 `json:"audio_tokens,omitempty"`
		CachedTokens int64 `json:"cached_tokens,omitempty"`
	}
)
