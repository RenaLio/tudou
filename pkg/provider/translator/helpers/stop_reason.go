package helpers

// Claude stop reasons:
// - end_turn
// - max_tokens
// - stop_sequence
// - tool_use
// - pause_turn
// - refusal

// OpenAI stop reasons:
// - stop
// - length
// - tool_calls
// - content_filter

// OpenAI Responses incomplete_details:
// - max_output_tokens
// - content_filter

// ClaudeStopReasonToOpenAI converts Claude stop reason to OpenAI Chat Completions finish_reason
//
// Parameter:
//   - stopReason - The stop reason from Claude API
//
// Returns:
//   - The corresponding finish_reason for OpenAI Chat Completions API
//
// Mapping:
//   - end_turn -> stop
//   - max_tokens -> length
//   - stop_sequence -> stop
//   - tool_use -> tool_calls
//   - pause_turn -> stop
//   - refusal -> content_filter
func ClaudeStopReasonToOpenAI(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "pause_turn":
		return "stop"
	case "refusal":
		return "content_filter"
	default:
		return "stop"
	}
}

// OpenAIStopReasonToClaude converts OpenAI Chat Completions finish_reason to Claude stop reason
//
// Parameter:
//   - stopReason - The finish_reason from OpenAI Chat Completions API
//
// Returns:
//   - The corresponding stop reason for Claude API
//
// Mapping:
//   - stop -> end_turn
//   - length -> max_tokens
//   - tool_calls -> tool_use
//   - content_filter -> refusal
func OpenAIStopReasonToClaude(stopReason string) string {
	switch stopReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "refusal"
	default:
		return "end_turn"
	}
}

// ClaudeStopReasonToOpenAIResponses converts Claude stop reason to OpenAI Responses status
//
// Parameter:
//   - stopReason - The stop reason from Claude API
//
// Returns:
//   - The status for OpenAI Responses API (completed or incomplete)
//   - The incomplete reason if status is incomplete, empty otherwise
//
// Mapping:
//   - end_turn -> completed, ""
//   - max_tokens -> incomplete, "max_output_tokens"
//   - stop_sequence -> completed, ""
//   - tool_use -> completed, ""
//   - pause_turn -> completed, ""
//   - refusal -> incomplete, "content_filter"
func ClaudeStopReasonToOpenAIResponses(stopReason string) (status string, incompleteReason string) {
	switch stopReason {
	case "end_turn":
		return "completed", ""
	case "max_tokens":
		return "incomplete", "max_output_tokens"
	case "stop_sequence":
		return "completed", ""
	case "tool_use":
		return "completed", ""
	case "pause_turn":
		return "completed", ""
	case "refusal":
		return "incomplete", "content_filter"
	default:
		return "completed", ""
	}
}

// OpenAIResponsesStopReasonToClaude converts OpenAI Responses incomplete reason to Claude stop reason
//
// Parameter:
//   - incompleteReason - The incomplete reason from OpenAI Responses API
//
// Returns:
//   - The corresponding stop reason for Claude API
//
// Mapping:
//   - max_output_tokens -> max_tokens
//   - content_filter -> refusal
//   - "" (completed) -> end_turn
func OpenAIResponsesStopReasonToClaude(incompleteReason string) string {
	switch incompleteReason {
	case "max_output_tokens":
		return "max_tokens"
	case "content_filter":
		return "refusal"
	default:
		return "end_turn"
	}
}

// OpenAIResponsesStatusToClaude converts OpenAI Responses status and incomplete reason to Claude stop reason
//
// Parameters:
//   - status - The status from OpenAI Responses API (completed or incomplete)
//   - incompleteReason - The incomplete reason if status is incomplete
//
// Returns:
//   - The corresponding stop reason for Claude API
//
// Mapping:
//   - completed -> end_turn
//   - incomplete + max_output_tokens -> max_tokens
//   - incomplete + content_filter -> refusal
func OpenAIResponsesStatusToClaude(status, incompleteReason string) string {
	if status == "completed" {
		return "end_turn"
	}
	return OpenAIResponsesStopReasonToClaude(incompleteReason)
}

// OpenAIStopReasonToResponses converts OpenAI Chat Completions finish_reason to OpenAI Responses status
//
// Parameter:
//   - stopReason - The finish_reason from OpenAI Chat Completions API
//
// Returns:
//   - The status for OpenAI Responses API (completed or incomplete)
//   - The incomplete reason if status is incomplete, empty otherwise
//
// Mapping:
//   - stop -> completed, ""
//   - length -> incomplete, "max_output_tokens"
//   - tool_calls -> completed, ""
//   - content_filter -> incomplete, "content_filter"
func OpenAIStopReasonToResponses(stopReason string) (status string, incompleteReason string) {
	switch stopReason {
	case "stop":
		return "completed", ""
	case "length":
		return "incomplete", "max_output_tokens"
	case "tool_calls":
		return "completed", ""
	case "content_filter":
		return "incomplete", "content_filter"
	default:
		return "completed", ""
	}
}

// OpenAIResponsesToStopReason converts OpenAI Responses status to OpenAI Chat Completions finish_reason
//
// Parameter:
//   - status - The status from OpenAI Responses API (completed or incomplete)
//   - incompleteReason - The incomplete reason if status is incomplete
//
// Returns:
//   - The corresponding finish_reason for OpenAI Chat Completions API
//
// Mapping:
//   - completed -> stop
//   - incomplete + max_output_tokens -> length
//   - incomplete + content_filter -> content_filter
func OpenAIResponsesToStopReason(status, incompleteReason string) string {
	if status == "completed" {
		return "stop"
	}
	switch incompleteReason {
	case "max_output_tokens":
		return "length"
	case "content_filter":
		return "content_filter"
	default:
		return "stop"
	}
}
