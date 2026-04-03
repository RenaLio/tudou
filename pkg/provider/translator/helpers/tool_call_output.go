package helpers

import (
	"fmt"

	"github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// ToolCallOutput is the output for a tool call.

// Claude:
// - ToolResultBlockParam
// - WebSearchToolResultBlockParam
// - WebFetchToolResultBlockParam
// - CodeExecutionToolResultBlockParam
// - BashCodeExecutionToolResultBlockParam
// - TextEditorCodeExecutionToolResultBlockParam
// - ToolSearchToolResultBlockParam

// Responses
// - FunctionCallOutput
// - CustomToolCallOutput
// - ComputerCallOutput
// - ToolSearchOutput
// - LocalShellCallOutput
// - ShellCallOutput
// - ApplyPatchCallOutput

// OpenAI(chat completion):
// - ChatCompletionToolMessageParam
// - ChatCompletionFunctionMessageParam - Deprecated

// ClaudeToolCallOutputToOpenAI converts Claude tool result block to OpenAI Chat Completions tool message
// Parameter:
//   - in - The JSON raw message of ToolResultBlockParam from Claude API
//
// Returns:
//   - The ChatToolMessage for Chat Completions API
//
// Note:
//   - Supports "tool_result" type only
//   - Uses gjson to extract fields directly without full struct unmarshaling for better performance
//   - Returns nil for unsupported types, nil input, or unsupported content types
//   - Handles is_error flag by wrapping content in error object
//   - Filters out unsupported content types: image, search_result, document, tool_reference
func ClaudeToolCallOutputToOpenAI(in json.RawMessage) *openai.ChatToolMessage {
	if in == nil {
		return nil
	}

	switch getType(in) {
	case "tool_result":
		// Use gjson to extract fields directly
		toolUseId := gjson.GetBytes(in, "tool_use_id").String()
		contentResult := gjson.GetBytes(in, "content")
		isError := gjson.GetBytes(in, "is_error").Bool()

		if toolUseId == "" {
			return nil
		}

		// Initialize response
		resp := &openai.ChatToolMessage{
			Role:       "tool",
			ToolCallId: toolUseId,
		}
		// Handle error case first
		if isError {
			errDetail := fmt.Sprintf("error:%s", contentResult.Raw)
			resp.Content = mustMarshal(errDetail)
			return resp
		}

		// Handle string content directly
		if contentResult.Type == gjson.String {
			resp.Content = json.RawMessage(contentResult.Raw)
			return resp
		}

		// Handle array content - iterate and filter unsupported types
		// Claude content can be an array of content blocks
		if contentResult.IsArray() {
			var validContents []json.RawMessage
			contentResult.ForEach(func(_, item gjson.Result) bool {
				itemType := item.Get("type").String()
				switch itemType {
				case "image", "search_result", "document", "tool_reference":
					// Skip unsupported content types
					return true
				}
				validContents = append(validContents, json.RawMessage(item.Raw))
				return true
			})
			if len(validContents) == 0 {
				return nil
			}
			content, _ := json.Marshal(validContents)
			resp.Content = content
			return resp
		}
	}
	return nil
}

// OpenAIToolMessageToClaudeToolCallOutput converts OpenAI Chat Completions tool message to Claude tool result block
// Parameter:
//   - in - The ChatToolMessage from Chat Completions API
//   - isError - Whether this is an error result
//
// Returns:
//   - json.RawMessage - The ToolResultBlockParam JSON for Claude API
//
// Note:
//   - Converts OpenAI tool message to Claude's "tool_result" type
//   - Content must be either a plain string or a JSON array
//   - Returns nil for nil input, empty tool_call_id, or invalid content type
//   - Sets is_error flag based on parameter
func OpenAIToolMessageToClaudeToolCallOutput(in *openai.ChatToolMessage, isError bool) json.RawMessage {
	if in == nil || in.ToolCallId == "" {
		return nil
	}

	// Build Claude ToolResultBlockParam
	block := claude.ToolResultBlockParam{
		Type:      "tool_result",
		ToolUseId: in.ToolCallId,
		IsError:   isError,
	}
	// string | []typeTextObj
	block.Content = in.Content

	data, _ := json.Marshal(block)
	return data
}

// ResponseToolCallOutputToClaude converts Responses API tool call output to Claude tool result block
// Parameter:
//   - in - The JSON raw message of FunctionCallOutput or ResponseCustomToolCallOutput from Responses API
//
// Returns:
//   - json.RawMessage - The ToolResultBlockParam JSON for Claude API
//
// Note:
//   - Supports "function_call_output" and "custom_tool_call_output" types
//   - Maps call_id to tool_use_id
//   - Output can be either string or array of objects
//   - For array output, performs type conversion:
//   - "input_text" -> "text"
//   - "input_image" -> "image" (with URL from image_url)
//   - "input_file" -> skipped (not supported by Claude tool result)
//   - Returns nil for unsupported types or nil input
func ResponseToolCallOutputToClaude(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}

	switch getType(in) {
	case "function_call_output", "custom_tool_call_output":
		// Use gjson to extract fields directly
		callId := gjson.GetBytes(in, "call_id").String()
		outputResult := gjson.GetBytes(in, "output")

		if callId == "" {
			return nil
		}

		// Build Claude ToolResultBlockParam using map to avoid double serialization
		result := map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": callId,
			"is_error":    false,
		}

		// Handle string output directly
		if outputResult.Type == gjson.String {
			result["content"] = outputResult.String()
			data, _ := json.Marshal(result)
			return data
		}

		// Handle array output with type conversion
		if outputResult.IsArray() {
			var validContents []json.RawMessage
			outputResult.ForEach(func(_, item gjson.Result) bool {
				itemType := item.Get("type").String()
				switch itemType {
				case "input_text":
					// Convert input_text to text type
					textContent := item.Get("text").String()
					textBlock := `{"type":"text","text":"` + textContent + `"}`
					validContents = append(validContents, json.RawMessage(textBlock))
				case "input_image":
					// Convert input_image to image type with URL
					url := item.Get("image_url").String()
					if url == "" {
						return true
					}
					imageBlock := `{"type":"image","source":{"type":"url","url":"` + url + `"}}`
					validContents = append(validContents, json.RawMessage(imageBlock))
				case "input_file":
					// Skip input_file - not supported in Claude tool result
				default:
					// Keep other types as-is
					validContents = append(validContents, json.RawMessage(item.Raw))
				}
				return true
			})
			result["content"] = validContents
			data, _ := json.Marshal(result)
			return data
		}

		// Output must be either string or array
		// Other types are not supported
		return nil
	}
	return nil
}

