package claude

import (
	"bytes"
	"context"

	"github.com/RenaLio/tudou/pkg/provider/plog"

	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

func ConvertClaudeStreamToChatCompletion(ctx context.Context, originReq *types.Request, resp *types.Response) (*types.Response, error) {
	cp := common.CloneResponse(resp)
	st := types.NewComplexEventStream(ctx, resp.Stream)
	cp.Stream = st
	cp.IsStream = true
	cp.Format = types.FormatChatCompletion
	go runConvertClaudeToChatSSE(st, originReq)
	return cp, nil
}

type chatCompletionState struct {
	messageID string
	model     string
	created   int64
	usage     *oaimodel.ChatCompletionUsage

	toolCallMap                map[string]oaimodel.ChatMessageToolCall // key: tool call id
	toolCalls                  []string                                // order
	toolId, toolName, toolArgs string

	msgIndex  int64
	toolIndex int

	currentBlockIndex int
	currentBlockType  string
	stopReason        string
}

func runConvertClaudeToChatSSE(stream *types.ComplexEventStream, originReq *types.Request) {
	defer stream.CloseCh()

	chatID := "chatcmpl-" + generateChatID()
	createdAt := time.Now().Unix()
	model := originReq.Model
	if model == "" {
		model = "claude-model"
	}

	state := &chatCompletionState{
		messageID:   chatID,
		model:       model,
		created:     createdAt,
		toolCallMap: make(map[string]oaimodel.ChatMessageToolCall),
		toolCalls:   make([]string, 0),
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
			plog.Debug("Recv error", "err", err)
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
		plog.Debug("trans_pull_line: ", "line", line)

		switch getType([]byte(line)) {
		case "ping", "error": // Ignore ping and error events
		case "message_stop":
			break
		case "message_delta":
			var claudeEvent claudemodel.MessageStreamEvent
			if err = json.Unmarshal([]byte(line), &claudeEvent); err != nil {
				continue
			}
			if claudeEvent.Usage != nil {
				chatUsage := helpers.ClaudeUsageToChatUsage(claudeEvent.Usage)
				state.usage = helpers.MergeOpenAIUsage(state.usage, chatUsage)
			}
			if claudeEvent.Delta != nil && claudeEvent.Delta.StopReason != "" {
				state.stopReason = convertStopReason(claudeEvent.Delta.StopReason)
			}
		case "message_start":
			var claudeEvent claudemodel.MessageStreamEvent
			if err = json.Unmarshal([]byte(line), &claudeEvent); err != nil {
				continue
			}
			if claudeEvent.Message != nil {
				state.messageID = claudeEvent.Message.ID
			}
			state.usage = helpers.ClaudeUsageToChatUsage(claudeEvent.Usage)
		case "content_block_start":
			// 如果是 tool_use,记录一下id
			var claudeEvent claudemodel.MessageContentBlockEvent
			if err = json.Unmarshal([]byte(line), &claudeEvent); err != nil {
				continue
			}
			blockType := getType(claudeEvent.ContentBlock)
			state.currentBlockType = blockType
			if blockType == "tool_use" {
				toolCallId := gjson.GetBytes(claudeEvent.ContentBlock, "id").String()
				state.toolId = toolCallId
				toolName := gjson.GetBytes(claudeEvent.ContentBlock, "name").String()
				state.toolName = toolName
				state.toolCallMap[toolCallId] = oaimodel.ChatMessageToolCall{
					Id:   toolCallId,
					Type: "function",
					Function: &oaimodel.ChatMessageToolCallFunction{
						Name:      toolName,
						Arguments: "",
					},
				}
				state.toolCalls = append(state.toolCalls, toolCallId)
				completions := getCompletion(state)
				toolCall := state.toolCallMap[toolCallId]
				toolCalls := []oaimodel.ChoiceDeltaToolCall{
					{ChatMessageToolCall: toolCall, Index: 0},
				}
				completions.Choices[0].Delta.ToolCalls = toolCalls
				writeChatCompletionEvent(stream, completions)
			}
		case "content_block_delta":
			var claudeEvent claudemodel.MessageContentBlockEvent
			if err = json.Unmarshal([]byte(line), &claudeEvent); err != nil {
				continue
			}
			chunk := convertClaudeDeltaToChatChunk(state, claudeEvent.Delta)
			if chunk != nil {
				writeChatCompletionEvent(stream, chunk)
			}
		case "content_block_stop":
			if state.currentBlockType == "tool_use" {
				state.toolIndex++
			}
			// Block finished, reset current block
			state.currentBlockType = ""
		}
	}
	chunk := getCompletion(state)
	//chunk.Choices[0].FinishReason = state.stopReason
	writeChatCompletionEvent(stream, chunk)
}

func getCompletion(state *chatCompletionState) *oaimodel.ChatCompletionStream {
	return &oaimodel.ChatCompletionStream{
		Id:      state.messageID,
		Object:  "chat.completion.chunk",
		Created: state.created,
		Model:   state.model,
		Usage:   state.usage,
		Choices: []oaimodel.StreamChoiceDelta{
			{
				Index: state.msgIndex,
				Delta: oaimodel.ChoiceDelta{
					Role: "assistant",
				},
				FinishReason: state.stopReason,
			},
		},
	}
}

func generateChatID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func convertClaudeDeltaToChatChunk(state *chatCompletionState, delta json.RawMessage) *oaimodel.ChatCompletionStream {
	if delta == nil {
		return nil
	}

	chatChunk := getCompletion(state)

	switch getType(delta) {
	case "text_delta":
		if gjson.GetBytes(delta, "text").Exists() {
			text := gjson.GetBytes(delta, "text").String()
			chatChunk.Choices[0].Delta.Content = text
		}
	case "thinking_delta":
		if gjson.GetBytes(delta, "thinking").Exists() {
			chatChunk.Choices[0].Delta.ReasoningContent = gjson.GetBytes(delta, "thinking").String()
		}
	case "input_json_delta", "partial_json":
		// Tool call delta
		if gjson.GetBytes(delta, "partial_json").Exists() {
			partialJSON := gjson.GetBytes(delta, "partial_json").String()
			chatChunk.Choices[0].Delta.ToolCalls = []oaimodel.ChoiceDeltaToolCall{
				{
					Index: state.toolIndex,
					ChatMessageToolCall: oaimodel.ChatMessageToolCall{
						Id:   state.toolId,
						Type: "function",
						Function: &oaimodel.ChatMessageToolCallFunction{
							Name:      state.toolName,
							Arguments: partialJSON,
						},
					},
				},
			}
		}
	}

	return chatChunk
}

func buildFinalChunk(state *chatCompletionState) *oaimodel.ChatCompletionStream {
	return &oaimodel.ChatCompletionStream{
		Id:      state.messageID,
		Object:  "chat.completion.chunk",
		Created: state.created,
		Model:   state.model,
		Usage:   state.usage,
		Choices: []oaimodel.StreamChoiceDelta{
			{
				Index:        0,
				FinishReason: state.stopReason,
				Delta:        oaimodel.ChoiceDelta{},
			},
		},
	}
}

func writeChatCompletionEvent(stream *types.ComplexEventStream, chunk *oaimodel.ChatCompletionStream) {
	if chunk == nil {
		return
	}
	data := mustMarshal(chunk)
	plog.Debug("trans_push_line: ", "data", string(data))
	err := stream.Send([]byte("data: " + string(data) + "\n\n"))
	if err != nil {
		stream.SetErr(err)
	}
}

func convertStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return reason
	}
}
