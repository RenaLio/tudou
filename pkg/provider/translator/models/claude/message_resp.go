package claude

import (
	"bytes"

	"fmt"

	"github.com/goccy/go-json"
)

// MessageResponse 对应 Claude /v1/messages 非流式响应。
type MessageResponse struct {
	ID           string            `json:"id"`
	Container    *RespContainer    `json:"container,omitempty"`
	Content      []json.RawMessage `json:"content"`
	Model        string            `json:"model,omitempty"`
	Role         string            `json:"role,omitempty"`
	StopReason   string            `json:"stop_reason,omitempty"`
	StopSequence string            `json:"stop_sequence,omitempty"`
	Type         string            `json:"type,omitempty" default:"message"`

	Usage *Usage `json:"usage,omitempty"`
}

type RespContainer struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

type Usage struct {
	CacheCreation            *CacheCreation   `json:"cache_creation,omitempty"`
	CacheCreationInputTokens *int64           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64           `json:"cache_read_input_tokens"`
	InferenceGeo             string           `json:"inference_geo,omitempty"`
	InputTokens              int64            `json:"input_tokens"`
	OutputTokens             int64            `json:"output_tokens"`
	ServerToolUse            *ServerToolUsage `json:"server_tool_use,omitempty"`
	ServiceTier              string           `json:"service_tier,omitempty"`
}

type CacheCreation struct {
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
}

type ServerToolUsage struct {
	WebFetchRequests  int64 `json:"web_fetch_requests"`
	WebSearchRequests int64 `json:"web_search_requests"`
}

// ContentBlock is the response content union where each payload is distinguished by `type`.
type ContentBlock struct {
	Type                              string
	Text                              *TextBlock
	Thinking                          *ThinkingBlock
	RedactedThinking                  *RedactedThinkingBlock
	ToolUse                           *ToolUseBlock
	ServerToolUse                     *ServerToolUseBlock
	WebSearchToolResult               *WebSearchToolResultBlock
	WebFetchToolResult                *WebFetchToolResultBlock
	CodeExecutionToolResult           *CodeExecutionToolResultBlock
	BashCodeExecutionToolResult       *BashCodeExecutionToolResultBlock
	TextEditorCodeExecutionToolResult *TextEditorCodeExecutionToolResultBlock
	ToolSearchToolResult              *ToolSearchToolResultBlock
	ContainerUpload                   *ContainerUploadBlock
	Raw                               json.RawMessage
}

func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	*c = ContentBlock{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}
	c.Type = t

	switch t {
	case "text":
		var v TextBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Text = &v
	case "thinking":
		var v ThinkingBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Thinking = &v
	case "redacted_thinking":
		var v RedactedThinkingBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.RedactedThinking = &v
	case "tool_use":
		var v ToolUseBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ToolUse = &v
	case "server_tool_use":
		var v ServerToolUseBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ServerToolUse = &v
	case "web_search_tool_result":
		var v WebSearchToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.WebSearchToolResult = &v
	case "web_fetch_tool_result":
		var v WebFetchToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.WebFetchToolResult = &v
	case "code_execution_tool_result":
		var v CodeExecutionToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.CodeExecutionToolResult = &v
	case "bash_code_execution_tool_result":
		var v BashCodeExecutionToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.BashCodeExecutionToolResult = &v
	case "text_editor_code_execution_tool_result":
		var v TextEditorCodeExecutionToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.TextEditorCodeExecutionToolResult = &v
	case "tool_search_tool_result":
		var v ToolSearchToolResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ToolSearchToolResult = &v
	case "container_upload":
		var v ContainerUploadBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ContainerUpload = &v
	default:
		// Keep unknown payload in Raw for forward compatibility.
	}

	return nil
}