// ClaudeToolCallOutputToResponse converts Claude tool result block to Responses API tool call output
// Parameter:
//   - in - The JSON raw message of ToolResultBlockParam from Claude API
//   - outputType - The output type: "function_call_output" or "custom_tool_call_output"
//
// Returns:
//   - json.RawMessage - The FunctionCallOutput or ResponseCustomToolCallOutput JSON for Responses API
//
// Note:
//   - Supports "tool_result" type only
//   - Maps tool_use_id to call_id
//   - Content can be either string or array of objects
//   - For array content, performs type conversion:
//   - "text" -> "input_text"
//   - "image" -> "input_image" (extracts URL to image_url)
//   - Returns nil for unsupported types, nil input, or invalid content type
func ClaudeToolCallOutputToResponse(in json.RawMessage, outputType string) json.RawMessage {
	if in == nil {
		return nil
	}

	if getType(in) != "tool_result" {
		return nil
	}

	if outputType == "" {
		outputType = "function_call_output"
	}

	// Use gjson to extract fields directly
	toolUseId := gjson.GetBytes(in, "tool_use_id").String()
	contentResult := gjson.GetBytes(in, "content")
	isError := gjson.GetBytes(in, "is_error").Bool()

	if toolUseId == "" {
		return nil
	}

	// Build output based on type
	var output json.RawMessage

	// Handle string content directly
	if contentResult.Type == gjson.String {
		output = json.RawMessage(contentResult.Raw)
	} else if contentResult.IsArray() {
		// Handle array content with type conversion
		var validContents []map[string]interface{}
		contentResult.ForEach(func(_, item gjson.Result) bool {
			itemType := item.Get("type").String()
			switch itemType {
			case "text":
				// Convert text to input_text
				content := map[string]interface{}{
					"type": "input_text",
				}
				if text := item.Get("text").String(); text != "" {
					content["text"] = text
				}
				validContents = append(validContents, content)
			case "image":
				// Convert image to input_image
				url := item.Get("source.url").String()
				if url != "" {
					content := map[string]interface{}{
						"type":      "input_image",
						"image_url": url,
					}
					validContents = append(validContents, content)
				}
			default:
				// Skip other types
			}
			return true
		})
		output, _ = json.Marshal(validContents)
	} else {
		// Content must be either string or array
		return nil
	}

	// Handle error case - wrap output in error object
	callStatus := "completed"
	if isError {
		errorOutput, _ := json.Marshal(fmt.Sprintf("error:%s", string(output)))
		output = errorOutput
		callStatus = "incomplete"
	}

	// Build response based on output type
	switch outputType {
	case "function_call_output":
		resp := openai.FunctionCallOutput{
			Type:   "function_call_output",
			CallId: toolUseId,
			//Id:     toolUseId,
			Output: output,
			Status: callStatus,
		}
		data, _ := json.Marshal(resp)
		return data
	case "custom_tool_call_output":
		resp := openai.ResponseCustomToolCallOutput{
			Type:   "custom_tool_call_output",
			CallId: toolUseId,
			//ID:     toolUseId,
			Output: output,
		}
		data, _ := json.Marshal(resp)
		return data
	}
	return nil
}

