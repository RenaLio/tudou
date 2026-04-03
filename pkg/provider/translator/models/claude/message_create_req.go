package claude

import "encoding/json"

type CreateMessageRequest struct {
	MaxTokens    *int64                 `json:"max_tokens,omitempty"`
	Messages     []MessageParam         `json:"messages"`
	Model        string                 `json:"model"`
	CacheControl *CacheControlEphemeral `json:"cache_control,omitempty"`
	// Container identifier for reuse across requests.
	//
	// 容器标识符，用于在请求之间重复使用。
	Container string `json:"container,omitempty"`
	// Specifies the geographic region for inference processing. If not specified, the workspace's `default_inference_geo` is used.
	//
	// 指定用于推理处理的地理区域。如果未指定，则使用工作区的 default_inference_geo 。
	InferenceGeo string `json:"inference_geo,omitempty"`
	// An object describing metadata about the request.
	//
	// 一个描述请求元数据的对象。
	// {user_id:string}
	MetaData     json.RawMessage `json:"metadata,omitempty"`
	OutputConfig *OutputConfig   `json:"output_config,omitempty"`
	ServiceTier  *string         `json:"service_tier,omitempty" enum:"auto,standard_only"` // auto | standard_only
	// Custom text sequences that will cause the model to stop generating.
	//
	// 自定义文本序列，模型将在生成过程中遇到这些文本时停止。
	//
	//  optional: []string
	StopSequences json.RawMessage `json:"stop_sequences,omitempty"`
	Stream        *bool           `json:"stream,omitempty"`
	// optional: string | [] TextBlock
	System      json.RawMessage   `json:"system,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	Thinking    *ThinkingParam    `json:"thinking,omitempty"`
	ToolChoice  json.RawMessage   `json:"tool_choice,omitempty"` // *ToolChoice
	Tools       []json.RawMessage `json:"tools,omitempty"`
	TopK        *int64            `json:"top_k,omitempty"`
	TopP        *float64          `json:"top_p,omitempty"`
}

type MessageParam struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` //  string | array of objects
}

type (
	CacheControlEphemeral struct {
		Type string `json:"type" default:"ephemeral"`
		TTL  string `json:"ttl,omitempty" enum:"5m,1h"`
	}
)

