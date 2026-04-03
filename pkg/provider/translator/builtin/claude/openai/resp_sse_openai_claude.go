package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func ConvertChatCompletionStreamToClaude(ctx context.Context, originReq *types.Request, resp *types.Response) (*types.Response, error) {
	cp := common.CloneResponse(resp)
	st := types.NewComplexEventStream(ctx, resp.Stream)
	cp.Stream = st
	cp.IsStream = true
	cp.Format = types.FormatClaudeMessages
	//go runChatToClaudeStream(st,req)
	go runConvertSSE(st, originReq)
	return cp, nil
}

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

func runConvertSSE(stream *types.ComplexEventStream, originReq *types.Request) {
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
	previousState := CompletionChunkNone
	data := &claudeState{}

	once := sync.Once{}
	for {
		select {
		case <-stream.Done():
			return
		default:
		}
		event, err := stream.Pull()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // 正常结束，触发 defer close
			}
			plog.Error("pull error", err)
			stream.SetErr(err) // 记录底层错误给 Recv 拿去用
			return             // 异常结束，触发 defer close
		}
		once.Do(func() {
			eventType := "message_start"
			startBlock := new(claudemodel.MessageStreamEvent)
			startBlock.Type = eventType
			startBlock.Message = message
			writeEvent(stream, "message_start", startBlock)
		})
		chunk := event.Content
		chunk = bytes.TrimSpace(chunk)
		line := string(chunk)

		if len(line) == 0 {
			continue
		}
		line = strings.TrimPrefix(line, "data: ")
		line = strings.TrimSpace(line)
		if line == "[DONE]" {
			break
		}
		completionChunk := oaimodel.ChatCompletionStream{}
		if err = common.UnmarshalJSON([]byte(line), &completionChunk); err != nil {
			stream.SetErr(err)
			return
		}

		plog.Debug("trans_pull_line: ", line)

		if completionChunk.Usage != nil {
			message.Usage = helpers.ChatUsageToClaudeUsage(completionChunk.Usage)
		}
		currentState := previousState
		// 判断当前chunk的类型
		if len(completionChunk.Choices) == 0 {
			continue
		}
		// N = 1，不存在多个情况
		choice := completionChunk.Choices[0]
		currentState = getCompletionChunkType(&choice)
		if currentState == CompletionChunkTypeFinished {
			continue
		}
		if currentState == CompletionChunkIgnore {
			plog.Warn("ignore chunk", "line", line)
			continue
		}
		fn3(previousState, currentState, stream, &choice, data)
		//fn3
		previousState = currentState
	}
	fn3(previousState, CompletionChunkNone, stream, nil, data)
	// usage
	eventType := "message_delta"
	block := claudemodel.MessageStreamEvent{
		Type:  eventType,
		Usage: message.Usage,
		Delta: &claudemodel.Delta{
			StopReason: "end_turn",
		},
	}
	writeEvent(stream, eventType, block)
	// message stop
	eventType = "message_stop"
	block = claudemodel.MessageStreamEvent{
		Type: eventType,
	}
	writeEvent(stream, eventType, block)
}

type functionCallState struct {
	//Item2       oaimodel.ChatMessageToolCallFunction
	Item        oaimodel.ResponseFunctionCall
	OutputIndex int
}
type claudeState struct {
	CurrentOutputIndex int
	// "delta":{"tool_calls":[{"id":"call_33b585a4c0654e0cb3ad6341","index":0,"type":"function","function":{"name":"shell","arguments":"{\""} } ]}
	ActiveFunctionCalls map[int]*functionCallState
	// first-seen order for active calls，key是所在的位置，value是tool_index
	// 如果同时存在多个工具的call，每次流式响应只返回第一个的结果，当结束时，再传所有tool_call(通过多个start_stop事件)
	// 也可以新建多个startEvent，然后同时更新(毕竟deltaEvent中带有index)
	FunctionCallOrder []int
	finishReason      string
}
type chatClaudeToolState struct {
	started        bool
	id, name, args string
}

type chatClaudeState struct {
	messageID   string
	model       string
	usage       *claudemodel.Usage
	started     bool
	textStarted bool
	textIndex   int
	toolStates  map[int]*chatClaudeToolState
	stopReason  string
}

type CompletionChunkType string

