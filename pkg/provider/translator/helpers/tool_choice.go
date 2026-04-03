package helpers

import (
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ToolChoice:	How the model should use the provided tools
// *: this is optional

// claude: from https://platform.claude.com/docs/en/api/messages/create
// - ToolChoiceAuto {"type":"auto","*disable_parallel_tool_use":"false"}
// - ToolChoiceAny {"type":"any","*disable_parallel_tool_use":"false"}
// - ToolChoiceTool {"type":"none"}
// - ToolChoiceNone {"type":"tool","name":"tool_name","*disable_parallel_tool_use":"false"}

// responses: from https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
// - ToolChoiceOptions string: "auto" | "none" | "required"
// - ToolChoiceAllowed {"type":"allowed_tools","mode":"auto|required","tools":[]any}
// - ToolChoiceTypes {"type":"file_search|web_search_preview|computer|computer_use_preview|computer_use|code_interpreter|image_generation"}
// - ToolChoiceFunction {"type":"function","name":"tool_name"}
// - ToolChoiceMcp{"server_label":string,"type":"mcp","*name":*string}
// - ToolChoiceCustom{"type":"custom","name":string}
// - ToolChoiceApplyPatch{"type":"apply_patch"}
// - ToolChoiceShell{}

// OpenAI(chat completions tools): from https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
// - ToolChoiceMode string: "auto" | "none" | "required"
// - ChatCompletionAllowedToolChoice {"type":"allowed_tools","allowed_tools":{"mode":"auto|required","tools":[]any}}
// - ChatCompletionNamedToolChoice {"type":"function","function":{"name":string}}
// - ChatCompletionNamedToolChoiceCustom {"type":"custom","custom":{"name":string}}

// ClaudeToolChoiceToOpenAIChoice : convert claude tool choice to openai tool choice
//
// Parameter:
//   - in - The Claude tool_choice
//
// Returns:
//   - The OpenAI tool_choice
//   - DisableParallelToolUse *bool
//
// Note:
//   - supported type: auto, none, tool:name_tool
func ClaudeToolChoiceToOpenAIChoice(in json.RawMessage) (json.RawMessage, *bool) {
	if in == nil {
		return nil, nil
	}
	switch getType(in) {
	case "any":
		return []byte(`"required"`), GetClaudeDisableParallelToolUse(in)
	case "auto":
		return []byte(`"auto"`), GetClaudeDisableParallelToolUse(in)
	case "none":
		return []byte(`"none"`), GetClaudeDisableParallelToolUse(in)
	case "tool":
		toolName := gjson.GetBytes(in, "name").String()
		if toolName != "" {
			result := `{"type":"function","function":{"name":""}}`
			result, _ = sjson.Set(result, "function.name", toolName)
			return []byte(result), GetClaudeDisableParallelToolUse(in)
		}
	default:
		return []byte(`"auto"`), GetClaudeDisableParallelToolUse(in)
	}
	return nil, nil
}

func GetClaudeDisableParallelToolUse(in json.RawMessage) *bool {
	if gjson.GetBytes(in, "disable_parallel_tool_use").Exists() {
		return new(gjson.GetBytes(in, "disable_parallel_tool_use").Bool())
	}
	return nil
}

// OpenAIChoiceToClaudeToolChoice
// Parameter:
//   - in - The OpenAI tool_choice
//   - disableParallelToolUse - The disable_parallel_tool_use
//
// Returns:
//   - The Claude tool_choice
func OpenAIChoiceToClaudeToolChoice(in json.RawMessage, disableParallelToolUse *bool) json.RawMessage {
	if in == nil {
		return nil
	}
	var s string
	if disableParallelToolUse != nil {
		if *disableParallelToolUse {
			s = `{"type":"","disable_parallel_tool_use":true}`
		} else {
			s = `{"type":"","disable_parallel_tool_use":false}`
		}
	} else {
		s = `{"type":""}`
	}
	// string auto | none
	// ChatCompletionNamedToolChoice
	if !gjson.ParseBytes(in).IsObject() {
		// gjson.ParseBytes(in).String() returns the string without quotes
		switch gjson.ParseBytes(in).String() {
		case "auto":
			result, err := sjson.Set(s, "type", "auto")
			if err != nil {
				return nil
			}
			return []byte(result)
		case "none":
			result, err := sjson.Set(s, "type", "none")
			if err != nil {
				return nil
			}
			return []byte(result)
		case "required":
			result, err := sjson.Set(s, "type", "any")
			if err != nil {
				return nil
			}
			return []byte(result)
		}
	} else {
		switch getType(in) {
		case "function":
			toolName := gjson.GetBytes(in, "function.name").String()
			if toolName != "" {
				result, err := sjson.Set(s, "type", "tool")
				if err != nil {
					return nil
				}
				result, err = sjson.Set(result, "name", toolName)
				if err != nil {
					return nil
				}
				return []byte(result)
			}
		}
	}
	return nil
}

// ClaudeToolChoiceToResponsesChoice converts Claude tool_choice to Responses API tool_choice
//
// Parameter:
//   - in - The Claude tool_choice
//
// Returns:
//   - The Responses API tool_choice
//   - DisableParallelToolUse *bool
//
// Note:
//   - Claude "auto" -> Responses "auto"
//   - Claude "any" -> Responses "required"
//   - Claude "none" -> Responses "none"
//   - Claude "tool" -> Responses {"type":"function","name":"..."}
func ClaudeToolChoiceToResponsesChoice(in json.RawMessage) (json.RawMessage, *bool) {
	if in == nil {
		return nil, nil
	}
	disableParallelToolUse := GetClaudeDisableParallelToolUse(in)
	switch getType(in) {
	case "auto":
		return []byte(`"auto"`), disableParallelToolUse
	case "any":
		return []byte(`"required"`), disableParallelToolUse
	case "none":
		return []byte(`"none"`), disableParallelToolUse
	case "tool":
		toolName := gjson.GetBytes(in, "name").String()
		if toolName != "" {
			result := `{"type":"function","name":""}`
			result, _ = sjson.Set(result, "name", toolName)
			return []byte(result), disableParallelToolUse
		}
	}
	return nil, nil
}

// ResponsesChoiceToClaudeToolChoice converts Responses API tool_choice to Claude tool_choice
//
// Parameter:
//   - in - The Responses API tool_choice
//   - disableParallelToolUse - The disable_parallel_tool_use flag
//
// Returns:
//   - The Claude tool_choice
//
// Note:
//   - Responses "auto" -> Claude {"type":"auto"}
//   - Responses "required" -> Claude {"type":"any"}
//   - Responses "none" -> Claude {"type":"none"}
//   - Responses {"type":"function","name":"..."} -> Claude {"type":"tool","name":"..."}
//   - Other object types (allowed_tools, custom, mcp, etc.) are not supported and return nil
func ResponsesChoiceToClaudeToolChoice(in json.RawMessage, disableParallelToolUse *bool) json.RawMessage {
	if in == nil {
		return nil
	}

	// Build base structure with disable_parallel_tool_use if provided
	var base string
	if disableParallelToolUse != nil {
		if *disableParallelToolUse {
			base = `{"type":"","disable_parallel_tool_use":true}`
		} else {
			base = `{"type":"","disable_parallel_tool_use":false}`
		}
	} else {
		base = `{"type":""}`
	}

	if !gjson.ParseBytes(in).IsObject() {
		// string: auto | none | required
		// gjson.ParseBytes(in).String() returns the string without quotes
		switch gjson.ParseBytes(in).String() {
		case "auto":
			result, _ := sjson.Set(base, "type", "auto")
			return []byte(result)
		case "required":
			result, _ := sjson.Set(base, "type", "any")
			return []byte(result)
		case "none":
			result, _ := sjson.Set(base, "type", "none")
			return []byte(result)
		}
		return nil
	}

	switch getType(in) {
	case "function":
		// ToolChoiceFunction -> Claude "tool"
		name := gjson.GetBytes(in, "name").String()
		if name != "" {
			result, _ := sjson.Set(base, "type", "tool")
			result, _ = sjson.Set(result, "name", name)
			return []byte(result)
		}
	}
	// Other types (allowed_tools, custom, mcp, file_search, etc.) are not supported in Claude
	return nil
}

// OpenAIChoiceToResponsesChoice converts OpenAI Chat Completions API tool_choice to Responses API tool_choice
// Parameter:
//   - in - The OpenAI Chat Completions API tool_choice
//
// Returns:
//   - The Responses API tool_choice
//
// Note:
//   - string values (auto, none, required) are passed through as-is
//   - ChatCompletionAllowedToolChoice -> ToolChoiceAllowed
//   - ChatCompletionNamedToolChoice -> ToolChoiceFunction
//   - ChatCompletionNamedToolChoiceCustom -> ToolChoiceCustom
func OpenAIChoiceToResponsesChoice(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	if !gjson.ParseBytes(in).IsObject() {
		// string: auto | none | required
		return in
	}

	typeValue := getType(in)
	switch typeValue {
	case "allowed_tools":
		// ChatCompletionAllowedToolChoice -> ToolChoiceAllowed
		mode := gjson.GetBytes(in, "allowed_tools.mode").String()
		tools := gjson.GetBytes(in, "allowed_tools.tools").Raw
		if mode != "" {
			result := `{"type":"allowed_tools","mode":""}`
			result, _ = sjson.Set(result, "mode", mode)
			if tools != "" {
				result, _ = sjson.SetRaw(result, "tools", tools)
			}
			return []byte(result)
		}
	case "function":
		// ChatCompletionNamedToolChoice -> ToolChoiceFunction
		name := gjson.GetBytes(in, "function.name").String()
		if name != "" {
			result := `{"type":"function","name":""}`
			result, _ = sjson.Set(result, "name", name)
			return []byte(result)
		}
	case "custom":
		// ChatCompletionNamedToolChoiceCustom -> ToolChoiceCustom
		name := gjson.GetBytes(in, "custom.name").String()
		if name != "" {
			result := `{"type":"custom","name":""}`
			result, _ = sjson.Set(result, "name", name)
			return []byte(result)
		}
	}
	return nil
}

// ResponsesChoiceToOpenAIChoice converts Responses API tool_choice to OpenAI Chat Completions API tool_choice
// Parameter:
//   - in - The Responses API tool_choice
//
// Returns:
//   - The OpenAI Chat Completions API tool_choice
//
// Note:
//   - string values (auto, none, required) are passed through as-is
//   - ToolChoiceAllowed -> ChatCompletionAllowedToolChoice
//   - ToolChoiceFunction -> ChatCompletionNamedToolChoice
//   - ToolChoiceCustom -> ChatCompletionNamedToolChoiceCustom
//   - ToolChoiceTypes and ToolChoiceMcp are not supported in Chat Completions API and return nil
func ResponsesChoiceToOpenAIChoice(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	if !gjson.ParseBytes(in).IsObject() {
		// string: auto | none | required
		return in
	}

	typeValue := getType(in)
	switch typeValue {
	case "allowed_tools":
		// ToolChoiceAllowed -> ChatCompletionAllowedToolChoice
		mode := gjson.GetBytes(in, "mode").String()
		tools := gjson.GetBytes(in, "tools").Raw
		if mode != "" {
			result := `{"type":"allowed_tools","allowed_tools":{"mode":""}}`
			result, _ = sjson.Set(result, "allowed_tools.mode", mode)
			if tools != "" {
				result, _ = sjson.SetRaw(result, "allowed_tools.tools", tools)
			}
			return []byte(result)
		}
	case "function":
		// ToolChoiceFunction -> ChatCompletionNamedToolChoice
		name := gjson.GetBytes(in, "name").String()
		if name != "" {
			result := `{"type":"function","function":{"name":""}}`
			result, _ = sjson.Set(result, "function.name", name)
			return []byte(result)
		}
	case "custom":
		// ToolChoiceCustom -> ChatCompletionNamedToolChoiceCustom
		name := gjson.GetBytes(in, "name").String()
		if name != "" {
			result := `{"type":"custom","custom":{"name":""}}`
			result, _ = sjson.Set(result, "custom.name", name)
			return []byte(result)
		}
	}
	// ToolChoiceTypes (file_search, web_search_preview, etc.) and ToolChoiceMcp
	// are not supported in Chat Completions API
	return nil
}