// ResponseToolCallOutputToOpenAI converts Responses API tool call output to OpenAI Chat Completions tool message
// Parameter:
//   - in - The JSON raw message of FunctionCallOutput or ResponseCustomToolCallOutput from Responses API
//
// Returns:
//   - The ChatToolMessage for Chat Completions API
//
// Note:
//   - Supports "function_call_output" and "custom_tool_call_output" types
//   - Uses gjson to extract fields directly without full struct unmarshaling for better performance
//   - Returns nil for unsupported types or nil input
//   - ResponseInputImageContent and ResponseInputFileContent are not handled (OpenAI completion doesn't support them)
func ResponseToolCallOutputToOpenAI(in json.RawMessage) *openai.ChatToolMessage {
	if in == nil {
		return nil
	}

	switch getType(in) {
	case "function_call_output", "custom_tool_call_output":
		// Use gjson to extract fields directly
		callId := gjson.GetBytes(in, "call_id").String()
		outputResult := gjson.GetBytes(in, "output")
		var output json.RawMessage
		// If output is a string, extract the string value
		if outputResult.Type == gjson.String {
			output = json.RawMessage(outputResult.Raw)
		} else {
			output = json.RawMessage(outputResult.Raw)
		}
		if callId == "" {
			return nil
		}
		return &openai.ChatToolMessage{
			Role:       "tool",
			ToolCallId: callId,
			// ResponseInputImageContent or ResponseInputFileContent not handled
			Content: output, // string | object
		}
	}
	return nil
}

// OpenAIToolMessageToResponseToolCallOutput converts OpenAI Chat Completions tool message to Responses API tool call output
// Parameter:
//   - in - The ChatToolMessage from Chat Completions API
//   - outputType - The output type: "function_call_output" or "custom_tool_call_output"
//
// Returns:
//   - json.RawMessage - The FunctionCallOutput or ResponseCustomToolCallOutput JSON for Responses API
//
// Note:
//   - Supports converting tool messages to both "function_call_output" and "custom_tool_call_output" types
//   - The outputType parameter determines the target type
//   - Returns nil for nil input or unsupported outputType
func OpenAIToolMessageToResponseToolCallOutput(in *openai.ChatToolMessage, outputType string) json.RawMessage {
	if in == nil || in.ToolCallId == "" {
		return nil
	}

	switch outputType {
	case "function_call_output":
		resp := openai.FunctionCallOutput{
			Type:   "function_call_output",
			CallId: in.ToolCallId,
			Id:     in.ToolCallId,
			Output: in.Content,
		}
		data, _ := json.Marshal(resp)
		return data
	case "custom_tool_call_output":
		resp := openai.ResponseCustomToolCallOutput{
			Type:   "custom_tool_call_output",
			CallId: in.ToolCallId,
			ID:     in.ToolCallId,
			Output: in.Content,
		}
		data, _ := json.Marshal(resp)
		return data
	}
	return nil
}
