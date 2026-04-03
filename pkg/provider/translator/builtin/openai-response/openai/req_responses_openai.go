package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

func ConvertResponseRequestToChatCompletion(_ context.Context, req *types.Request) (*types.Request, error) {
	var err error
	req = common.CloneRequest(req)
	params := new(openai.CreateResponseRequest)
	if err := common.UnmarshalJSON(req.Payload, params); err != nil {
		return nil, err
	}

	// 一些通用参数
	out, _ := NormalizeCommonParams(params)

	// reasoning_effort、response_format、verbosity、stream_option
	if params.Reasoning != nil {
		out.ReasoningEffort = params.Reasoning.Effort
	}
	if params.Text != nil {
		out.Verbosity = params.Text.Verbosity
		if params.Text.Format != nil {
			out.ResponseFormat = &openai.ChatResponseFormat{
				Type: params.Text.Format.Type,
			}
		}
		if params.Text.Format != nil && params.Text.Format.Type == "json_schema" {
			jsonSchema := &openai.JSONSchema{
				Name:        params.Text.Format.Name,
				Description: params.Text.Format.Description,
				Schema:      params.Text.Format.Schema,
				Strict:      params.Text.Format.Strict,
			}
			out.ResponseFormat.JsonSchema = jsonSchema
		}

	}
	out.StreamOptions = &openai.ChatStreamOptions{
		IncludeUsage: true,
	}
	if params.Stream != nil && params.StreamOptions != nil {
		if params.StreamOptions.IncludeObfuscation != nil {
			out.StreamOptions.IncludeObfuscation = *params.StreamOptions.IncludeObfuscation
		}
	}

	// 系统提示词
	var messages []json.RawMessage
	if len(params.Instructions) > 0 {
		strBytes, _ := common.MarshalJSON(params.Instructions)
		developerMsg := openai.ChatDeveloperMessage{
			Role:    "system",
			Content: strBytes,
		}
		msgBytes, _ := common.MarshalJSON(developerMsg)
		messages = append(messages, msgBytes)
	}
	// 消息
	chatMessage, err := ResponseInputItemsToChatMessage(params)
	if err != nil {
		fmt.Println("用户输入参数设置失败")
		fmt.Println(err.Error())
		return nil, err
	}
	messages = append(messages, chatMessage...)
	out.Messages = messages
	// tool
	out.Tools = ResponseToolsToChatTools(params.Tools)
	// tool_choice
	out.ToolChoice = helpers.ResponsesChoiceToOpenAIChoice(params.ToolChoice)
	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	fmt.Println("origin req:", string(req.Payload))
	req.Payload = data
	fmt.Println("transd req:", string(req.Payload))
	return req, nil
}

func ConvertResponseRequestToChatCompletion2(_ context.Context, req *types.Request) (*types.Request, error) {
	var err error
	req = common.CloneRequest(req)
	params := new(openai.CreateResponseRequest)
	if err := common.UnmarshalJSON(req.Payload, params); err != nil {
		return nil, err
	}
	fmt.Println("req:", string(req.Payload))

	// 一些通用参数
	out, _ := NormalizeCommonParams(params)

	// reasoning_effort、response_format、verbosity、stream_option
	if params.Reasoning != nil {
		out.ReasoningEffort = params.Reasoning.Effort
	}
	if params.Text != nil {
		out.Verbosity = params.Text.Verbosity
		if params.Text.Format != nil {
			out.ResponseFormat = &openai.ChatResponseFormat{
				Type: params.Text.Format.Type,
			}
		}
		if params.Text.Format != nil && params.Text.Format.Type == "json_schema" {
			jsonSchema := &openai.JSONSchema{
				Name:        params.Text.Format.Name,
				Description: params.Text.Format.Description,
				Schema:      params.Text.Format.Schema,
				Strict:      params.Text.Format.Strict,
			}
			out.ResponseFormat.JsonSchema = jsonSchema
		}

	}
	out.StreamOptions = &openai.ChatStreamOptions{
		IncludeUsage: true,
	}
	if params.Stream != nil && params.StreamOptions != nil {
		if params.StreamOptions.IncludeObfuscation != nil {
			out.StreamOptions.IncludeObfuscation = *params.StreamOptions.IncludeObfuscation
		}
	}

	// 系统提示词
	// 消息
	messages := helpers.ResponseInputToOpenAIMessage(params.Instructions, params.Input)
	out.Messages = messages
	// tool
	tools := make([]openai.ChatTool, 0, len(params.Tools))
	for _, v := range params.Tools {
		tool := helpers.ResponsesToolToOpenAITool2(v)
		if tool != nil {
			tools = append(tools, *tool)
		}
	}
	out.Tools = tools
	// tool_choice
	out.ToolChoice, _ = ResponseToolChoiceToChatToolChoice(params.ToolChoice)
	data, err := common.MarshalJSON(out)
	if err != nil {
		return nil, err
	}
	req.Payload = data
	fmt.Println("trans:", string(req.Payload))
	return req, nil
}