const (
	CompletionChunkNone             CompletionChunkType = ""
	CompletionChunkIgnore           CompletionChunkType = "ignore"
	CompletionChunkTypeMessage      CompletionChunkType = "message"
	CompletionChunkTypeReasoning    CompletionChunkType = "reasoning"
	CompletionChunkTypeFunctionCall CompletionChunkType = "function_call"
	CompletionChunkTypeRefusal      CompletionChunkType = "refusal"
	CompletionChunkTypeFinished     CompletionChunkType = "finished"
)

func RandomStringId(length int) string {
	//const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

func GetIndex() func() int {
	var index int
	return func() int {
		index++
		return index
	}
}

func getCompletionChunkType(choice *oaimodel.StreamChoiceDelta) CompletionChunkType {
	// openai 官方文档 tool_calls中就没有 custom字段
	if len(choice.Delta.ReasoningContent) > 0 {
		return CompletionChunkTypeReasoning
	}
	if len(choice.Delta.ToolCalls) > 0 {
		return CompletionChunkTypeFunctionCall
	}
	if len(choice.Delta.Refusal) > 0 {
		return CompletionChunkTypeRefusal
	}
	if len(choice.Delta.Content) > 0 || len(choice.Delta.Refusal) > 0 {
		return CompletionChunkTypeMessage
	}
	if len(choice.FinishReason) > 0 {
		return CompletionChunkTypeFinished
	}
	return CompletionChunkIgnore
}

type InnerData struct {
	OutputMessage         oaimodel.ResponseOutputMessage
	OutputTextContent     oaimodel.ResponseOutputText
	OutPutMessageOutIndex int
	OutputRefusal         oaimodel.ResponseOutputRefusal
	FunctionToolCalls     []oaimodel.ResponseFunctionCall // completed calls, used by response.completed
	// "delta":{"tool_calls":[{"id":"call_33b585a4c0654e0cb3ad6341","index":0,"type":"function","function":{"name":"shell","arguments":"{\""} } ]}
	ActiveFunctionCalls      map[int]*functionCallState // active calls in current function_call state, 考虑多个tool_calls,key-int代表所在的索引index
	FunctionCallOrder        []int                      // first-seen order for active calls，key是所在的位置，value是tool_index
	ReasoningItem            oaimodel.ResponseReasoningItem
	ReasoningItemOutputIndex int
	ReasoningSummaryText     string
	IdxFunc                  func() int
	CurrentOutputIndex       int
	outputItems              []json.RawMessage
}

func fn3(prevState, currentState CompletionChunkType, stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, state *claudeState) {
	if prevState == CompletionChunkNone && currentState == CompletionChunkNone {
		return
	}

	if prevState != currentState {
		finishChunkState(prevState, stream, state)
		if currentState != CompletionChunkNone {
			startChunkState(currentState, stream, choice, state)
		}
		return
	}

	appendChunkDelta(currentState, stream, choice, state)
}

func finishChunkState(tpe CompletionChunkType, stream *types.ComplexEventStream, state *claudeState) {
	switch tpe {
	case CompletionChunkTypeMessage:
		fallthrough
	case CompletionChunkTypeRefusal:
		finishDefaultState(stream, state)
	case CompletionChunkTypeReasoning:
		signature := &claudemodel.MessageContentBlockEvent{
			Type:  "content_block_delta",
			Index: state.CurrentOutputIndex,
		}
		block := map[string]string{
			"signature": "",
			"type":      "signature_delta",
		}
		signature.Delta = mustMarshal(block)
		writeEvent(stream, "content_block_delta", mustMarshal(signature))
		finishDefaultState(stream, state)
	case CompletionChunkTypeFunctionCall:
		finishFunctionCallState(stream, state)
	}
}

func finishDefaultState(stream *types.ComplexEventStream, data *claudeState) {
	contentBlock := &claudemodel.MessageContentBlockEvent{
		Type:  "content_block_stop",
		Index: data.CurrentOutputIndex,
	}
	writeEvent(stream, "content_block_stop", mustMarshal(contentBlock))
	data.CurrentOutputIndex++
}

func finishFunctionCallState(stream *types.ComplexEventStream, data *claudeState) {
	// 完成第一个函数调用
	// 完成剩余函数调用
	defer func() {
		resetFunctionCallState(data)
	}()
	if len(data.FunctionCallOrder) == 0 {
		return
	}
	firstIndex := data.FunctionCallOrder[0]
	firstFunction := data.ActiveFunctionCalls[firstIndex]
	contentBlock := &claudemodel.MessageContentBlockEvent{
		Type:  "content_block_stop",
		Index: firstFunction.OutputIndex,
	}
	writeEvent(stream, "content_block_stop", mustMarshal(contentBlock))
	data.CurrentOutputIndex++
	if len(data.FunctionCallOrder) <= 1 {
		return
	}
	for _, index := range data.FunctionCallOrder[1:] {
		// - 新建 blockEvent
		// - delta 事件
		// - finish 事件
		function := data.ActiveFunctionCalls[index]
		if function == nil {
			continue
		}
		eventType := "content_block_start"
		blockEvent := claudemodel.MessageContentBlockEvent{
			Type:         eventType,
			Index:        data.CurrentOutputIndex,
			ContentBlock: nil,
			Delta:        nil,
		}
		block := claudemodel.ToolUseBlock{
			ID:    function.Item.CallID,
			Input: make(map[string]any),
			Name:  function.Item.Name,
			Type:  "tool_use",
		}
		blockEvent.ContentBlock = mustMarshal(block)
		writeEvent(stream, eventType, blockEvent)

		eventType = "content_block_delta"
		jsonDelta := map[string]string{
			"delta": function.Item.Arguments,
			"type":  "input_json_delta",
		}
		contentEvent := claudemodel.MessageContentBlockEvent{
			Type:  eventType,
			Index: data.CurrentOutputIndex,
			Delta: nil,
		}
		contentEvent.Delta = mustMarshal(jsonDelta)
		writeEvent(stream, eventType, contentEvent)

		contentBlock = &claudemodel.MessageContentBlockEvent{
			Type:  "content_block_stop",
			Index: data.CurrentOutputIndex,
		}
		writeEvent(stream, "content_block_stop", mustMarshal(contentBlock))
		data.CurrentOutputIndex++
	}
}

func resetFunctionCallState(data *claudeState) {
	data.ActiveFunctionCalls = make(map[int]*functionCallState)
	data.FunctionCallOrder = nil
}

func startChunkState(state CompletionChunkType, stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	switch state {
	case CompletionChunkTypeMessage:
		startMessageState(stream, choice, data)
	case CompletionChunkTypeRefusal:
		// do nothing
	case CompletionChunkTypeReasoning:
		startReasoningState(stream, choice, data)
	case CompletionChunkTypeFunctionCall:
		startFunctionCallState(stream, choice, data)
	}
}

func startMessageState(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	eventType := "content_block_start"
	blockEvent := claudemodel.MessageContentBlockEvent{
		Type:         eventType,
		Index:        data.CurrentOutputIndex,
		ContentBlock: nil,
		Delta:        nil,
	}
	block := claudemodel.TextBlock{
		Text: "",
		Type: "text",
	}
	blockEvent.ContentBlock = mustMarshal(block)

	writeEvent(stream, eventType, blockEvent)
	appendMessageDelta(stream, choice, data)
}

func appendMessageDelta(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	if choice == nil || choice.Delta.Content == "" {
		return
	}
	eventType := "content_block_delta"
	contentEvent := claudemodel.MessageContentBlockEvent{
		Type:  eventType,
		Index: data.CurrentOutputIndex,
		Delta: nil,
	}
	block := claudemodel.TextBlock{
		Text: choice.Delta.Content,
		Type: "text_delta",
	}
	contentEvent.Delta = mustMarshal(block)
	writeEvent(stream, eventType, contentEvent)
}

func startReasoningState(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	eventType := "content_block_start"
	blockEvent := claudemodel.MessageContentBlockEvent{
		Type:         eventType,
		Index:        data.CurrentOutputIndex,
		ContentBlock: nil,
		Delta:        nil,
	}
	block := claudemodel.ThinkingBlock{
		Thinking: "",
		Type:     "thinking",
	}
	blockEvent.ContentBlock = mustMarshal(block)
	writeEvent(stream, eventType, blockEvent)

	appendReasoningSummaryDelta(stream, choice, data)
}

func appendReasoningSummaryDelta(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	if choice == nil || choice.Delta.ReasoningContent == "" {
		return
	}
	eventType := "content_block_delta"
	contentEvent := claudemodel.MessageContentBlockEvent{
		Type:  eventType,
		Index: data.CurrentOutputIndex,
		Delta: nil,
	}
	block := claudemodel.ThinkingBlock{
		Thinking: choice.Delta.ReasoningContent,
		Type:     "thinking_delta",
	}
	contentEvent.Delta = mustMarshal(block)
	writeEvent(stream, eventType, contentEvent)
}

func startFunctionCallState(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	resetFunctionCallState(data)
	if choice == nil {
		return
	}
	eventType := "content_block_start"
	blockEvent := claudemodel.MessageContentBlockEvent{
		Type:         eventType,
		Index:        data.CurrentOutputIndex,
		ContentBlock: nil,
		Delta:        nil,
	}
	for _, tool := range choice.Delta.ToolCalls {
		appendFunctionCallDelta(stream, data, tool)
	}
	index := data.FunctionCallOrder[0]
	function := data.ActiveFunctionCalls[index]
	if function == nil {
		fmt.Println("function == nil!!!!!!!!!")
		return
	}
	block := claudemodel.ToolUseBlock{
		ID:    function.Item.CallID,
		Input: make(map[string]any),
		Name:  function.Item.Name,
		Type:  "tool_use",
	}

	// start
	blockEvent.ContentBlock = mustMarshal(block)
	writeEvent(stream, eventType, blockEvent)

	// 在这里处理增量event
	appendFunctionCallsDelta(stream, choice, data)
}

func appendFunctionCallsDelta(stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	if choice == nil {
		return
	}
	firstIndex := data.FunctionCallOrder[0]
	for _, tool := range choice.Delta.ToolCalls {
		// 只是更新状态
		appendFunctionCallDelta(stream, data, tool)
		// 返回第一个函数调用的事件
		if tool.Index == firstIndex {
			eventType := "content_block_delta"
			jsonDelta := map[string]string{
				"partial_json": tool.Function.Arguments,
				"type":         "input_json_delta",
			}
			contentEvent := claudemodel.MessageContentBlockEvent{
				Type:  eventType,
				Index: data.CurrentOutputIndex,
				Delta: nil,
			}
			contentEvent.Delta = mustMarshal(jsonDelta)
			writeEvent(stream, eventType, contentEvent)
		}
	}
}

func appendFunctionCallDelta(stream *types.ComplexEventStream, data *claudeState, tool oaimodel.ChoiceDeltaToolCall) {
	// 不做任何event-stream处理，只是更新存储状态
	// ensureFunctionCallState 拿到函数调用状态
	state := ensureFunctionCallState(stream, data, tool)
	if tool.Function != nil && state.Item.Name == "" {
		state.Item.Name = tool.Function.Name
	}
	if tool.Function == nil || tool.Function.Arguments == "" {
		return
	}
	state.Item.Arguments += tool.Function.Arguments
}

func ensureFunctionCallState(stream *types.ComplexEventStream, data *claudeState, tool oaimodel.ChoiceDeltaToolCall) *functionCallState {
	if data.ActiveFunctionCalls == nil {
		data.ActiveFunctionCalls = make(map[int]*functionCallState)
	}
	if state, ok := data.ActiveFunctionCalls[tool.Index]; ok {
		if state.Item.Name == "" && tool.Function != nil {
			state.Item.Name = tool.Function.Name
		}
		return state
	}

	callID := tool.Id
	if callID == "" {
		callID = "call_" + RandomStringId(16)
	}
	name := ""
	if tool.Function != nil {
		name = tool.Function.Name
	}
	state := &functionCallState{
		Item: oaimodel.ResponseFunctionCall{
			Type:      string(CompletionChunkTypeFunctionCall),
			Arguments: "",
			CallID:    callID,
			Name:      name,
			Id:        "fc_" + RandomStringId(48),
		},
		OutputIndex: data.CurrentOutputIndex,
	}
	//data.CurrentOutputIndex++
	data.ActiveFunctionCalls[tool.Index] = state
	data.FunctionCallOrder = append(data.FunctionCallOrder, tool.Index)
	//writeOutputItemAdded(stream, data, state.OutputIndex, state.Item)
	return state
}

func appendChunkDelta(state CompletionChunkType, stream *types.ComplexEventStream, choice *oaimodel.StreamChoiceDelta, data *claudeState) {
	switch state {
	case CompletionChunkTypeMessage:
		appendMessageDelta(stream, choice, data)
	case CompletionChunkTypeRefusal:
		// do nothing
	case CompletionChunkTypeReasoning:
		appendReasoningSummaryDelta(stream, choice, data)
	case CompletionChunkTypeFunctionCall:
		appendFunctionCallsDelta(stream, choice, data)
	}
}
