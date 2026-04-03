package openai

type (
	// ChatCompletion 是 /v1/chat/completions 的标准非流式响应对象。
	ChatCompletion struct {
		Id          string                 `json:"id"`
		Choices     []ChatCompletionChoice `json:"choices"`
		Created     int64                  `json:"created"`
		Model       string                 `json:"model"`
		Object      string                 `json:"object"`
		ServiceTier string                 `json:"service_tier,omitempty"`
		Usage       *ChatCompletionUsage   `json:"usage,omitempty"`
		// Deprecated: SystemFingerprint is deprecated.
		SystemFingerprint string `json:"system_fingerprint,omitempty"`
	}
	// ChatCompletionChoice 表示一次候选生成结果。
	ChatCompletionChoice struct {
		FinishReason string                      `json:"finish_reason"`
		Index        int64                       `json:"index"`
		Logprobs     *ChatCompletionLogprobs     `json:"logprobs,omitempty"`
		Message      ChatCompletionChoiceMessage `json:"message"`
	}
	// ChatCompletionChoiceMessage 表示 choice 中的最终消息载荷。
	ChatCompletionChoiceMessage struct {
		Content     string                     `json:"content,omitempty"`
		Refusal     string                     `json:"refusal,omitempty"` // string or null
		Role        string                     `json:"role"`
		Annotations []ChatCompletionAnnotation `json:"annotations,omitempty"`
		Audio       *ChatCompletionAudio       `json:"audio,omitempty"`
		// Deprecated: Use tool_calls instead.
		FunctionCall *ChatMessageFunctionCall `json:"function_call,omitempty"`
		ToolCalls    []ChatMessageToolCall    `json:"tool_calls,omitempty"`

		// for reasoning content
		ReasoningContent string `json:"reasoning_content,omitempty"`
	}
	ChatCompletionAnnotation struct {
		Type        string                     `json:"type"`
		UrlCitation *ChatCompletionUrlCitation `json:"url_citation,omitempty"`
	}
	ChatCompletionUrlCitation struct {
		StartIndex int    `json:"start_index"`
		EndIndex   int    `json:"end_index"`
		Title      string `json:"title"`
		Url        string `json:"url"`
	}
)

type (
	// ChatCompletionStream 表示 streaming 模式下每个 chunk 的数据结构。
	ChatCompletionStream struct {
		Id          string              `json:"id"`
		Choices     []StreamChoiceDelta `json:"choices"`
		Created     int64               `json:"created"`
		Model       string              `json:"model"`
		Object      string              `json:"object"`
		ServiceTier string              `json:"service_tier,omitempty"`
		// Deprecated: SystemFingerprint
		SystemFingerprint string               `json:"system_fingerprint,omitempty"`
		Usage             *ChatCompletionUsage `json:"usage,omitempty"`
	}

	// StreamChoiceDelta 表示 chunk 中一个 choice 的增量部分。
	StreamChoiceDelta struct {
		Delta        ChoiceDelta             `json:"delta"`
		FinishReason string                  `json:"finish_reason,omitempty"`
		Index        int64                   `json:"index"`
		Logprobs     *ChatCompletionLogprobs `json:"logprobs,omitempty"`
	}
	// ChoiceDelta 是 streaming 下的消息增量体（内容、角色、工具调用等）。
	ChoiceDelta struct {
		Content string `json:"content,omitempty"`
		Refusal string `json:"refusal,omitempty"` // string or null
		Role    string `json:"role"`
		// Deprecated: Use tool_calls instead.
		FunctionCall *ChatMessageFunctionCall `json:"function_call,omitempty"`
		ToolCalls    []ChoiceDeltaToolCall    `json:"tool_calls,omitempty"`
		// for reasoning content
		ReasoningContent string `json:"reasoning_content,omitempty"`
	}
	ChoiceDeltaToolCall struct {
		ChatMessageToolCall
		Index int `json:"index"`
	}
)
