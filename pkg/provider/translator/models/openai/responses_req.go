package openai

import "encoding/json"

type CreateResponseRequest struct {
	Background           *bool               `json:"background,omitempty"`
	ContextManagement    []ContextManagement `json:"context_management,omitempty"`
	Conversation         json.RawMessage     `json:"conversation,omitempty"` // string | ResponseConversationParam
	Include              []string            `json:"include,omitempty"`
	Input                json.RawMessage     `json:"input,omitempty"`        // string or [] ResponseInputItem
	Instructions         string              `json:"instructions,omitempty"` // string
	MaxOutputTokens      *int64              `json:"max_output_tokens,omitempty"`
	MaxToolCalls         *int64              `json:"max_tool_calls,omitempty"`
	Metadata             json.RawMessage     `json:"metadata,omitempty"`
	Model                string              `json:"model,omitempty"`
	ParallelToolCalls    *bool               `json:"parallel_tool_calls,omitempty"`
	PreviousResponseID   string              `json:"previous_response_id,omitempty"`
	Prompt               *ResponsePrompt     `json:"prompt,omitempty"`
	PromptCacheKey       string              `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string              `json:"prompt_cache_retention,omitempty"`
	Reasoning            *Reasoning          `json:"reasoning,omitempty"`
	SafetyIdentifier     string              `json:"safety_identifier,omitempty"`
	ServiceTier          string              `json:"service_tier,omitempty" binding:"omitempty,oneof=auto default flex priority"`
	Store                *bool               `json:"store,omitempty"`
	Stream               *bool               `json:"stream,omitempty"`
	StreamOptions        *StreamOptions      `json:"stream_options,omitempty"`
	Temperature          *float64            `json:"temperature,omitempty" binding:"omitempty,min=0,max=2"`
	Text                 *ResponseTextConfig `json:"text,omitempty"`
	ToolChoice           json.RawMessage     `json:"tool_choice,omitempty"` // string or object（ ResponseToolChoice ）
	Tools                []json.RawMessage   `json:"tools,omitempty"`       // Tool
	TopLogprobs          *int64              `json:"top_logprobs,omitempty"`
	TopP                 *float64            `json:"top_p,omitempty"`
	Truncation           string              `json:"truncation,omitempty"`
	// Deprecated: This field is being replaced by safety_identifier and prompt_cache_key
	User string `json:"user,omitempty"`
}

type (
	ContextManagement struct {
		Type             string `json:"type"`
		CompactThreshold *int64 `json:"compact_threshold,omitempty"`
	}
	ResponseConversationParam = IdObject
	ResponsePrompt            struct {
		ID string `json:"id"`
		//Variables map[string]json.RawMessage `json:"variables,omitempty"` // string | ResponseInputContent
		Variables json.RawMessage `json:"variables,omitempty"` // string | ResponseInputContent
		Version   string          `json:"version,omitempty"`
	}
	Reasoning struct {
		Effort string `json:"effort,omitempty"`
		// Deprecated: use summary instead.
		GenerateSummary string `json:"generate_summary,omitempty"`
		Summary         string `json:"summary,omitempty"`
	}
	StreamOptions struct {
		IncludeObfuscation *bool `json:"include_obfuscation,omitempty"`
	}
	ResponseTextConfig struct {
		Format    *ResponseTextConfigFormat `json:"format,omitempty"`
		Verbosity string                    `json:"verbosity,omitempty"`
	}
	ResponseTextConfigFormat struct {
		Type string `json:"type,omitempty"`
		JSONSchema
	}

	ResponseUserLocation struct {
		Approximate
		UserLocation
	}
)

type (
	ResponseInputContent interface {
		ResponseInputText | ResponseInputImage | ResponseInputFile
	}
	ResponseInputText  = TypeTextObject // type = "text"
	ResponseInputImage struct {
		Detail   string `json:"detail"`
		Type     string `json:"type"` // "image_url"
		FileId   string `json:"file_id,omitempty"`
		ImageUrl string `json:"image_url,omitempty"`
	}
	ResponseInputFile struct {
		Type     string `json:"type"` // "input_file"
		FileId   string `json:"file_id,omitempty"`
		FileData string `json:"file_data,omitempty"`
		FileUrl  string `json:"file_url,omitempty"`
		FileName string `json:"filename,omitempty"`
	}
)

type (
	// ResponseToolChoice  = ToolChoiceAllowed | ToolChoiceTypes | ToolChoiceFunction | ToolChoiceMcp | ToolChoiceCustom | ToolChoiceApplyPatch | ToolChoiceShell
	ResponseToolChoice struct {
		Type        string          `json:"type"`
		Name        string          `json:"name,omitempty"`
		Mode        string          `json:"mode,omitempty"`  // for "allowed_tools" type, enum: "auto" | "required"
		Tools       json.RawMessage `json:"tools,omitempty"` // for "allowed_tools" type
		ServerLabel string          `json:"server_label"`    // for "mcp" type
	}
	ToolChoiceAllowed struct {
		Type  string          `json:"type"` // "allowed_tools"
		Mode  string          `json:"mode"` // "auto" | "required"
		Tools json.RawMessage `json:"tools,omitempty"`
	}
	ToolChoiceTypes    = TypeObject     // "file_search" | "web_search_preview" | "computer_use_preview" | "code_interpreter" | "image_generation"
	ToolChoiceFunction = NameTypeObject // "function"
	ToolChoiceMcp      struct {
		Type        string `json:"type"` // "mcp"
		ServerLabel string `json:"server_label"`
		Name        string `json:"name"`
	}
	ToolChoiceCustom     = NameTypeObject // "custom"
	ToolChoiceApplyPatch = TypeObject     // "apply_patch"
	ToolChoiceShell      = TypeObject     // "shell"
)

type (
	FunctionTool   = Function
	FileSearchTool struct {
		Type           string          `json:"type"` // "file_search"
		VectorStoreIDs []string        `json:"vector_store_ids,omitempty"`
		Filters        json.RawMessage `json:"filters,omitempty"`
		MaxNumResults  int             `json:"max_num_results,omitempty"`
		RankingOptions json.RawMessage `json:"ranking_options,omitempty"`
	}
	ComputerTool struct {
		Type          string `json:"type"` // "computer_use_preview"
		DisplayHeight int    `json:"display_height,omitempty"`
		DisplayWidth  int    `json:"display_width,omitempty"`
		Environment   string `json:"environment,omitempty"` // "windows" | "mac" | "linux"
	}
	WebSearchTool struct {
		Type              string                `json:"type"` // "web_search_preview"
		Filters           json.RawMessage       `json:"filters,omitempty"`
		SearchContentSize string                `json:"search_content_size,omitempty"`
		UserLocation      *ResponseUserLocation `json:"user_location,omitempty"`
	}
	Mcp struct {
		Type              string                     `json:"type"` // "mcp"
		ServerLabel       string                     `json:"server_label"`
		AllowedTools      json.RawMessage            `json:"allowed_tools,omitempty"`
		Authorization     string                     `json:"authorization,omitempty"`
		ConnectorId       string                     `json:"connector_id,omitempty"` //  One of server_url or connector_id must be provided.
		Headers           map[string]json.RawMessage `json:"headers,omitempty"`
		RequireApproval   string                     `json:"require_approval,omitempty"`
		ServerDescription string                     `json:"server_description,omitempty"`
		ServerURL         string                     `json:"server_url,omitempty"` // The URL for the MCP server. One of server_url or connector_id must be provided.
	}
	CodeInterpreter struct {
		Type      string          `json:"type"` // "code_interpreter"
		Container json.RawMessage `json:"container,omitempty"`
	}
	ImageGeneration struct {
		Type              string          `json:"type"` // "image_generation"
		Background        string          `json:"background,omitempty"`
		InputFidelity     string          `json:"input_fidelity,omitempty"`
		InputImageMask    json.RawMessage `json:"input_image_mask,omitempty"`
		Model             string          `json:"model,omitempty"`
		Moderation        string          `json:"moderation,omitempty"`
		OutputCompression int             `json:"output_compression,omitempty"`
		OutputFormat      string          `json:"output_format,omitempty"`
		PartialImages     int             `json:"partial_images,omitempty"`
		Quality           string          `json:"quality,omitempty"`
		Size              string          `json:"size,omitempty"`
	}
	LocalShell    = TypeObject // "local_shell"
	FunctionShell struct {
		Type        string          `json:"type"` // "shell"
		Environment json.RawMessage `json:"environment,omitempty"`
	}
	CustomTool = Custom
	// WebSearchPreviewTool is earlier Tool for WebSearchTool, use WebSearchTool First
	WebSearchPreviewTool struct {
		Type          string                `json:"type"` // "web_search_preview" or "web_search_preview_2025_03_11"
		SearchContext string                `json:"search_context,omitempty"`
		UserLocation  *ResponseUserLocation `json:"user_location,omitempty"`
	}
	ApplyPatch = TypeObject // "apply_patch"
)

type (
	ResponseInputItem interface {
		EasyInputMessage | Message | ResponseOutputMessage // ...
	}
	EasyInputMessage struct {
		Type    string          `json:"type,omitempty"`    // "message"
		Content json.RawMessage `json:"content,omitempty"` // string | [] ResponseInputContent
		Role    string          `json:"role"`              // "user" or "assistant" or "system" or "developer"
	}
	Message struct {
		Type    string          `json:"type,omitempty"`    // "message"
		Content json.RawMessage `json:"content,omitempty"` // [] ResponseInputContent
		Role    string          `json:"role"`              // "user" or "system" or "developer"
		Status  string          `json:"status,omitempty"`

		Id string `json:"id,omitempty"`
	}
	ResponseOutputMessage struct {
		Type    string          `json:"type,omitempty"` // "message"
		Id      string          `json:"id,omitempty"`
		Content json.RawMessage `json:"content,omitempty"` // [] ResponseOutputText | [] ResponseOutputRefusal
		Role    string          `json:"role"`              // "assistant"
		Status  string          `json:"status,omitempty"`
	}
	ResponseFileSearchToolCall struct {
		Type    string          `json:"type"` // "file_search_call"
		Id      string          `json:"id"`
		Queries []string        `json:"queries"`
		Status  string          `json:"status,omitempty"`
		Results json.RawMessage `json:"results,omitempty"`
	}
	ResponseComputerToolCall struct {
		Type                string          `json:"type"` // "computer_call"
		Id                  string          `json:"id"`
		Action              json.RawMessage `json:"action"`
		CallID              string          `json:"call_id"`
		PendingSafetyChecks json.RawMessage `json:"pending_safety_checks,omitempty"` //[]{id,code,message}
		Status              string          `json:"status,omitempty"`
	}
	ResponseComputerCallOutputCallOut struct {
		Type                string          `json:"type"` // "computer_call_output"
		CallID              string          `json:"call_id"`
		Output              json.RawMessage `json:"output"`
		Id                  string          `json:"id"`
		PendingSafetyChecks json.RawMessage `json:"pending_safety_checks,omitempty"` //[]{id,code,message}
		Status              string          `json:"status,omitempty"`
	}
	ResponseFunctionWebSearch struct {
		Type   string          `json:"type"` // "web_search_call"
		Id     string          `json:"id"`
		Action json.RawMessage `json:"action"`
		Status string          `json:"status,omitempty"`
	}
	ResponseFunctionCall struct {
		Type      string `json:"type"` // "function_call"
		Arguments string `json:"arguments"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Id        string `json:"id,omitempty"`
		Status    string `json:"status,omitempty"`
	}
	FunctionCallOutput struct {
		Type   string          `json:"type"`   // "function_call_output"
		Output json.RawMessage `json:"output"` // string or [] ResponseInputContent
		CallId string          `json:"call_id"`
		Id     string          `json:"id,omitempty"`
		Status string          `json:"status,omitempty"`
	}
	ResponseReasoningItem struct {
		Type             string           `json:"type"` // "reasoning"
		Id               string           `json:"id,omitempty"`
		Summary          []TypeTextObject `json:"summary,omitempty"` // type = "summary_text"
		Content          []TypeTextObject `json:"content,omitempty"` // type = "reasoning_text"
		EncryptedContent string           `json:"encrypted_content,omitempty"`
		Status           string           `json:"status,omitempty"`
	}
	ResponseCompactionItemParam struct {
		Type             string `json:"type"` // "compaction"
		EncryptedContent string `json:"encrypted_content,omitempty"`
		Id               string `json:"id,omitempty"`
	}
	ImageGenerationCall struct {
		Type   string `json:"type"` // "image_generation_call"
		Id     string `json:"id"`
		Result string `json:"result"`
		Status string `json:"status,omitempty"`
	}
	ResponseCodeInterpreterToolCall struct {
		Type        string          `json:"type"` // "code_interpreter_call"
		Id          string          `json:"id"`
		Code        string          `json:"code"`
		ContainerId string          `json:"container_id"`
		Outputs     json.RawMessage `json:"outputs"`
		Status      string          `json:"status,omitempty"`
	}
	LocalShellCall struct {
		Type   string          `json:"type"` // "local_shell_call"
		Action json.RawMessage `json:"action"`
		CallID string          `json:"call_id"`
		Id     string          `json:"id"`
		Status string          `json:"status,omitempty"`
	}
	LocalShellCallOutput struct {
		Type   string `json:"type"` // "local_shell_call_output"
		Id     string `json:"id"`
		Status string `json:"status,omitempty"`
		Output string `json:"output"`
	}
	ShellCall struct {
		Type        string          `json:"type"` // "shell_call"
		Action      json.RawMessage `json:"action"`
		CallID      string          `json:"call_id"`
		Id          string          `json:"id"`
		Status      string          `json:"status,omitempty"`
		Environment json.RawMessage `json:"environment"`
	}
	ShellCallOutput struct {
		Type            string          `json:"type"` // "shell_call_output"
		CallID          string          `json:"call_id"`
		Output          json.RawMessage `json:"output"`
		Id              string          `json:"id"`
		Status          string          `json:"status,omitempty"`
		MaxOutputLength int             `json:"max_output_length"`
	}
	ApplyPatchCall struct {
		Type      string          `json:"type"` // "apply_patch_call"
		CallId    string          `json:"call_id"`
		Operation json.RawMessage `json:"operation"`
		Status    string          `json:"status,omitempty"`
		Id        string          `json:"id,omitempty"`
	}
	ApplyPatchCallOutput struct {
		Type   string `json:"type"` // "apply_patch_call_output"
		CallId string `json:"call_id"`
		Id     string `json:"id,omitempty"`
		Output string `json:"output,omitempty"`
		Status string `json:"status,omitempty"`
	}
	McpListTools struct {
		Type        string          `json:"type"` // "mcp_list_tools"
		Id          string          `json:"id"`
		ServerLabel string          `json:"server_label"`
		Tools       json.RawMessage `json:"tools"`
		Error       string          `json:"error,omitempty"`
	}
	McpApprovalRequest struct {
		Type        string `json:"type"` // "mcp_approval_request"
		Id          string `json:"id"`
		Arguments   string `json:"arguments"`
		ServerLabel string `json:"server_label"`
		Reason      string `json:"reason,omitempty"`
	}
	McpApprovalResponse struct {
		Type             string `json:"type"` // "mcp_approval_response"
		ApproveRequestId string `json:"approve_request_id"`
		Approve          bool   `json:"approve"`
		Id               string `json:"id,omitempty"`
		Reason           string `json:"reason,omitempty"`
	}
	McpCall struct {
		Type           string `json:"type"` // "mcp_call"
		Id             string `json:"id"`
		Arguments      string `json:"arguments"`
		Name           string `json:"name"`
		ServerLabel    string `json:"server_label"`
		ApproveRequest string `json:"approve_request,omitempty"`
		Error          string `json:"error,omitempty"`
		Output         string `json:"output,omitempty"`
		Status         string `json:"status,omitempty"`
	}
	ResponseCustomToolCallOutput struct {
		// call_id: string
		// The call ID, used to map this custom tool call output to a custom tool call.
		CallId string `json:"call_id"`

		// output: string or array of ResponseInputContent
		// The output from the custom tool call generated by your code.
		// Can be a string or a list of output content.
		Output json.RawMessage `json:"output"`

		// type: "custom_tool_call_output"
		// The type of the custom tool call output. Always custom_tool_call_output.
		Type string `json:"type"`

		// id: optional string
		// The unique ID of the custom tool call output in the OpenAI platform.
		ID string `json:"id,omitempty"`
	}
	ResponseCustomToolCall struct {
		// call_id: string
		// An identifier used to map this custom tool call to a tool call output.
		CallId string `json:"call_id"`

		// input: string
		// The input for the custom tool call generated by the model.
		Input string `json:"input"`

		// name: string
		// The name of the custom tool being called.
		Name string `json:"name"`

		// type: "custom_tool_call"
		// The type of the custom tool call. Always custom_tool_call.
		Type string `json:"type"`

		// id: optional string
		// The unique ID of the custom tool call in the OpenAI platform.
		ID *string `json:"id,omitempty"`
	}
	ItemReference struct {
		Type string `json:"type"` // item_reference
		Id   string `json:"id"`
	}
)
