package helpers

import (
	"fmt"

	"github.com/RenaLio/tudou/pkg/provider/common"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"

	"github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// CreateRequestMessages(Input)

// Claude Messages:
// - string
// - TextBlockParam
// - ImageBlockParam
// - DocumentBlockParam
// - SearchResultBlockParam
// - ThinkingBlockParam
// - RedactedThinkingBlockParam
// - ToolUseBlockParam
// - ToolResultBlockParam
// - ServerToolUseBlockParam
// - WebSearchToolResultBlockParam
// - WebFetchToolResultBlockParam
// - CodeExecutionToolResultBlockParam
// - BashCodeExecutionToolResultBlockParam
// - TextEditorCodeExecutionToolResultBlockParam
// - ToolSearchToolResultBlockParam
// - ContainerUploadBlockParam

// OpenAIResponseInput: string | []object
// - string
// - EasyInputMessage
// - Message
// - ResponseOutputMessage
// - FileSearchCall
// - ComputerCall
// - ComputerCallOutput
// - WebSearchCall
// - FunctionCall
// - FunctionCallOutput
// - ToolSearchCall
// - ToolSearchOutput
// - Reasoning
// - Compaction
// - ImageGenerationCall
// - CodeInterpreterCall
// - LocalShellCall
// - LocalShellCallOutput
// - ShellCall
// - ShellCallOutput
// - ApplyPatchCall
// - ApplyPatchCallOutput
// - McpListTools
// - McpApprovalRequest
// - McpApprovalResponse
// - McpCall
// - CustomToolCallOutput
// - CustomToolCall
// - ItemReference

// OpenAIChatMessage
// - ChatCompletionDeveloperMessageParam
// - ChatCompletionSystemMessageParam
// - ChatCompletionUserMessageParam
// - ChatCompletionAssistantMessageParam
// - ChatCompletionToolMessageParam
// - ChatCompletionFunctionMessageParam

// ClaudeMessageToOpenAI converts Claude messages to OpenAI Chat Completions message format
// Parameters:
//   - systemMsg - The Claude system message (string or array of TextBlockParam)
//   - msgList - The list of Claude MessageParam (user/assistant messages with content blocks)
//
// Returns:
//   - []json.RawMessage - The list of OpenAI Chat Completions messages
//
// Note:
//   - System message can be string or array format
//   - Handles text, image, thinking, tool_use, tool_result content blocks
//   - Separates content blocks and tool calls into different messages
//   - Ignores unsupported content types (document, search_result, container_upload)
//   - Groups consecutive content blocks of the same type
func ClaudeMessageToOpenAI(systemMsg json.RawMessage, msgList []claude.MessageParam) []json.RawMessage {
	systemRole := "system"
	msgS := make([]json.RawMessage, 0)
	// Claude system
	if gjson.ParseBytes(systemMsg).Type == gjson.String {
		msg := oaimodel.ChatSystemMessage{Role: systemRole, Content: systemMsg}
		msgS = append(msgS, mustMarshal(msg))
	} else if gjson.ParseBytes(systemMsg).IsArray() {
		var list []json.RawMessage
		gjson.ParseBytes(systemMsg).ForEach(func(_, value gjson.Result) bool {
			text := value.Get("text").String()
			if text != "" {
				content := oaimodel.ChatContentPartText{Type: "text", Text: text}
				list = append(list, mustMarshal(content))
			}
			return true
		})
		listData, _ := json.Marshal(list)
		msg := oaimodel.ChatSystemMessage{Role: systemRole, Content: listData}
		msgS = append(msgS, mustMarshal(msg))
	}
	for _, item := range msgList {
		if gjson.ParseBytes(item.Content).Type == gjson.String {
			msg := oaimodel.ChatDeveloperMessage{Role: item.Role, Content: item.Content}
			msgS = append(msgS, mustMarshal(msg))
			continue
		}
		if !gjson.ParseBytes(item.Content).IsArray() {
			continue
		}
		// user text,tool_result
		// assistant text... ,tool_use
		msg := &oaimodel.ChatAssistantMessage{}
		contents := make([]json.RawMessage, 0)
		toolCalls := make([]oaimodel.ChatMessageToolCall, 0)
		gjson.ParseBytes(item.Content).ForEach(func(_, value gjson.Result) bool {
			switch getType(json.RawMessage(value.Raw)) {
			case "text":
				fallthrough
			case "image":
				fallthrough
			case "document":
				fallthrough
			case "redacted_thinking":
				fallthrough
			case "thinking":
				if len(toolCalls) > 0 {
					// 先结束上一个
					msg.ToolCalls = toolCalls
					msg.Role = item.Role
					msgS = append(msgS, mustMarshal(msg))
					toolCalls = make([]oaimodel.ChatMessageToolCall, 0)
					//contents = make([]json.RawMessage, 0)
				}
				content := ClaudeMediaContentToOpenAI(json.RawMessage(value.Raw))
				if content != nil {
					contents = append(contents, content)
				}
				//contents = append(contents, content)
			case "tool_use":
				fallthrough
			case "server_tool_use":
				if len(contents) > 0 {
					// 先结束上一个
					msg.Content = mustMarshal(contents)
					msg.Role = item.Role
					msgS = append(msgS, mustMarshal(msg))
					contents = make([]json.RawMessage, 0)
				}
				data := ClaudeToolUseToChatMessageToolCall(json.RawMessage(value.Raw))
				if data != nil {
					toolCalls = append(toolCalls, *data)
				}
			case "web_search_tool_result":
				fallthrough
			case "web_fetch_tool_result":
				fallthrough
			case "code_execution_tool_result":
				fallthrough
			case "bash_code_execution_tool_result":
				fallthrough
			case "text_editor_code_execution_tool_result":
				fallthrough
			case "tool_search_tool_result":
				fallthrough
			case "tool_result":
				data := ClaudeToolCallOutputToOpenAI(json.RawMessage(value.Raw))
				if data != nil {
					msgS = append(msgS, mustMarshal(data))
				}

			case "search_result":
			case "container_upload":
			}
			return true
		})
		if len(contents) > 0 {
			msg.Content = mustMarshal(contents)
			msg.Role = item.Role
			msgS = append(msgS, mustMarshal(msg))
		}
		if len(toolCalls) > 0 {
			msg.ToolCalls = toolCalls
			msg.Role = item.Role
			msgS = append(msgS, mustMarshal(msg))
		}
	}
	return msgS
}

func getRole(in json.RawMessage) string {
	return gjson.GetBytes(in, "role").String()
}

// OpenAIChatMessageToClaude converts OpenAI Chat Completions messages to Claude messages
// Parameters:
//   - msgS - The list of OpenAI Chat Completions messages (system/developer/user/assistant/tool)
//
// Returns:
//   - json.RawMessage - The Claude system message (string or array of TextBlockParam)
//   - []claude.MessageParam - The list of Claude MessageParam (user/assistant messages)
//
// Note:
//   - System/developer messages are combined into a single system message
//   - User messages with content parts are converted to Claude content blocks
//   - Assistant messages with tool_calls are converted to tool_use blocks
//   - Tool messages are converted to tool_result blocks
//   - Function messages (deprecated) are treated as tool messages
func OpenAIChatMessageToClaude(msgS []json.RawMessage) (json.RawMessage, []claude.MessageParam) {
	if msgS == nil || len(msgS) == 0 {
		return nil, nil
	}

	var systemParts []string
	var claudeMessages []claude.MessageParam

	for _, raw := range msgS {

		switch getRole(raw) {
		case "system", "developer":
			// Extract content from system/developer message
			var msg oaimodel.ChatSystemMessage
			if err := json.Unmarshal(raw, &msg); err == nil {
				content := openAIMessageContentToText(msg.Content)
				if content != nil {
					systemParts = append(systemParts, content...)
				}
			}

		case "user":
			// Convert user message to Claude format
			var msg oaimodel.ChatUserMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			content := openAIUserContentToClaude(msg.Content)
			if content != nil {
				claudeMessages = append(claudeMessages, claude.MessageParam{
					Role:    "user",
					Content: content,
				})
			}

		case "assistant":
			// Convert assistant message to Claude format
			var msg oaimodel.ChatAssistantMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			content := openAIAssistantContentToClaude(&msg)
			if content != nil {
				claudeMessages = append(claudeMessages, claude.MessageParam{
					Role:    "assistant",
					Content: content,
				})
			}

		case "tool":
			// Convert tool message to Claude tool_result
			var msg oaimodel.ChatToolMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			param := claude.MessageParam{
				Role:    "user",
				Content: nil,
			}
			toolContent := OpenAIToolMessageToClaudeToolCallOutput(&msg, false)
			if toolContent != nil {
				param.Content = mustMarshal([]json.RawMessage{toolContent})
				claudeMessages = append(claudeMessages, param)
			}
		}
	}

	// Build system message
	var systemMsg json.RawMessage
	if len(systemParts) == 1 {
		systemMsg = mustMarshal(systemParts[0])
		return systemMsg, claudeMessages
	}
	systemBlocks := make([]claude.TextBlockParam, 0)
	for _, part := range systemParts {
		systemBlocks = append(systemBlocks, claude.TextBlockParam{
			Type: "text",
			Text: part,
		})
	}

	return mustMarshal(systemBlocks), claudeMessages
}

// openAIMessageContentToText extracts text content from OpenAI message content
func openAIMessageContentToText(content json.RawMessage) []string {
	if content == nil {
		return nil
	}
	// If it's a plain string
	if gjson.ParseBytes(content).Type == gjson.String {
		return []string{gjson.ParseBytes(content).String()}
	}
	// If it's an array of content parts, extract text parts
	if gjson.ParseBytes(content).IsArray() {
		var texts []string
		gjson.ParseBytes(content).ForEach(func(_, value gjson.Result) bool {
			if value.Get("type").String() == "text" {
				texts = append(texts, value.Get("text").String())
			}
			return true
		})
		return texts
	}
	return nil
}

// openAIUserContentToClaude converts OpenAI user message content to Claude content blocks
func openAIUserContentToClaude(content json.RawMessage) json.RawMessage {
	if content == nil {
		return nil
	}

	// If it's a plain string
	if gjson.ParseBytes(content).Type == gjson.String {
		return content
	}

	// If it's an array of content parts
	if gjson.ParseBytes(content).IsArray() {
		var blocks []interface{}
		gjson.ParseBytes(content).ForEach(func(_, value gjson.Result) bool {
			partType := value.Get("type").String()
			switch partType {
			case "text":
				blocks = append(blocks, claude.TextBlockParam{
					Type: "text",
					Text: value.Get("text").String(),
				})
			case "image_url":
				// Convert image_url to Claude ImageBlockParam
				url := value.Get("image_url.url").String()
				if url != "" {
					blocks = append(blocks, claude.ImageBlockParam{
						Type: "image",
						Source: claude.ImageSourceParam{
							Type: "url",
							Url:  url,
						},
					})
				}
			case "input_audio":
				// Audio is not directly supported in Claude messages, skip
			case "file":
				// File is not directly supported in Claude messages, skip
			}
			return true
		})
		if len(blocks) > 0 {
			return mustMarshal(blocks)
		}
	}

	return nil
}

// openAIAssistantContentToClaude converts OpenAI assistant message to Claude content blocks
func openAIAssistantContentToClaude(msg *oaimodel.ChatAssistantMessage) json.RawMessage {
	if msg == nil {
		return nil
	}
	var blocks []any

	// Handle text content
	if msg.Content != nil {
		content := openAIMessageContentToText(msg.Content)
		if content != nil {
			for _, text := range content {
				blocks = append(blocks, claude.TextBlockParam{
					Type: "text",
					Text: text,
				})
			}
		}
	}

	// Handle refusal
	if msg.Refusal != "" {
		blocks = append(blocks, claude.TextBlockParam{
			Type: "text",
			Text: "Refusal: " + msg.Refusal,
		})
	}

	// Handle tool_calls - convert to tool_use blocks
	for _, tc := range msg.ToolCalls {
		data := ChatMessageToolCallToClaudeToolUse(&tc)
		if data != nil {
			blocks = append(blocks, data)
		}
	}

	if len(blocks) > 0 {
		return mustMarshal(blocks)
	}
	return nil
}

// openAIToolMessageToClaude converts OpenAI tool message to Claude tool_result block
func openAIToolMessageToClaude(msg *oaimodel.ChatToolMessage) json.RawMessage {
	if msg == nil || msg.ToolCallId == "" {
		return nil
	}

	// Normalize content
	var content json.RawMessage
	if msg.Content != nil {
		if gjson.ParseBytes(msg.Content).Type == gjson.String {
			content = msg.Content
		} else {
			// Try to extract text from content parts
			content = msg.Content
		}
	}

	blocks := []claude.ToolResultBlockParam{
		{
			Type:      "tool_result",
			ToolUseId: msg.ToolCallId,
			Content:   content,
			IsError:   false,
		},
	}
	return mustMarshal(blocks)
}

// openAIFunctionMessageToClaude converts deprecated OpenAI function message to Claude tool_result
func openAIFunctionMessageToClaude(msg *oaimodel.ChatFunctionMessage) json.RawMessage {
	if msg == nil {
		return nil
	}

	// Extract function name from message or use "unknown"
	funcName := msg.Name
	if funcName == "" {
		funcName = "unknown"
	}

	blocks := []claude.ToolResultBlockParam{
		{
			Type:      "tool_result",
			ToolUseId: "function_" + funcName,
			Content:   msg.Content,
			IsError:   false,
		},
	}
	return mustMarshal(blocks)
}

func ResponseInputToOpenAIMessage(systemMsg string, in json.RawMessage) []json.RawMessage {
	if in == nil {
		return nil
	}
	var messages []json.RawMessage
	// Handle system message
	if len(systemMsg) > 0 {
		strBytes, _ := common.MarshalJSON(systemMsg)
		developerMsg := oaimodel.ChatDeveloperMessage{
			Role:    "system",
			Content: strBytes,
		}
		messages = append(messages, mustMarshal(developerMsg))
	}
	// handle input
	if gjson.ParseBytes(in).Type == gjson.String {
		msg := oaimodel.ChatSystemMessage{Role: "user", Content: in}
		messages = append(messages, mustMarshal(msg))
		return messages
	}
	if gjson.ParseBytes(in).IsArray() {
		gjson.ParseBytes(in).ForEach(func(_, value gjson.Result) bool {
			// todo
			if value.Type != gjson.String {
				return true
			}
			fmt.Println("getType", getType(json.RawMessage(value.Raw)))
			switch getType(json.RawMessage(value.Raw)) {
			case "reasoning":
				fallthrough
			case "message":
				data := ResponseMediaInputToOpenAI(json.RawMessage(value.Raw))
				if data != nil {
					messages = append(messages, data)
				}
			case "file_search_call":
				fallthrough
			case "computer_call":
				fallthrough
			case "web_search_call":
				fallthrough
			case "tool_search_call":
				fallthrough
			case "image_generation_call":
				fallthrough
			case "code_interpreter_call":
				fallthrough
			case "local_shell_call":
				fallthrough
			case "shell_call":
				fallthrough
			case "apply_patch_call":
				fallthrough
			case "mcp_call":
				fallthrough
			case "function_call":
				data := ResponseToolCallToChatMessageToolCall(json.RawMessage(value.Raw))
				if data != nil {
					msg := oaimodel.ChatAssistantMessage{
						Role:      "assistant",
						ToolCalls: []oaimodel.ChatMessageToolCall{*data},
					}
					messages = append(messages, mustMarshal(msg))
				}
			case "computer_call_output":
				fallthrough
			case "tool_search_output":
				fallthrough
			case "local_shell_call_output":
				fallthrough
			case "shell_call_output":
				fallthrough
			case "apply_patch_call_output":
				fallthrough
			case "custom_tool_call_output":
				fallthrough
			case "custom_tool_call":
				fallthrough
			case "function_call_output":
				data := ResponseToolCallOutputToOpenAI(json.RawMessage(value.Raw))
				if data != nil {
					messages = append(messages, mustMarshal(data))
				}
			case "mcp_list_tools":
			case "mcp_approval_request":
			case "mcp_approval_response":
			case "compaction":
			case "item_reference":
			}
			return true
		})
	}
	return messages
}
