package claude

import "encoding/json"

type MessageStreamEvent struct {
	Type    string           `json:"type"`
	Message *MessageResponse `json:"message,omitempty"`
	Delta   *Delta           `json:"delta,omitempty"`
	Usage   *Usage           `json:"usage,omitempty"`
}

type MessageContentBlockEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
}

type Delta struct {
	Container    *RespContainer `json:"container,omitempty"`
	StopReason   string         `json:"stop_reason,omitempty"`
	StopSequence string         `json:"stop_sequence,omitempty"`
}

type ContentBlockDelta struct {
	Type      string          `json:"type,omitempty"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
}