func NormalizeCommonParams(params *openai.CreateResponseRequest) (*openai.CreateChatCompletionRequest, error) {
	var err error
	out := &openai.CreateChatCompletionRequest{
		Model:  params.Model,
		Stream: params.Stream,

		MaxTokens:           params.MaxOutputTokens,
		MaxCompletionTokens: params.MaxOutputTokens,

		Metadata:             params.Metadata,
		ParallelToolCalls:    params.ParallelToolCalls,
		ServiceTier:          params.ServiceTier,
		Store:                params.Store,
		Temperature:          params.Temperature,
		ToolChoice:           params.ToolChoice,
		TopLogprobs:          params.TopLogprobs,
		TopP:                 params.TopP,
		PromptCacheRetention: params.PromptCacheRetention,
		PromptCacheKey:       params.PromptCacheKey,
		SafetyIdentifier:     params.SafetyIdentifier,
		User:                 params.User,
	}
	return out, err
}

func ResponseInputItemsToChatMessage(params *openai.CreateResponseRequest) ([]json.RawMessage, error) {
	if len(params.Input) <= 0 {
		return nil, fmt.Errorf("ResponseInput is empty")
	}
	var err error
	// string input
	var inputText string
	if err = common.UnmarshalJSON(params.Input, &inputText); err == nil {
		userMsg := openai.ChatUserMessage{
			Role:    "user",
			Content: params.Input,
		}
		jsonMsg, _ := common.MarshalJSON(userMsg)
		return []json.RawMessage{jsonMsg}, err
	}
	var messages []json.RawMessage
	// array input
	// supported type: message、function_call_out、function_call、custom_tool_xx
	// unsported item type:
	//	- file_search_xx
	//	- computer_xx
	//	- web_search_xx
	//	- reasoning
	//	- compaction
	//	- image_generation_call
	//	- code_interpreter_call
	//	- shell_xx
	//	- apply_patch_xx
	//	- mcp_xx
	//	- reference_xx
	var inputItems []json.RawMessage
	if err = json.Unmarshal(params.Input, &inputItems); err != nil {
		return messages, fmt.Errorf("unmarshal input items failed: %w", err)
	}
	for _, inputItem := range inputItems {
		itemType := "message"
		if gjson.GetBytes(inputItem, "type").String() != "" {
			itemType = gjson.GetBytes(inputItem, "type").String()
		}
		//itemType = strings.Trim(itemType, "\"")

		switch itemType {
		case "message":
			var messageInputItem openai.Message
			if err = json.Unmarshal(inputItem, &messageInputItem); err != nil {
				return messages, fmt.Errorf("unmarshal input message item failed: %w", err)
			}
			message, err := ResponseMessageToChatMessage(&messageInputItem)
			if err != nil {
				return messages, fmt.Errorf("convert input message failed: %w", err)
			}
			messages = append(messages, message)
		case "function_call":
			var functionToolCallItem openai.ResponseFunctionCall
			if err = json.Unmarshal(inputItem, &functionToolCallItem); err != nil {
				return messages, fmt.Errorf("unmarshal input message failed: %w", err)
			}
			message := ResponseFunctionToolCallToChatMessage(&functionToolCallItem)
			messages = append(messages, message)
		case "custom_tool_call":
			var customToolCallItem openai.ResponseCustomToolCall
			if err = json.Unmarshal(inputItem, &customToolCallItem); err != nil {
				return messages, fmt.Errorf("unmarshal input message failed: %w", err)
			}
			message := ResponseCustomToolCallToChatMessage(&customToolCallItem)
			messages = append(messages, message)
		case "function_call_output":
			var functionCallOutItem openai.FunctionCallOutput
			if err = json.Unmarshal(inputItem, &functionCallOutItem); err != nil {
				return messages, fmt.Errorf("unmarshal input message failed: %w", err)
			}
			message := ResponseFunctionCallOutputToChatMessage(&functionCallOutItem)
			messages = append(messages, message)
		case "custom_tool_call_output":
			var customToolCallOutItem openai.ResponseCustomToolCallOutput
			if err = json.Unmarshal(inputItem, &customToolCallOutItem); err != nil {
				return messages, fmt.Errorf("unmarshal input message failed: %w", err)
			}
			message := ResponseCustomToolCallOutputToChatMessage(&customToolCallOutItem)
			messages = append(messages, message)
		default:
			fmt.Printf("unsupported input item type: %s \n", string(itemType))
			//return messages, fmt.Errorf("unsupported input item type: %s", string(itemType))
		}

	}
	return messages, nil
}