func (c *ContentBlock) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "text":
		if c.Text == nil {
			return nil, fmt.Errorf("content block type text without payload")
		}
		return json.Marshal(c.Text)
	case "thinking":
		if c.Thinking == nil {
			return nil, fmt.Errorf("content block type thinking without payload")
		}
		return json.Marshal(c.Thinking)
	case "redacted_thinking":
		if c.RedactedThinking == nil {
			return nil, fmt.Errorf("content block type redacted_thinking without payload")
		}
		return json.Marshal(c.RedactedThinking)
	case "tool_use":
		if c.ToolUse == nil {
			return nil, fmt.Errorf("content block type tool_use without payload")
		}
		return json.Marshal(c.ToolUse)
	case "server_tool_use":
		if c.ServerToolUse == nil {
			return nil, fmt.Errorf("content block type server_tool_use without payload")
		}
		return json.Marshal(c.ServerToolUse)
	case "web_search_tool_result":
		if c.WebSearchToolResult == nil {
			return nil, fmt.Errorf("content block type web_search_tool_result without payload")
		}
		return json.Marshal(c.WebSearchToolResult)
	case "web_fetch_tool_result":
		if c.WebFetchToolResult == nil {
			return nil, fmt.Errorf("content block type web_fetch_tool_result without payload")
		}
		return json.Marshal(c.WebFetchToolResult)
	case "code_execution_tool_result":
		if c.CodeExecutionToolResult == nil {
			return nil, fmt.Errorf("content block type code_execution_tool_result without payload")
		}
		return json.Marshal(c.CodeExecutionToolResult)
	case "bash_code_execution_tool_result":
		if c.BashCodeExecutionToolResult == nil {
			return nil, fmt.Errorf("content block type bash_code_execution_tool_result without payload")
		}
		return json.Marshal(c.BashCodeExecutionToolResult)
	case "text_editor_code_execution_tool_result":
		if c.TextEditorCodeExecutionToolResult == nil {
			return nil, fmt.Errorf("content block type text_editor_code_execution_tool_result without payload")
		}
		return json.Marshal(c.TextEditorCodeExecutionToolResult)
	case "tool_search_tool_result":
		if c.ToolSearchToolResult == nil {
			return nil, fmt.Errorf("content block type tool_search_tool_result without payload")
		}
		return json.Marshal(c.ToolSearchToolResult)
	case "container_upload":
		if c.ContainerUpload == nil {
			return nil, fmt.Errorf("content block type container_upload without payload")
		}
		return json.Marshal(c.ContainerUpload)
	default:
		if len(c.Raw) > 0 {
			return c.Raw, nil
		}
		return nil, fmt.Errorf("unknown content block type %q", c.Type)
	}
}

type TextBlock struct {
	Citations []TextCitation `json:"citations,omitempty"`
	Text      string         `json:"text"`
	Type      string         `json:"type"`
}

type TextCitation struct {
	Type                    string
	CharLocation            *CitationCharLocation
	PageLocation            *CitationPageLocation
	ContentBlockLocation    *CitationContentBlockLocation
	WebSearchResultLocation *CitationWebSearchResultLocation
	SearchResultLocation    *CitationSearchResultLocation
	Raw                     json.RawMessage
}

func (c *TextCitation) UnmarshalJSON(data []byte) error {
	*c = TextCitation{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}
	c.Type = t

	switch t {
	case "char_location":
		var v CitationCharLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.CharLocation = &v
	case "page_location":
		var v CitationPageLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.PageLocation = &v
	case "content_block_location":
		var v CitationContentBlockLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ContentBlockLocation = &v
	case "web_search_result_location":
		var v CitationWebSearchResultLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.WebSearchResultLocation = &v
	case "search_result_location":
		var v CitationSearchResultLocation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.SearchResultLocation = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *TextCitation) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "char_location":
		if c.CharLocation == nil {
			return nil, fmt.Errorf("citation type char_location without payload")
		}
		return json.Marshal(c.CharLocation)
	case "page_location":
		if c.PageLocation == nil {
			return nil, fmt.Errorf("citation type page_location without payload")
		}
		return json.Marshal(c.PageLocation)
	case "content_block_location":
		if c.ContentBlockLocation == nil {
			return nil, fmt.Errorf("citation type content_block_location without payload")
		}
		return json.Marshal(c.ContentBlockLocation)
	case "web_search_result_location":
		if c.WebSearchResultLocation == nil {
			return nil, fmt.Errorf("citation type web_search_result_location without payload")
		}
		return json.Marshal(c.WebSearchResultLocation)
	case "search_result_location":
		if c.SearchResultLocation == nil {
			return nil, fmt.Errorf("citation type search_result_location without payload")
		}
		return json.Marshal(c.SearchResultLocation)
	default:
		if len(c.Raw) > 0 {
			return c.Raw, nil
		}
		return nil, fmt.Errorf("unknown citation type %q", c.Type)
	}
}

