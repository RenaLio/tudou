package helpers

import (
	"github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ClaudeTools: from https://platform.claude.com/docs/en/api/messages/create
// - custom tool(type maybe "")
// - bash(bash_20250124)
// - code_execution(code_execution_20250522,code_execution_20250825,code_execution_20260120)
// - memory(memory_20250818)
// - text_editor(text_editor_20250124,text_editor_20250429,text_editor_20250728)
// - web_search/web_fetch(web_search_20250305,web_fetch_20250910,web_search_20260209,web_fetch_20260209,web_fetch_20260309)
// - tool_search(tool_search_tool_bm25,tool_search_tool_regex)

// OpenAITools(chat completions tools): from https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
// - function tool
// - custom tool

// OpenAIResponsesTool: from https://developers.openai.com/api/reference/resources/responses/methods/create
// - function
// - custom
// - file_search
// - computer
// - computer_use_preview
// - web_search
// - mcp
// - code_interpreter
// - image_generation
// - local_shell
// - shell
// - namespace
// - tool_search
// - web_search_preview
// - apply_patch

// ClaudeToolToOpenAITool converts a Claude tool to an OpenAI tool.
// Parameter:
//   - tool - The Claude tool to convert.(claude.Tool)
//
// Returns:
//   - The OpenAI tool(openai.ChatTool).
//
// Note:
//   - supported type: claude custom tool
func ClaudeToolToOpenAITool(in json.RawMessage) *openai.ChatTool {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "custom", "":
		cTool := new(claude.Tool)
		_ = json.Unmarshal(in, cTool)
		oTool := openai.ChatTool{
			Type: "function",
			Function: &openai.Function{
				Type:         "function",
				Name:         cTool.Name,
				Description:  cTool.Description,
				Parameters:   cTool.InputSchema,
				Strict:       cTool.Strict,
				DeferLoading: cTool.DeferLoading,
			},
		}
		return &oTool
	}
	return nil
}

func OpenAIToolToClaudeTool(in *openai.ChatTool) json.RawMessage {
	if in == nil {
		return nil
	}
	switch in.Type {
	case "function":
		oTool := in
		cTool := claude.Tool{
			Name:         oTool.Function.Name,
			Type:         "custom",
			InputSchema:  oTool.Function.Parameters,
			DeferLoading: oTool.Function.DeferLoading,
			Description:  oTool.Function.Description,
			Strict:       oTool.Function.Strict,
		}
		out, _ := json.Marshal(cTool)
		return out
	}
	return nil
}

// ClaudeToolToResponsesTool converts a Claude tool to a Responses tool.
//
// Parameter:
//   - tool - The Claude tool to convert.(claude.Tool)
//
// Returns:
//   - The Responses tool(openai.Function).
//
// Note:
//   - supported type: claude custom tool
//   - maybe support type(it depends on tool_call_out(tool_result)):
//     # bash -> shell
//     # code_execution -> code_interpreter
//     # web_search/web_fetch -> web_search
//     # tool_search -> tool_search
func ClaudeToolToResponsesTool(in json.RawMessage) json.RawMessage {
	// function
	// maybe support(it depends on tool_call_out(tool_result))
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "custom", "":
		cTool := new(claude.Tool)
		_ = json.Unmarshal(in, cTool)
		oTool := &openai.Function{
			Type:         "function",
			Name:         cTool.Name,
			Description:  cTool.Description,
			Parameters:   cTool.InputSchema,
			Strict:       cTool.Strict,
			DeferLoading: cTool.DeferLoading,
		}
		out, _ := json.Marshal(oTool)
		return out
	}
	return nil
}

func ResponsesToolToClaudeTool(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "function":
		oTool := new(openai.Function)
		_ = json.Unmarshal(in, oTool)
		cTool := claude.Tool{
			Name:         oTool.Name,
			Type:         "custom",
			InputSchema:  oTool.Parameters,
			DeferLoading: oTool.DeferLoading,
			Description:  oTool.Description,
			Strict:       oTool.Strict,
		}
		out, _ := json.Marshal(cTool)
		return out
	}
	return nil
}

// ResponsesToolToOpenAITool converts a Responses tool to an OpenAI tool.
// Parameter:
//   - tool - The Responses tool to convert.(e.g. openai.Function,openai.Custom)
//
// Returns:
//   - The OpenAI tool(openai.ChatTool).
//
// Note:
//   - supported type: response function tool, response custom tool
func ResponsesToolToOpenAITool(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "function":
		// `openai.Function` -> `openai.ChatTool`
		out := `{"type":"function","function":{}}` // openai.ChatTool
		result, err := sjson.SetRaw(out, "function", string(in))
		if err != nil {
			return nil
		}
		return []byte(result)
	case "custom":
		out := `{"type":"custom","custom":{}}` // openai.ChatTool
		result, err := sjson.SetRaw(out, "custom", string(in))
		if err != nil {
			return nil
		}
		return []byte(result)
	}
	return nil
}

func ResponsesToolToOpenAITool2(in json.RawMessage) *openai.ChatTool {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "function":
		var functionTool openai.FunctionTool
		if err := json.Unmarshal(in, &functionTool); err != nil {
			return nil
		}
		tool := openai.ChatTool{
			Type:     "function",
			Function: &functionTool,
		}
		return &tool
	case "custom":
		var customTool openai.CustomTool
		if err := json.Unmarshal(in, &customTool); err != nil {
			return nil
		}
		tool := openai.ChatTool{
			Type:   "custom",
			Custom: &customTool,
		}
		return &tool
	}
	return nil
}

func OpenAIToolToResponsesTool(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "function":
		modified, err := sjson.SetBytes(in, "function.type", "function")
		if err != nil {
			return nil
		}
		result := gjson.GetBytes(modified, "function")
		if !result.Exists() {
			return nil
		}
		return json.RawMessage(result.Raw)
	case "custom":
		modified, err := sjson.SetBytes(in, "custom.type", "custom")
		if err != nil {
			return nil
		}
		result := gjson.GetBytes(modified, "custom")
		if !result.Exists() {
			return nil
		}
		return json.RawMessage(result.Raw)
	}
	return nil
}
