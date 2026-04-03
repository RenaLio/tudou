package openairesponses

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/tidwall/gjson"
)

// ConvertResponsesStreamToClaude 将 OpenAI Responses 流式响应转换为 Claude 流式响应
func ConvertResponsesStreamToClaude(ctx context.Context, originReq *types.Request, resp *types.Response) (*types.Response, error) {
	cp := common.CloneResponse(resp)
	st := types.NewComplexEventStream(ctx, resp.Stream)
	cp.Stream = st
	cp.IsStream = true
	cp.Format = types.FormatClaudeMessages
	go runResponsesToClaudeStream(st, originReq)
	return cp, nil
}

// runResponsesToClaudeStream 运行流式转换
func runResponsesToClaudeStream(stream *types.ComplexEventStream, originReq *types.Request) {
	defer stream.CloseCh()
	message := &claudemodel.MessageResponse{
		ID:        "msg_" + strconv.FormatInt(time.Now().Unix(), 10),
		Container: nil,
		Content:   []json.RawMessage{},
		Model:     originReq.Model,
		Role:      "assistant",
		Type:      "message",
		Usage: &claudemodel.Usage{
			InputTokens:  0,
			OutputTokens: 0,
		},
	}
	data := &claudeState{
		CurrentOutputIndex: 0,
		CurrentBlockType:   NoneDelta,
		//FunctionCalls:      make(map[int]*functionCallState),
	}

	for {
		select {
		case <-stream.Done():
			return
		default:
		}
		event, err := stream.Pull()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			plog.Warn("Recv error", err)
			stream.SetErr(err)
			return
		}

		chunk := event.Content
		chunk = bytes.TrimSpace(chunk)
		line := string(chunk)

		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			continue
		}
		line = strings.TrimPrefix(line, "data:")
		line = strings.TrimSpace(line)

		plog.Debug("trans_pull_line: ", line)

		// 获取事件类型
		switch getType(json.RawMessage(line)) {
		case "response.created":
			block := claudemodel.MessageStreamEvent{
				Type:    "message_start",
				Message: message,
			}
			message.ID = firstNotEmpty(gjson.Get(line, "response.id").String(), message.ID)
			writeEvent(stream, "message_start", block)
		case "response.completed":
			usageJson := gjson.Get(line, "response.usage").Raw
			var usage oaimodel.ResponseUsage
			if err := json.Unmarshal([]byte(usageJson), &usage); err == nil {
				message.Usage = helpers.ResponseUsageToClaudeUsage(&usage)
			}
			block := claudemodel.MessageStreamEvent{
				Type:    "message_delta",
				Message: message,
				Usage:   message.Usage,
				Delta: &claudemodel.Delta{
					Container:  nil,
					StopReason: "end_turn",
				},
			}
			writeEvent(stream, "message_delta", block)
		case "response.failed":
		case "response.incomplete":
		case "response.output_item.added":
			handleOutputAdded(stream, line, data)
		case "response.output_item.done":
			finishChunkState(stream, line, data)
		case "response.content_part.added":
		case "response.content_part.done":
		case "response.output_text.delta":
			handleOutputTextDelta(stream, line, data)
		case "response.output_text.done":
		case "response.refusal.delta":
		case "response.refusal.done":
		case "response.function_call_arguments.delta":
			handleFunctionCallDelta(stream, line, data)
		case "response.function_call_arguments.done":
		case "response.file_search_call.in_progress":
		case "response.file_search_call.searching":
		case "response.file_search_call.completed":
		case "response.web_search_call.in_progress":
		case "response.web_search_call.searching":
		case "response.web_search_call.completed":
		case "response.reasoning_summary_part.added":
		case "response.reasoning_summary_part.done":
		case "response.reasoning_summary_text.delta":
			handleReasoningDelta(stream, line, data)
		case "response.reasoning_summary_text.done":
		case "response.reasoning_text.delta":
		case "response.reasoning_text.done":
		case "response.image_generation_call.completed":
		case "response.image_generation_call.generating":
		case "response.image_generation_call.in_progress":
		case "response.image_generation_call.partial_image":
		case "response.mcp_call_arguments.delta":
		case "response.mcp_call_arguments.done":
		case "response.mcp_call.completed":
		case "response.mcp_call.failed":
		case "response.mcp_call.in_progress":
		case "response.mcp_list_tools.completed":
		case "response.mcp_list_tools.failed":
		case "response.mcp_list_tools.in_progress":
		case "response.code_interpreter_call.in_progress":
		case "response.code_interpreter_call.interpreting":
		case "response.code_interpreter_call.completed":
		case "response.code_interpreter_call_code.delta":
		case "response.code_interpreter_call_code.done":
		case "response.output_text.annotation.added":
		case "response.queued":
		case "response.custom_tool_call_input.delta":
		case "response.custom_tool_call_input.done":
		case "error":
		case "response.audio.transcript.done":
		case "response.audio.transcript.delta":
		case "response.audio.done":
		case "response.audio.delta":
		}
	}

	// message stop
	block := claudemodel.MessageStreamEvent{
		Type: "message_stop",
	}
	writeEvent(stream, "message_stop", block)
}

func getOutIndex(line string, data *claudeState) int64 {
	outIndex := gjson.Get(line, "output_index").Int()
	return outIndex - data.IndexOffset
}

