package helpers

import (
	"fmt"

	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// Content Types Documentation:
//
// Claude Media Content Types:
// - TextBlockParam: {"type": "text", "text": "...", "cache_control": {...}, "citations": [...]}
// - ImageBlockParam: {"type": "image", "source": {"type": "url|base64", "url": "..." | "data": "...", "media_type": "..."}}
// - DocumentBlockParam: {"type": "document", "source": {...}, "title": "...", "cache_control": {...}}
// - ThinkingBlockParam: {"type": "thinking", "thinking": "...", "signature": "..."}
// - RedactedThinkingBlockParam: {"type": "redacted_thinking", "data": "..."}
//
// OpenAI Chat Content Part Types:
// - ChatContentPartText: {"type": "text", "text": "..."}
// - ChatContentPartImage: {"type": "image_url", "image_url": {"url": "..."}}
// - ChatContentPartAudio: {"type": "input_audio", "input_audio": {...}}

// OpenAI Response Types:
// - EasyInputMessage
// - Message
// - ResponseOutputMessage
// - Reasoning

// ClaudeMediaContentToOpenAI converts Claude media content blocks to OpenAI Chat Completions content format
// Parameter:
//   - in - The JSON raw message of Claude media content block (TextBlockParam, ImageBlockParam, etc.)
//
// Returns:
//   - json.RawMessage - The OpenAI ChatContentPart format (text or image_url)
//
// Note:
//   - Supports "text", "image", "redacted_thinking", "thinking" types
//   - For text type: returns as-is (ignores cache_control and citations)
//   - For image type: extracts URL from source.url or source.data
//   - For thinking/redacted_thinking: extracts text content
//   - Document type is not supported (returns nil)
//   - Returns nil for nil input or unsupported types
func ClaudeMediaContentToOpenAI(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "text":
		// ignore cache_control,citations
		data := openai.ChatContentPartText{Type: "text", Text: gjson.GetBytes(in, "text").String()}
		return mustMarshal(data)
	case "image":
		data := openai.ChatContentPartImage{Type: "image_url"}
		if gjson.GetBytes(in, "source.url").Exists() {
			data.ImageUrl.Url = gjson.GetBytes(in, "source.url").String()
			return mustMarshal(data)
		}
		if gjson.GetBytes(in, "source.data").Exists() {
			data.ImageUrl.Url = gjson.GetBytes(in, "source.data").String()
			return mustMarshal(data)
		}
	case "document":
	case "redacted_thinking":
		// maybe not need
		text := gjson.GetBytes(in, "data").String()
		if text != "" {
			return mustMarshal(openai.ChatContentPartText{Text: text, Type: "text"})
		}
	case "thinking":
		// maybe not need
		text := gjson.GetBytes(in, "thinking").String()
		if text != "" {
			return mustMarshal(openai.ChatContentPartText{Text: text, Type: "text"})
		}
	}
	return nil
}

// ResponseMediaInputToOpenAI converts OpenAI Response Media Input to OpenAI Chat Completions content format
// Parameter:
//   - in - The JSON raw message of OpenAI Response Media Input
//
// Returns:
//   - json.RawMessage - The OpenAI ChatContentPartText format
//   - nil if input is nil or cannot be unmarshalled to ChatContentPartText

func ResponseMediaInputToOpenAI(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	switch getType(in) {
	case "reasoning": // don't need
	case "message":
		var chatMessage openai.ChatAssistantMessage
		chatMessage.Role = gjson.GetBytes(in, "role").String()
		content := ResponseInputContentToChatMessageContent(json.RawMessage(gjson.GetBytes(in, "content").Raw))
		if content == nil {
			return nil
		}
		chatMessage.Content = content
		return mustMarshal(chatMessage)
	}
	return nil
}

func ResponseInputContentToChatMessageContent(content json.RawMessage) json.RawMessage {
	if content == nil {
		return nil
	}
	// string content
	var strContent string
	if err := json.Unmarshal(content, &strContent); err == nil {
		return content
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
		return nil
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
	return mustMarshal(results)
}