type (
	TextBlockParam struct {
		Type         string                 `json:"type"`
		Text         string                 `json:"text"`
		CacheControl *CacheControlEphemeral `json:"cache_control,omitempty"`
		Citations    []json.RawMessage      `json:"citations,omitempty"`
	}
	ImageBlockParam struct {
		Type         string                 `json:"type"`
		Source       ImageSourceParam       `json:"source"`
		CacheControl *CacheControlEphemeral `json:"cache_control,omitempty"`
	}
	DocumentBlockParam struct {
		Type         string                 `json:"type"`
		Source       DocumentSourceParam    `json:"source"`
		CacheControl *CacheControlEphemeral `json:"cache_control,omitempty"`
		Citations    json.RawMessage        `json:"citations,omitempty"`
		Context      string                 `json:"context,omitempty"`
		Title        string                 `json:"title,omitempty"`
	}
	SearchResultParam struct {
		Type         string                 `json:"type"`
		Content      []TextBlockParam       `json:"content"`
		Source       string                 `json:"source"`
		Title        string                 `json:"title"`
		CacheControl *CacheControlEphemeral `json:"cache_control,omitempty"`
		Citations    json.RawMessage        `json:"citations,omitempty"`
	}
	ThinkingBlockParam struct {
		Type      string `json:"type"`
		Thinking  string `json:"thinking"`
		Signature string `json:"signature"`
	}
	RedactedThinkingBlockParam struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	ToolUseBlockParam struct {
		Type         string                     `json:"type"`
		ID           string                     `json:"id"`
		Name         string                     `json:"name"`
		Input        map[string]json.RawMessage `json:"input"`
		CacheControl json.RawMessage            `json:"cache_control,omitempty"`
		Caller       json.RawMessage            `json:"caller,omitempty"`
	}
	ToolResultBlockParam struct {
		Type         string          `json:"type"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
		Content      json.RawMessage `json:"content"`
		IsError      bool            `json:"is_error"`
	}
	ToolReferenceBlockParam struct {
		Type         string          `json:"type"`
		ToolName     string          `json:"tool_name"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
	ServerToolUseBlockParam struct {
		Type         string          `json:"type"`
		Id           string          `json:"id"`
		Input        json.RawMessage `json:"input"`
		Name         string          `json:"name"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
		Caller       json.RawMessage `json:"caller,omitempty"`
	}
	WebSearchToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
		Caller       json.RawMessage `json:"caller,omitempty"`
	}
	WebFetchToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
		Caller       json.RawMessage `json:"caller,omitempty"`
	}
	CodeExecutionToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
	BashCodeExecutionToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
	TextEditorCodeExecutionToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
	ToolSearchToolResultBlockParam struct {
		Type         string          `json:"type"`
		Content      json.RawMessage `json:"content"`
		ToolUseId    string          `json:"tool_use_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
	ContainerUploadBlockPara struct {
		Type         string          `json:"type"`
		FileId       string          `json:"file_id"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`
	}
)
type (
	ImageSourceParam struct {
		Type      string `json:"type"`
		Data      string `json:"data,omitempty"`
		MediaType string `json:"media_type,omitempty"`
		Url       string `json:"url,omitempty"`
	}
	DocumentSourceParam struct {
		Type      string `json:"type"`
		Data      string `json:"data,omitempty"`
		MediaType string `json:"media_type,omitempty"`
		Url       string `json:"url,omitempty"`
		Content   string `json:"content,omitempty"`
	}
)
type (
	Tool struct {
		Name         string          `json:"name"`
		Type         string          `json:"type,omitempty"`
		InputSchema  json.RawMessage `json:"input_schema"`
		AllowCallers []string        `json:"allow_callers,omitempty"`

		DeferLoading        *bool           `json:"defer_loading,omitempty"`
		Description         string          `json:"description,omitempty"`
		EagerInputStreaming *bool           `json:"eager_input_streaming,omitempty"`
		InputExamples       json.RawMessage `json:"input_examples,omitempty"`
		Strict              *bool           `json:"strict,omitempty"`
	}

	ToolChoice struct {
		Type string `json:"type"`

		Name                   *string `json:"name,omitempty"`
		DisableParallelToolUse *bool   `json:"disable_parallel_tool_use,omitempty"`
	}
	InputSchema struct {
		Type       string          `json:"type"`
		Properties json.RawMessage `json:"properties"`
		required   json.RawMessage `json:"required"`
	}

	ToolUnion struct {
		Name string `json:"name"`
		Type string `json:"type"`

		AllowCallers []string        `json:"allow_callers,omitempty"`
		CacheControl json.RawMessage `json:"cache_control,omitempty"`

		DeferLoading        bool   `json:"defer_loading,omitempty"`
		Description         string `json:"description,omitempty"`
		EagerInputStreaming bool   `json:"eager_input_streaming,omitempty"`

		InputExamples json.RawMessage `json:"input_examples,omitempty"`
		Strict        bool            `json:"strict,omitempty"`

		MaxCharacters int `json:"max_characters,omitempty"`

		InputSchema json.RawMessage `json:"input_schema"`

		AllowDomains   []string      `json:"allow_domains,omitempty"`
		BlockedDomains []string      `json:"blocked_domains,omitempty"`
		MaxUses        int           `json:"max_uses,omitempty"`
		UserLocation   *UserLocation `json:"user_location,omitempty"`

		MaxContentTokens int `json:"max_content_tokens,omitempty"`
	}
)

type (
	OutputConfig struct {
		Effort string            `json:"effort,omitempty" enum:"low,medium,high,max"`
		Format *JSONOutputFormat `json:"format,omitempty"`
	}
	JSONOutputFormat struct {
		Schema json.RawMessage `json:"schema"`
		Type   string          `json:"type" default:"json_schema"`
	}

	UserLocation struct {
		Type     string `json:"type"` //approximate
		City     string `json:"city,omitempty"`
		Country  string `json:"country,omitempty"`
		Region   string `json:"region,omitempty"`
		Timezone string `json:"timezone,omitempty"`
	}
	ThinkingParam struct {
		Type         string `json:"type"`
		BudgetTokens *int64 `json:"budget_tokens,omitempty"`
		Display      string `json:"display,omitempty"`
	}
)