type CitationCharLocation struct {
	CitedText      string  `json:"cited_text"`
	DocumentIndex  int     `json:"document_index"`
	DocumentTitle  string  `json:"document_title"`
	EndCharIndex   int     `json:"end_char_index"`
	FileID         *string `json:"file_id,omitempty"`
	StartCharIndex int     `json:"start_char_index"`
	Type           string  `json:"type"`
}

type CitationPageLocation struct {
	CitedText       string  `json:"cited_text"`
	DocumentIndex   int     `json:"document_index"`
	DocumentTitle   string  `json:"document_title"`
	EndPageNumber   int     `json:"end_page_number"`
	FileID          *string `json:"file_id,omitempty"`
	StartPageNumber int     `json:"start_page_number"`
	Type            string  `json:"type"`
}

type CitationContentBlockLocation struct {
	CitedText       string  `json:"cited_text"`
	DocumentIndex   int     `json:"document_index"`
	DocumentTitle   string  `json:"document_title"`
	EndBlockIndex   int     `json:"end_block_index"`
	FileID          *string `json:"file_id,omitempty"`
	StartBlockIndex int     `json:"start_block_index"`
	Type            string  `json:"type"`
}

type CitationWebSearchResultLocation struct {
	CitedText      string `json:"cited_text"`
	EncryptedIndex string `json:"encrypted_index"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	URL            string `json:"url"`
}

type CitationSearchResultLocation struct {
	CitedText         string `json:"cited_text"`
	EndBlockIndex     int    `json:"end_block_index"`
	SearchResultIndex int    `json:"search_result_index"`
	Source            string `json:"source"`
	StartBlockIndex   int    `json:"start_block_index"`
	Title             string `json:"title"`
	Type              string `json:"type"`
}

type ThinkingBlock struct {
	Signature string `json:"signature"`
	Thinking  string `json:"thinking"`
	Type      string `json:"type"`
}

type RedactedThinkingBlock struct {
	Data string `json:"data"`
	Type string `json:"type"`
}

// Caller is a union: direct or server-generated caller metadata.
type Caller struct {
	Type               string
	Direct             *DirectCaller
	ServerToolCaller   *ServerToolCaller
	ServerToolCallerV2 *ServerToolCaller20260120
	Raw                json.RawMessage
}

func (c *Caller) UnmarshalJSON(data []byte) error {
	*c = Caller{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}
	c.Type = t

	switch t {
	case "direct":
		var v DirectCaller
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Direct = &v
	case "code_execution_20250825":
		var v ServerToolCaller
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ServerToolCaller = &v
	case "code_execution_20260120":
		var v ServerToolCaller20260120
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ServerToolCallerV2 = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *Caller) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "direct":
		if c.Direct == nil {
			return nil, fmt.Errorf("caller type direct without payload")
		}
		return json.Marshal(c.Direct)
	case "code_execution_20250825":
		if c.ServerToolCaller == nil {
			return nil, fmt.Errorf("caller type code_execution_20250825 without payload")
		}
		return json.Marshal(c.ServerToolCaller)
	case "code_execution_20260120":
		if c.ServerToolCallerV2 == nil {
			return nil, fmt.Errorf("caller type code_execution_20260120 without payload")
		}
		return json.Marshal(c.ServerToolCallerV2)
	default:
		if len(c.Raw) > 0 {
			return c.Raw, nil
		}
		return nil, fmt.Errorf("unknown caller type %q", c.Type)
	}
}

type DirectCaller struct {
	Type string `json:"type"`
}

type ServerToolCaller struct {
	ToolID string `json:"tool_id"`
	Type   string `json:"type"`
}

type ServerToolCaller20260120 struct {
	ToolID string `json:"tool_id"`
	Type   string `json:"type"`
}

type ToolUseBlock struct {
	ID     string         `json:"id"`
	Caller *Caller        `json:"caller,omitempty"`
	Input  map[string]any `json:"input"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
}