func handleOutputAdded(stream *types.ComplexEventStream, line string, data *claudeState) {
	//index := int(getOutIndex(line, data))
	// 如果是不支持的类型，offset++,最终index: resp_output_index - offset
	block := claudemodel.MessageContentBlockEvent{
		Type:         "content_block_start",
		Index:        data.CurrentOutputIndex,
		ContentBlock: nil,
	}
	switch gjson.Get(line, "item.type").String() {
	case "message":
		block.ContentBlock = mustMarshal(claudemodel.TextBlock{
			Text: "",
			Type: "text",
		})
		writeEvent(stream, "content_block_start", block)
	case "reasoning":
		block.ContentBlock = mustMarshal(claudemodel.ThinkingBlock{
			Thinking: "",
			Type:     "thinking",
		})
		writeEvent(stream, "content_block_start", block)
	case "function_call":
		toolCallId := gjson.Get(line, "item.call_id").String()
		toolName := gjson.Get(line, "item.name").String()
		block.ContentBlock = mustMarshal(claudemodel.ToolUseBlock{
			ID:     "fc_" + toolCallId,
			Caller: nil,
			Input:  make(map[string]any),
			Name:   toolName,
			Type:   "tool_use",
		})
		//data.FunctionCalls[index] = &functionCallState{
		//	Item: oaimodel.ResponseFunctionCall{
		//		Type:      "function",
		//		Arguments: "",
		//		CallID:    toolCallId,
		//		Name:      toolName,
		//		Id:        toolCallId,
		//		Status:    "",
		//	},
		//	OutputIndex: index,
		//}
		writeEvent(stream, "content_block_start", block)
	default:
		data.IndexOffset++
	}
	data.CurrentOutputIndex++
}

// finishChunkState 完成当前块状态
func finishChunkState(stream *types.ComplexEventStream, line string, state *claudeState) {
	index := int(getOutIndex(line, state))
	block := claudemodel.MessageContentBlockEvent{
		Type:         "",
		Index:        index,
		ContentBlock: nil,
		Delta:        nil,
	}
	switch gjson.Get(line, "item.type").String() {
	case "message":
		block.Type = "content_block_stop"
		writeEvent(stream, "content_block_stop", block)
	case "reasoning":
		fallthrough
	case "function_call":
		//signature := map[string]any{
		//	"signature": "",
		//	"type":      "signature_delta",
		//}
		//block.Type = "content_block_delta"
		//block.Delta = mustMarshal(signature)
		//writeEvent(stream, "content_block_delta", block)
		block.Delta = nil
		block.Type = "content_block_stop"
		writeEvent(stream, "content_block_stop", block)
	}
}

// handleOutputTextDelta 处理文本增量事件
func handleOutputTextDelta(stream *types.ComplexEventStream, line string, data *claudeState) {
	delta := gjson.Get(line, "delta").String()
	eventType := "content_block_delta"
	contentEvent := claudemodel.MessageContentBlockEvent{
		Type:  eventType,
		Index: int(getOutIndex(line, data)),
		Delta: nil,
	}
	block := claudemodel.TextBlock{
		Text: delta,
		Type: "text_delta",
	}
	contentEvent.Delta = mustMarshal(block)
	writeEvent(stream, eventType, contentEvent)
}

// handleFunctionCallDelta 处理函数调用增量事件
func handleFunctionCallDelta(stream *types.ComplexEventStream, line string, data *claudeState) {
	delta := gjson.Get(line, "delta").String()
	eventType := "content_block_delta"
	jsonDelta := map[string]string{
		"partial_json": delta,
		"type":         "input_json_delta",
	}
	contentEvent := claudemodel.MessageContentBlockEvent{
		Type:  eventType,
		Index: int(getOutIndex(line, data)),
		Delta: nil,
	}
	contentEvent.Delta = mustMarshal(jsonDelta)
	writeEvent(stream, eventType, contentEvent)
}

// handleReasoningDelta 处理推理增量事件
func handleReasoningDelta(stream *types.ComplexEventStream, line string, data *claudeState) {
	delta := gjson.Get(line, "delta").String()

	eventType := "content_block_delta"
	contentEvent := claudemodel.MessageContentBlockEvent{
		Type:  eventType,
		Index: int(getOutIndex(line, data)),
		Delta: nil,
	}
	block := claudemodel.ThinkingBlock{
		Thinking: delta,
		Type:     "thinking_delta",
	}
	contentEvent.Delta = mustMarshal(block)
	writeEvent(stream, eventType, contentEvent)
}

// writeEvent 向流中写入事件
func writeEvent(stream *types.ComplexEventStream, event string, data any) {
	var err error
	b := mustMarshal(data)
	plog.Debug("trans_push_line: ", string(b))
	err = stream.Send([]byte("event: " + event + "\n"))
	if err != nil {
		stream.SetErr(err)
		return
	}
	err = stream.Send([]byte("data: " + string(b) + "\n\n"))
	if err != nil {
		stream.SetErr(err)
	}
	return
}

// claudeState 维护 Claude 流式转换状态
type claudeState struct {
	IndexOffset int64

	CurrentOutputIndex int
	CurrentBlockType   BlockDeltaType

	//FunctionCalls     map[int]*functionCallState
	//FunctionCallOrder []int
}

type BlockDeltaType string

const (
	NoneDelta     BlockDeltaType = ""
	TextDelta     BlockDeltaType = "text"
	ThinkingDelta BlockDeltaType = "thinking"
	ToolUseDelta  BlockDeltaType = "tool_use"
)

// functionCallState 维护函数调用状态
type functionCallState struct {
	Item        oaimodel.ResponseFunctionCall
	OutputIndex int
}

// RandomStringId 生成随机字符串ID
func RandomStringId(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

func firstNotEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