func ResponseMessageToChatMessage(inputItem *openai.Message) (json.RawMessage, error) {
	var chatMessage openai.ChatAssistantMessage
	var err error
	if inputItem.Role == "developer" {
		chatMessage.Role = "system"
	} else {
		chatMessage.Role = inputItem.Role
	}
	//chatMessage.Role = inputItem.Role
	content, err := ResponseInputContentToChatMessageContent(inputItem.Content)
	if err != nil {
		return nil, fmt.Errorf("convert input content failed: %w", err)
	}
	chatMessage.Content = content
	jsonMsg, _ := json.Marshal(chatMessage)
	return jsonMsg, err
}

func ResponseInputContentToChatMessageContent(content json.RawMessage) (json.RawMessage, error) {
	if len(content) <= 0 {
		return nil, fmt.Errorf("ResponseInputContent is empty")
	}
	// string content
	var strContent string
	if err := json.Unmarshal(content, &strContent); err == nil {
		return content, nil
	}

	type Part struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Detail   string `json:"detail"`
		FileId   string `json:"file_id"`
		IMageUrl string `json:"image_url"`
		FileData string `json:"file_data"`
		FileUrl  string `json:"file_url"`
		FileName string `json:"file_name"`
		// outputText
		LogProbs    json.RawMessage `json:"log_probs"`
		Annotations json.RawMessage `json:"annotations"`
		// Refusal
		Refusal string `json:"refusal"`
	}
	var contents []Part
	if err := json.Unmarshal(content, &contents); err != nil {
		return nil, fmt.Errorf("unmarshal ResponseInputContent failed: %w", err)
	}
	var results []json.RawMessage
	//types: output_text refusal text image_url input_file
	for _, content := range contents {
		switch content.Type {
		case "text", "output_text", "input_text":
			textContent := openai.ChatContentPartText{
				Text: content.Text,
				Type: "text",
			}
			jsonMsg, _ := json.Marshal(textContent)
			results = append(results, jsonMsg)
		case "image_url":
			imageContent := openai.ChatContentPartImage{
				ImageUrl: struct {
					Url    string `json:"url"`
					Detail string `json:"detail,omitempty"`
				}{Url: content.IMageUrl, Detail: content.Detail},
				Type: "image_url",
			}
			jsonMsg, _ := json.Marshal(imageContent)
			results = append(results, jsonMsg)
		case "input_file":
			fileContent := openai.ChatContentPartFile{
				Type: "file",
				File: struct {
					FileData string `json:"file_data,omitempty"`
					FileId   string `json:"file_id,omitempty"`
					Filename string `json:"filename,omitempty"`
				}{content.FileData, content.FileId, content.FileName},
			}
			jsonMsg, _ := json.Marshal(fileContent)
			results = append(results, jsonMsg)
		case "refusal":
			refusalContent := openai.ChatContentPartRefusal{
				Refusal: content.Refusal,
				Type:    "refusal",
			}
			jsonMsg, _ := json.Marshal(refusalContent)
			results = append(results, jsonMsg)
		default:
			fmt.Printf("unsported ResponseInputContent type: %s \n", content.Type)
		}
	}
	jsonMsg, _ := json.Marshal(results)
	return jsonMsg, nil
}