type ServerToolUseBlock struct {
	ID     string         `json:"id"`
	Caller *Caller        `json:"caller,omitempty"`
	Input  map[string]any `json:"input"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
}

type WebSearchToolResultBlock struct {
	Caller    *Caller                    `json:"caller,omitempty"`
	Content   WebSearchToolResultContent `json:"content"`
	ToolUseID string                     `json:"tool_use_id"`
	Type      string                     `json:"type"`
}

// WebSearchToolResultContent can be either an error object or a result array.
type WebSearchToolResultContent struct {
	Error   *WebSearchToolResultError
	Results []WebSearchResultBlock
	Raw     json.RawMessage
}

func (c *WebSearchToolResultContent) UnmarshalJSON(data []byte) error {
	*c = WebSearchToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty web search tool result content")
	}

	if trimmed[0] == '[' {
		var v []WebSearchResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Results = v
		return nil
	}

	t, err := getTypeField(data)
	if err != nil {
		return err
	}
	if t == "web_search_tool_result_error" {
		var v WebSearchToolResultError
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
		return nil
	}

	return nil
}

func (c *WebSearchToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.Results != nil {
		return json.Marshal(c.Results)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty web search tool result content payload")
}

type WebSearchToolResultError struct {
	ErrorCode string `json:"error_code"`
	Type      string `json:"type"`
}

type WebSearchResultBlock struct {
	EncryptedContent string `json:"encrypted_content"`
	PageAge          string `json:"page_age"`
	Title            string `json:"title"`
	Type             string `json:"type"`
	URL              string `json:"url"`
}

type WebFetchToolResultBlock struct {
	Caller    *Caller                   `json:"caller,omitempty"`
	Content   WebFetchToolResultContent `json:"content"`
	ToolUseID string                    `json:"tool_use_id"`
	Type      string                    `json:"type"`
}

// WebFetchToolResultContent can be a success result or an error block.
type WebFetchToolResultContent struct {
	Error *WebFetchToolResultErrorBlock
	Data  *WebFetchResultBlock
	Raw   json.RawMessage
}

func (c *WebFetchToolResultContent) UnmarshalJSON(data []byte) error {
	*c = WebFetchToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}

	switch t {
	case "web_fetch_tool_result_error":
		var v WebFetchToolResultErrorBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
	case "web_fetch_result":
		var v WebFetchResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Data = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *WebFetchToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.Data != nil {
		return json.Marshal(c.Data)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty web fetch tool result content payload")
}

type WebFetchToolResultErrorBlock struct {
	ErrorCode string `json:"error_code"`
	Type      string `json:"type"`
}

type WebFetchResultBlock struct {
	Content     DocumentBlock `json:"content"`
	RetrievedAt string        `json:"retrieved_at"`
	Type        string        `json:"type"`
	URL         string        `json:"url"`
}

type DocumentBlock struct {
	Citations *CitationsConfig `json:"citations,omitempty"`
	Source    DocumentSource   `json:"source"`
	Title     string           `json:"title"`
	Type      string           `json:"type"`
}

type CitationsConfig struct {
	Enabled bool `json:"enabled"`
}

// DocumentSource is a union of `base64` pdf source and `text` source.
type DocumentSource struct {
	Type      string
	Base64PDF *Base64PDFSource
	PlainText *PlainTextSource
	Raw       json.RawMessage
}

func (s *DocumentSource) UnmarshalJSON(data []byte) error {
	*s = DocumentSource{}
	s.Raw = append(s.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}
	s.Type = t

	switch t {
	case "base64":
		var v Base64PDFSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		s.Base64PDF = &v
	case "text":
		var v PlainTextSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		s.PlainText = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (s *DocumentSource) MarshalJSON() ([]byte, error) {
	switch s.Type {
	case "base64":
		if s.Base64PDF == nil {
			return nil, fmt.Errorf("document source type base64 without payload")
		}
		return json.Marshal(s.Base64PDF)
	case "text":
		if s.PlainText == nil {
			return nil, fmt.Errorf("document source type text without payload")
		}
		return json.Marshal(s.PlainText)
	default:
		if len(s.Raw) > 0 {
			return s.Raw, nil
		}
		return nil, fmt.Errorf("unknown document source type %q", s.Type)
	}
}

type Base64PDFSource struct {
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
	Type      string `json:"type"`
}

type PlainTextSource struct {
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
	Type      string `json:"type"`
}

type CodeExecutionToolResultBlock struct {
	Content   CodeExecutionToolResultContent `json:"content"`
	ToolUseID string                         `json:"tool_use_id"`
	Type      string                         `json:"type"`
}

// CodeExecutionToolResultContent can be an error, plaintext result, or encrypted result.
type CodeExecutionToolResultContent struct {
	Error           *CodeExecutionToolResultError
	Result          *CodeExecutionResultBlock
	EncryptedResult *EncryptedCodeExecutionResultBlock
	Raw             json.RawMessage
}

func (c *CodeExecutionToolResultContent) UnmarshalJSON(data []byte) error {
	*c = CodeExecutionToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}

	switch t {
	case "code_execution_tool_result_error":
		var v CodeExecutionToolResultError
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
	case "code_execution_result":
		var v CodeExecutionResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Result = &v
	case "encrypted_code_execution_result":
		var v EncryptedCodeExecutionResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.EncryptedResult = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *CodeExecutionToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.Result != nil {
		return json.Marshal(c.Result)
	}
	if c.EncryptedResult != nil {
		return json.Marshal(c.EncryptedResult)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty code execution tool result content payload")
}

type CodeExecutionToolResultError struct {
	ErrorCode string `json:"error_code"`
	Type      string `json:"type"`
}

type CodeExecutionResultBlock struct {
	Content    []CodeExecutionOutputBlock `json:"content"`
	ReturnCode int                        `json:"return_code"`
	Stderr     string                     `json:"stderr"`
	Stdout     string                     `json:"stdout"`
	Type       string                     `json:"type"`
}

type EncryptedCodeExecutionResultBlock struct {
	Content         []CodeExecutionOutputBlock `json:"content"`
	EncryptedStdout string                     `json:"encrypted_stdout"`
	ReturnCode      int                        `json:"return_code"`
	Stderr          string                     `json:"stderr"`
	Type            string                     `json:"type"`
}

type CodeExecutionOutputBlock struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
}

type BashCodeExecutionToolResultBlock struct {
	Content   BashCodeExecutionToolResultContent `json:"content"`
	ToolUseID string                             `json:"tool_use_id"`
	Type      string                             `json:"type"`
}

type BashCodeExecutionToolResultContent struct {
	Error *BashCodeExecutionToolResultError
	Data  *BashCodeExecutionResultBlock
	Raw   json.RawMessage
}

func (c *BashCodeExecutionToolResultContent) UnmarshalJSON(data []byte) error {
	*c = BashCodeExecutionToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}

	switch t {
	case "bash_code_execution_tool_result_error":
		var v BashCodeExecutionToolResultError
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
	case "bash_code_execution_result":
		var v BashCodeExecutionResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Data = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *BashCodeExecutionToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.Data != nil {
		return json.Marshal(c.Data)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty bash code execution tool result content payload")
}

type BashCodeExecutionToolResultError struct {
	ErrorCode string `json:"error_code"`
	Type      string `json:"type"`
}

type BashCodeExecutionResultBlock struct {
	Content    []BashCodeExecutionOutputBlock `json:"content"`
	ReturnCode int                            `json:"return_code"`
	Stderr     string                         `json:"stderr"`
	Stdout     string                         `json:"stdout"`
	Type       string                         `json:"type"`
}

type BashCodeExecutionOutputBlock struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
}

type TextEditorCodeExecutionToolResultBlock struct {
	Content   TextEditorCodeExecutionToolResultContent `json:"content"`
	ToolUseID string                                   `json:"tool_use_id"`
	Type      string                                   `json:"type"`
}

type TextEditorCodeExecutionToolResultContent struct {
	Error            *TextEditorCodeExecutionToolResultError
	ViewResult       *TextEditorCodeExecutionViewResultBlock
	CreateResult     *TextEditorCodeExecutionCreateResultBlock
	StrReplaceResult *TextEditorCodeExecutionStrReplaceResultBlock
	Raw              json.RawMessage
}

func (c *TextEditorCodeExecutionToolResultContent) UnmarshalJSON(data []byte) error {
	*c = TextEditorCodeExecutionToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}

	switch t {
	case "text_editor_code_execution_tool_result_error":
		var v TextEditorCodeExecutionToolResultError
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
	case "text_editor_code_execution_view_result":
		var v TextEditorCodeExecutionViewResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.ViewResult = &v
	case "text_editor_code_execution_create_result":
		var v TextEditorCodeExecutionCreateResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.CreateResult = &v
	case "text_editor_code_execution_str_replace_result":
		var v TextEditorCodeExecutionStrReplaceResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.StrReplaceResult = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *TextEditorCodeExecutionToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.ViewResult != nil {
		return json.Marshal(c.ViewResult)
	}
	if c.CreateResult != nil {
		return json.Marshal(c.CreateResult)
	}
	if c.StrReplaceResult != nil {
		return json.Marshal(c.StrReplaceResult)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty text editor code execution tool result content payload")
}

type TextEditorCodeExecutionToolResultError struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
}

type TextEditorCodeExecutionViewResultBlock struct {
	Content    string `json:"content"`
	FileType   string `json:"file_type"`
	NumLines   int    `json:"num_lines"`
	StartLine  int    `json:"start_line"`
	TotalLines int    `json:"total_lines"`
	Type       string `json:"type"`
}

type TextEditorCodeExecutionCreateResultBlock struct {
	IsFileUpdate bool   `json:"is_file_update"`
	Type         string `json:"type"`
}

type TextEditorCodeExecutionStrReplaceResultBlock struct {
	Lines    []string `json:"lines"`
	NewLines int      `json:"new_lines"`
	NewStart int      `json:"new_start"`
	OldLines int      `json:"old_lines"`
	OldStart int      `json:"old_start"`
	Type     string   `json:"type"`
}

type ToolSearchToolResultBlock struct {
	Content   ToolSearchToolResultContent `json:"content"`
	ToolUseID string                      `json:"tool_use_id"`
	Type      string                      `json:"type"`
}

type ToolSearchToolResultContent struct {
	Error *ToolSearchToolResultError
	Data  *ToolSearchToolSearchResultBlock
	Raw   json.RawMessage
}

func (c *ToolSearchToolResultContent) UnmarshalJSON(data []byte) error {
	*c = ToolSearchToolResultContent{}
	c.Raw = append(c.Raw[:0], data...)

	t, err := getTypeField(data)
	if err != nil {
		return err
	}

	switch t {
	case "tool_search_tool_result_error":
		var v ToolSearchToolResultError
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Error = &v
	case "tool_search_tool_search_result":
		var v ToolSearchToolSearchResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Data = &v
	default:
		// Keep unknown payload in Raw.
	}

	return nil
}

func (c *ToolSearchToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Error != nil {
		return json.Marshal(c.Error)
	}
	if c.Data != nil {
		return json.Marshal(c.Data)
	}
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	return nil, fmt.Errorf("empty tool search tool result content payload")
}

type ToolSearchToolResultError struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
}

type ToolSearchToolSearchResultBlock struct {
	ToolReferences []ToolReferenceBlock `json:"tool_references"`
	Type           string               `json:"type"`
}

type ToolReferenceBlock struct {
	ToolName string `json:"tool_name"`
	Type     string `json:"type"`
}

type ContainerUploadBlock struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
}

func getTypeField(data []byte) (string, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return "", err
	}
	if head.Type == "" {
		return "", fmt.Errorf("missing type field")
	}
	return head.Type, nil
}