func ResponseFunctionToolCallToChatMessage(inputItem *openai.ResponseFunctionCall) json.RawMessage {
	message := openai.ChatAssistantMessage{
		Role: "assistant",
	}
	toolCall := openai.ChatMessageToolCall{
		Id:   inputItem.CallID,
		Type: "function",
		Function: &openai.ChatMessageToolCallFunction{
			Name:      inputItem.Name,
			Arguments: inputItem.Arguments,
		},
	}
	message.ToolCalls = []openai.ChatMessageToolCall{toolCall}
	jsonMsg, _ := json.Marshal(message)
	return jsonMsg
}

func ResponseCustomToolCallToChatMessage(inputItem *openai.ResponseCustomToolCall) json.RawMessage {
	message := openai.ChatAssistantMessage{
		Role: "assistant",
	}
	toolCall := openai.ChatMessageToolCall{
		Id:   inputItem.CallId,
		Type: "custom",
		Custom: &openai.ChatMessageToolCallCustom{
			Input: inputItem.Input,
			Name:  inputItem.Name,
		},
	}
	message.ToolCalls = []openai.ChatMessageToolCall{toolCall}
	jsonMsg, _ := json.Marshal(message)
	return jsonMsg
}

func ResponseFunctionCallOutputToChatMessage(inputItem *openai.FunctionCallOutput) json.RawMessage {
	message := openai.ChatToolMessage{
		Role:       "tool",
		ToolCallId: inputItem.CallId,
		// 没有考虑inputImageContent和inputFileContent
		// OpenAI completion 不支持 image和file
		Content: inputItem.Output,
	}

	jsonMsg, _ := json.Marshal(message)
	return jsonMsg
}

func ResponseCustomToolCallOutputToChatMessage(inputItem *openai.ResponseCustomToolCallOutput) json.RawMessage {
	message := openai.ChatToolMessage{
		Role:       "tool",
		ToolCallId: inputItem.CallId,
		Content:    inputItem.Output,
	}

	jsonMsg, _ := json.Marshal(message)
	return jsonMsg
}

func getType(raw json.RawMessage) string {
	var mp map[string]any
	if err := json.Unmarshal(raw, &mp); err != nil {
		return ""
	}
	if v, ok := mp["type"].(string); ok {
		return v
	}
	//typ, ok := mp["type"].(string)
	//if !ok {
	//	return ""
	//}
	return ""
}

func ResponseToolsToChatTools(tools []json.RawMessage) []openai.ChatTool {
	chatTools := make([]openai.ChatTool, 0, len(tools))
	for _, tool := range tools {
		typ := getType(tool)
		if typ == "function" {
			var functionTool openai.FunctionTool
			if err := json.Unmarshal(tool, &functionTool); err != nil {
				fmt.Println("unmarshal function tool failed: ", err)
				continue
			}
			chatTools = append(chatTools, openai.ChatTool{
				Type:     "function",
				Function: &functionTool,
			})
			continue
		}
		if typ == "custom" {
			var customTool openai.CustomTool
			if err := json.Unmarshal(tool, &customTool); err != nil {
				fmt.Println("unmarshal custom tool failed: ", err)
				continue
			}
			chatTools = append(chatTools, openai.ChatTool{
				Type:   "custom",
				Custom: &customTool,
			})
		}
	}
	return chatTools
}

func ResponseToolChoiceToChatToolChoice(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	// 只支持: string,allowedToolChoice,NamedToolChoice,Custom
	var text string
	var err error
	if err = json.Unmarshal(raw, &text); err == nil {
		return raw, nil
	}
	var choice openai.ResponseToolChoice
	if err = json.Unmarshal(raw, &choice); err == nil {
		return raw, nil
	}
	chatChoice := openai.ChatToolChoice{}
	switch choice.Type {
	case "allowed_tools":
		chatChoice.Type = "allowed_tools"
		chatChoice.AllowedTools = &openai.ChatAllowTools{
			Mode:  choice.Mode,
			Tools: choice.Tools,
		}
	case "function":
		chatChoice.Type = "function"
		chatChoice.Function = &openai.NameObject{
			Name: choice.Name,
		}
	case "custom":
		chatChoice.Type = "custom"
		chatChoice.Custom = &openai.NameObject{
			Name: choice.Name,
		}
	default:
		fmt.Printf("unsupported tool choice type: %v\n", choice.Type)
		fmt.Printf("default: %v\n", choice)
		return nil, nil
	}
	jsonBytes, err := json.Marshal(chatChoice)
	if err != nil {
		return nil, fmt.Errorf("marshal tool choice failed: %w", err)
	}
	return jsonBytes, err
}
