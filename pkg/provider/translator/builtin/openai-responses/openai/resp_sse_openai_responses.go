package openai

import (
	"bytes"
	"context"

	"github.com/RenaLio/tudou/pkg/provider/plog"

	"errors"
	"io"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/common"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
)

func ConvertChatCompletionStreamToResponse(ctx context.Context, req *types.Request, resp *types.Response) *types.Response {
	temp := common.CloneResponse(resp)
	stream := NewComplexEventStream(ctx, temp.Stream)
	temp.Stream = stream
	go func() {
		Worker(stream, req)
	}()
	return temp
}

func WriteEventToStream(stream *ComplexEventStream, eventType string, d any) {
	WriteDataToStream(stream, []byte("event: "+eventType+"\n"))
	eventBytes, _ := common.MarshalJSON(d)
	plog.Debug("trans_push_line: ", string(eventBytes))
	eventItem := "data: " + string(eventBytes) + "\n\n"
	WriteDataToStream(stream, []byte(eventItem))
}

func WriteDataToStream(stream *ComplexEventStream, data []byte) {
	stream.ch <- data
}

func Worker(stream *ComplexEventStream, req *types.Request) {
	defer close(stream.ch)
	getIndex := GetIndex()

	params := new(openai.CreateResponseRequest)
	_ = common.UnmarshalJSON(req.Payload, params)
	respId := "resp_" + RandomStringId(48)
	instructions, _ := common.MarshalJSON(params.Instructions)
	// 类似非流式转换
	// 只不过是分步骤进行的

	// 初始化 Response
	// 补充output
	// 补充usage
	outputResponse := &openai.Response{
		Id:                   respId,
		Object:               "response",
		CreatedAt:            time.Now().Unix(),
		Status:               "in_progress",
		Instructions:         instructions,
		MaxOutputTokens:      params.MaxOutputTokens,
		Model:                params.Model,
		ParallelToolCalls:    params.ParallelToolCalls,
		Temperature:          params.Temperature,
		ToolChoice:           params.ToolChoice,
		Tools:                params.Tools,
		TopP:                 params.TopP,
		MaxToolCalls:         params.MaxToolCalls,
		PreviousResponseID:   params.PreviousResponseID,
		Prompt:               params.Prompt,
		PromptCacheKey:       params.PromptCacheKey,
		PromptCacheRetention: params.PromptCacheRetention,
		Reasoning:            params.Reasoning,
		SafetyIdentifier:     params.SafetyIdentifier,
		ServiceTier:          params.ServiceTier,
		Text:                 params.Text,
		TopLogprobs:          params.TopLogprobs,
		Truncation:           params.Truncation,
	}

	once := new(sync.Once)

	// 补充output
	// 补充usage
	usage := &openai.ResponseUsage{}
	previousState := CompletionChunkNone // 有限状态转换 none output_message output_usage

	data := &InnerData{
		IdxFunc: getIndex,
	}

	for {
		select {
		case <-stream.ctx.Done():
			// 外部调用了 Close()，立刻退出协程。
			// 退出时会触发 defer close(e.ch)，极其安全！
			return
		default:
		}

		event, err := stream.upstream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // 正常结束，触发 defer close
			}
			plog.Error("Recv error", "err", err)
			stream.err = err // 记录底层错误给 Recv 拿去用
			return           // 异常结束，触发 defer close
		}
		once.Do(func() {
			// create
			eventType := "response.created"
			outputResponse.Status = "in_progress"
			createEvent := openai.ResponseCreatedEvent{
				Type:           eventType,
				SequenceNumber: getIndex(),
				Response:       outputResponse,
			}
			WriteEventToStream(stream, eventType, createEvent)

			// in_progress
			eventType = "response.in_progress"
			outputResponse.Status = "in_progress"
			inProgressEvent := openai.ResponseInProgressEvent{
				Type:           eventType,
				SequenceNumber: getIndex(),
				Response:       outputResponse,
			}
			WriteEventToStream(stream, eventType, inProgressEvent)
		})
		chunk := event.Content
		chunk = bytes.TrimSpace(chunk)
		line := string(chunk)

		if len(line) == 0 {
			continue
		}
		line = strings.TrimPrefix(line, "data:")
		line = strings.TrimSpace(line)
		if line == "[DONE]" {
			break
		}

		plog.Debug("trans_pull_line: ", line)

		completionChunk := openai.ChatCompletionStream{}
		if err = common.UnmarshalJSON([]byte(line), &completionChunk); err != nil {
			plog.Error("UnmarshalJSON error", "err", err)
			stream.err = err
			return
		}
		if completionChunk.Usage != nil {
			usage = ChatUsageToResponseUsage(completionChunk.Usage)
		}
		currentState := previousState
		// 判断当前chunk的类型
		if len(completionChunk.Choices) == 0 {
			// 无choices，跳过
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
		// 状态转换 -> 动作
		fn3(previousState, currentState, stream, &choice, data)
		//fn3
		previousState = currentState
	}
	fn3(previousState, CompletionChunkNone, stream, nil, data)

	// usage
	// complete
	eventType := "response.completed"
	//var outputItems []json.RawMessage
	//{
	//	if len(data.ReasoningItem.Id) > 0 {
	//		reasoningBytes, _ := json.Marshal(data.ReasoningItem)
	//		outputItems = append(outputItems, reasoningBytes)
	//	}
	//	if len(data.OutputMessage.Id) > 0 {
	//		messageBytes, _ := json.Marshal(data.OutputMessage)
	//		outputItems = append(outputItems, messageBytes)
	//	}
	//	for _, toolCall := range data.FunctionToolCalls {
	//		toolCallBytes, _ := json.Marshal(toolCall)
	//		outputItems = append(outputItems, toolCallBytes)
	//	}
	//}
	//outputResponse.Output = outputItems
	outputResponse.Output = data.outputItems
	outputResponse.Usage = usage
	outputResponse.Status = "completed"
	completedEvent := openai.ResponseCompletedEvent{
		Type:           eventType,
		SequenceNumber: getIndex(),
		Response:       outputResponse,
	}
	buf, _ := json.Marshal(completedEvent)
	plog.Debug("chat.completion -> responses output", "output", string(buf))
	WriteEventToStream(stream, eventType, completedEvent)
}

func RandomStringId(length int) string {

	//const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)

	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}

	return string(b)
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

func GetIndex() func() int {
	var index int
	return func() int {
		index++
		return index
	}
}

func getCompletionChunkType(choice *openai.StreamChoiceDelta) CompletionChunkType {
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

type functionCallState struct {
	Item        openai.ResponseFunctionCall
	OutputIndex int
}

type InnerData struct {
	OutputMessage         openai.ResponseOutputMessage
	OutputTextContent     openai.ResponseOutputText
	OutPutMessageOutIndex int
	OutputRefusal         openai.ResponseOutputRefusal
	FunctionToolCalls     []openai.ResponseFunctionCall // completed calls, used by response.completed
	// "delta":{"tool_calls":[{"id":"call_33b585a4c0654e0cb3ad6341","index":0,"type":"function","function":{"name":"shell","arguments":"{\""} } ]}
	ActiveFunctionCalls      map[int]*functionCallState // active calls in current function_call state, 考虑多个tool_calls,key-int代表所在的索引index
	FunctionCallOrder        []int                      // first-seen order for active calls，key是所在的位置，value是tool_index
	ReasoningItem            openai.ResponseReasoningItem
	ReasoningItemOutputIndex int
	ReasoningSummaryText     string
	IdxFunc                  func() int
	CurrentOutputIndex       int
	outputItems              []json.RawMessage
}

func marshalRawMessage(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func writeOutputItemAdded(stream *ComplexEventStream, data *InnerData, outputIndex int, item any) {
	eventType := "response.output_item.added"
	event := &openai.ResponseOutputItemAddedEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    outputIndex,
		Item:           marshalRawMessage(item),
	}
	WriteEventToStream(stream, eventType, event)
}

func writeOutputItemDone(stream *ComplexEventStream, data *InnerData, outputIndex int, item any) {
	eventType := "response.output_item.done"
	event := &openai.ResponseOutputItemDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    outputIndex,
		Item:           marshalRawMessage(item),
	}
	data.outputItems = append(data.outputItems, event.Item)
	WriteEventToStream(stream, eventType, event)
}

func writeContentPartAdded(stream *ComplexEventStream, data *InnerData, itemID string, outputIndex int, part any) {
	eventType := "response.content_part.added"
	event := &openai.ResponseContentPartAddedEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		ItemID:         itemID,
		OutputIndex:    outputIndex,
		ContentIndex:   0,
		Part:           marshalRawMessage(part),
	}
	WriteEventToStream(stream, eventType, event)
}

func writeContentPartDone(stream *ComplexEventStream, data *InnerData, itemID string, outputIndex int, part any) {
	eventType := "response.content_part.done"
	event := &openai.ResponseContentPartDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		ItemID:         itemID,
		OutputIndex:    outputIndex,
		ContentIndex:   0,
		Part:           marshalRawMessage(part),
	}
	WriteEventToStream(stream, eventType, event)
}

func writeSummaryPartAdded(stream *ComplexEventStream, data *InnerData, itemID string, outputIndex int, part *openai.TypeTextObject) {
	eventType := "response.summary_part.added"
	event := &openai.ResponseReasoningSummaryPartAddedEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		ItemID:         itemID,
		OutputIndex:    outputIndex,
		SummaryIndex:   0,
		Part:           part,
	}
	WriteEventToStream(stream, eventType, event)
}

func writeSummaryPartDone(stream *ComplexEventStream, data *InnerData, itemID string, outputIndex int, part *openai.TypeTextObject) {
	eventType := "response.summary_part.done"
	event := &openai.ResponseReasoningSummaryPartDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		ItemID:         itemID,
		OutputIndex:    outputIndex,
		SummaryIndex:   0,
		Part:           part,
	}
	WriteEventToStream(stream, eventType, event)
}

func resetMessageState(data *InnerData) {
	data.OutputMessage = openai.ResponseOutputMessage{}
	data.OutputTextContent = openai.ResponseOutputText{}
	data.OutputRefusal = openai.ResponseOutputRefusal{}
	data.OutPutMessageOutIndex = 0
}

func resetReasoningState(data *InnerData) {
	data.ReasoningItem = openai.ResponseReasoningItem{}
	data.ReasoningItemOutputIndex = 0
	data.ReasoningSummaryText = ""
}

func resetFunctionCallState(data *InnerData) {
	data.ActiveFunctionCalls = make(map[int]*functionCallState)
	data.FunctionCallOrder = nil
}

func startMessageState(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	resetMessageState(data)
	data.OutPutMessageOutIndex = data.CurrentOutputIndex
	data.OutputMessage = openai.ResponseOutputMessage{
		Id:   "msg_" + RandomStringId(48),
		Type: string(CompletionChunkTypeMessage),
		Role: "assistant",
	}
	data.OutputTextContent = openai.ResponseOutputText{Type: "output_text"}

	writeOutputItemAdded(stream, data, data.OutPutMessageOutIndex, data.OutputMessage)
	writeContentPartAdded(stream, data, data.OutputMessage.Id, data.OutPutMessageOutIndex, data.OutputTextContent)
	appendMessageDelta(stream, choice, data)
}

func appendMessageDelta(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	if choice == nil || choice.Delta.Content == "" {
		return
	}
	eventType := "response.output_text.delta"
	event := &openai.ResponseOutputTextDeltaEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.OutPutMessageOutIndex,
		ItemID:         data.OutputMessage.Id,
		ContentIndex:   0,
		Delta:          choice.Delta.Content,
	}
	WriteEventToStream(stream, eventType, event)
	data.OutputTextContent.Text += choice.Delta.Content
}

func finishMessageState(stream *ComplexEventStream, data *InnerData) {
	if data.OutputMessage.Id == "" {
		return
	}
	eventType := "response.output_text.done"
	textDone := &openai.ResponseOutputTextDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.OutPutMessageOutIndex,
		ItemID:         data.OutputMessage.Id,
		ContentIndex:   0,
		Text:           data.OutputTextContent.Text,
	}
	WriteEventToStream(stream, eventType, textDone)

	data.OutputMessage.Content = marshalRawMessage([]openai.ResponseOutputText{data.OutputTextContent})
	writeContentPartDone(stream, data, data.OutputMessage.Id, data.OutPutMessageOutIndex, data.OutputTextContent)
	writeOutputItemDone(stream, data, data.OutPutMessageOutIndex, data.OutputMessage)

	data.CurrentOutputIndex++
}

func startRefusalState(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	resetMessageState(data)
	data.OutPutMessageOutIndex = data.CurrentOutputIndex
	data.OutputMessage = openai.ResponseOutputMessage{
		Id:   "msg_refusal_" + RandomStringId(48),
		Type: string(CompletionChunkTypeMessage),
		Role: "assistant",
	}
	data.OutputRefusal = openai.ResponseOutputRefusal{Type: "refusal"}

	writeOutputItemAdded(stream, data, data.OutPutMessageOutIndex, data.OutputMessage)
	writeContentPartAdded(stream, data, data.OutputMessage.Id, data.OutPutMessageOutIndex, data.OutputRefusal)
	appendRefusalDelta(stream, choice, data)
}

func appendRefusalDelta(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	if choice == nil || choice.Delta.Refusal == "" {
		return
	}
	eventType := "response.refusal.delta"
	event := &openai.ResponseRefusalDeltaEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.OutPutMessageOutIndex,
		ItemID:         data.OutputMessage.Id,
		ContentIndex:   0,
		Delta:          choice.Delta.Refusal,
	}
	WriteEventToStream(stream, eventType, event)
	data.OutputRefusal.Refusal += choice.Delta.Refusal
}

func finishRefusalState(stream *ComplexEventStream, data *InnerData) {
	if data.OutputMessage.Id == "" {
		return
	}
	eventType := "response.refusal.done"
	refusalDone := &openai.ResponseRefusalDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.OutPutMessageOutIndex,
		ItemID:         data.OutputMessage.Id,
		ContentIndex:   0,
		Refusal:        data.OutputRefusal.Refusal,
	}
	WriteEventToStream(stream, eventType, refusalDone)

	data.OutputMessage.Content = marshalRawMessage([]openai.ResponseOutputRefusal{data.OutputRefusal})
	writeContentPartDone(stream, data, data.OutputMessage.Id, data.OutPutMessageOutIndex, data.OutputRefusal)
	writeOutputItemDone(stream, data, data.OutPutMessageOutIndex, data.OutputMessage)

	data.CurrentOutputIndex++
}

func startReasoningState(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	resetReasoningState(data)
	data.ReasoningItemOutputIndex = data.CurrentOutputIndex
	data.ReasoningItem = openai.ResponseReasoningItem{
		Id:      "rs_" + RandomStringId(48),
		Type:    string(CompletionChunkTypeReasoning),
		Summary: make([]openai.TypeTextObject, 0),
	}
	emptyPart := openai.TypeTextObject{Type: "summary_text", Text: ""}

	writeOutputItemAdded(stream, data, data.ReasoningItemOutputIndex, data.ReasoningItem)
	//writeContentPartAdded(stream, data, data.ReasoningItem.Id, data.ReasoningItemOutputIndex, emptyPart)
	writeSummaryPartAdded(stream, data, data.ReasoningItem.Id, data.ReasoningItemOutputIndex, &emptyPart)
	appendReasoningSummaryDelta(stream, choice, data)
}

func appendReasoningSummaryDelta(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	if choice == nil || choice.Delta.ReasoningContent == "" {
		return
	}
	eventType := "response.reasoning_summary_text.delta"
	event := &openai.ResponseReasoningSummaryTextDeltaEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.ReasoningItemOutputIndex,
		ItemID:         data.ReasoningItem.Id,
		SummaryIndex:   0,
		Delta:          choice.Delta.ReasoningContent,
	}
	WriteEventToStream(stream, eventType, event)
	data.ReasoningSummaryText += choice.Delta.ReasoningContent
}

func finishReasoningState(stream *ComplexEventStream, data *InnerData) {
	if data.ReasoningItem.Id == "" {
		return
	}
	eventType := "response.reasoning_text.done"
	done := &openai.ResponseReasoningTextDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.ReasoningItemOutputIndex,
		ItemID:         data.ReasoningItem.Id,
		ContentIndex:   0,
		Text:           data.ReasoningSummaryText,
	}
	WriteEventToStream(stream, eventType, done)

	part := openai.TypeTextObject{Type: "reasoning_text", Text: data.ReasoningSummaryText}
	data.ReasoningItem.Content = []openai.TypeTextObject{part}

	writeContentPartDone(stream, data, data.ReasoningItem.Id, data.ReasoningItemOutputIndex, part)
	writeOutputItemDone(stream, data, data.ReasoningItemOutputIndex, data.ReasoningItem)

	data.CurrentOutputIndex++
}

func finishReasoningSummaryState(stream *ComplexEventStream, data *InnerData) {
	if data.ReasoningItem.Id == "" {
		return
	}
	eventType := "response.reasoning_summary_text.done"
	done := &openai.ResponseReasoningSummaryTextDoneEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    data.ReasoningItemOutputIndex,
		ItemID:         data.ReasoningItem.Id,
		SummaryIndex:   0,
		Text:           data.ReasoningSummaryText,
	}
	WriteEventToStream(stream, eventType, done)

	part := openai.TypeTextObject{Type: "summary_text", Text: data.ReasoningSummaryText}
	data.ReasoningItem.Summary = []openai.TypeTextObject{part}

	writeSummaryPartDone(stream, data, data.ReasoningItem.Id, data.ReasoningItemOutputIndex, &part)
	writeOutputItemDone(stream, data, data.ReasoningItemOutputIndex, data.ReasoningItem)

	data.CurrentOutputIndex++
}

func ensureFunctionCallState(stream *ComplexEventStream, data *InnerData, tool openai.ChoiceDeltaToolCall) *functionCallState {
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
		Item: openai.ResponseFunctionCall{
			Type:      string(CompletionChunkTypeFunctionCall),
			Arguments: "",
			CallID:    callID,
			Name:      name,
			Id:        "fc_" + RandomStringId(48),
		},
		OutputIndex: data.CurrentOutputIndex,
	}
	data.CurrentOutputIndex++
	data.ActiveFunctionCalls[tool.Index] = state
	data.FunctionCallOrder = append(data.FunctionCallOrder, tool.Index)
	writeOutputItemAdded(stream, data, state.OutputIndex, state.Item)
	return state
}

func appendFunctionCallDelta(stream *ComplexEventStream, data *InnerData, tool openai.ChoiceDeltaToolCall) {
	state := ensureFunctionCallState(stream, data, tool)
	if tool.Function != nil && state.Item.Name == "" {
		state.Item.Name = tool.Function.Name
	}
	if tool.Function == nil || tool.Function.Arguments == "" {
		return
	}
	eventType := "response.function_call_arguments.delta"
	event := &openai.ResponseFunctionCallArgumentsDeltaEvent{
		Type:           eventType,
		SequenceNumber: data.IdxFunc(),
		OutputIndex:    state.OutputIndex,
		ItemID:         state.Item.Id,
		Delta:          tool.Function.Arguments,
	}
	WriteEventToStream(stream, eventType, event)
	state.Item.Arguments += tool.Function.Arguments
}

func startFunctionCallState(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	resetFunctionCallState(data)
	if choice == nil {
		return
	}
	for _, tool := range choice.Delta.ToolCalls {
		appendFunctionCallDelta(stream, data, tool)
	}
}

func appendFunctionCallsDelta(stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	if choice == nil {
		return
	}
	for _, tool := range choice.Delta.ToolCalls {
		appendFunctionCallDelta(stream, data, tool)
	}
}

func finishFunctionCallState(stream *ComplexEventStream, data *InnerData) {
	if len(data.FunctionCallOrder) == 0 {
		return
	}
	for _, idx := range data.FunctionCallOrder {
		state, ok := data.ActiveFunctionCalls[idx]
		if !ok {
			continue
		}
		eventType := "response.function_call_arguments.done"
		event := &openai.ResponseFunctionCallArgumentsDoneEvent{
			Type:           eventType,
			SequenceNumber: data.IdxFunc(),
			OutputIndex:    state.OutputIndex,
			ItemID:         state.Item.Id,
			Name:           state.Item.Name,
			Arguments:      state.Item.Arguments,
		}
		WriteEventToStream(stream, eventType, event)
	}
	for _, idx := range data.FunctionCallOrder {
		state, ok := data.ActiveFunctionCalls[idx]
		if !ok {
			continue
		}
		state.Item.Status = "completed"
		writeOutputItemDone(stream, data, state.OutputIndex, state.Item)
		data.FunctionToolCalls = append(data.FunctionToolCalls, state.Item)
	}
	resetFunctionCallState(data)
}

func finishChunkState(state CompletionChunkType, stream *ComplexEventStream, data *InnerData) {
	switch state {
	case CompletionChunkTypeMessage:
		finishMessageState(stream, data)
	case CompletionChunkTypeRefusal:
		finishRefusalState(stream, data)
	case CompletionChunkTypeReasoning:
		finishReasoningSummaryState(stream, data)
	case CompletionChunkTypeFunctionCall:
		finishFunctionCallState(stream, data)
	}
}

func startChunkState(state CompletionChunkType, stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	switch state {
	case CompletionChunkTypeMessage:
		startMessageState(stream, choice, data)
	case CompletionChunkTypeRefusal:
		startRefusalState(stream, choice, data)
	case CompletionChunkTypeReasoning:
		startReasoningState(stream, choice, data)
	case CompletionChunkTypeFunctionCall:
		startFunctionCallState(stream, choice, data)
	}
}

func appendChunkDelta(state CompletionChunkType, stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	switch state {
	case CompletionChunkTypeMessage:
		appendMessageDelta(stream, choice, data)
	case CompletionChunkTypeRefusal:
		appendRefusalDelta(stream, choice, data)
	case CompletionChunkTypeReasoning:
		appendReasoningSummaryDelta(stream, choice, data)
	case CompletionChunkTypeFunctionCall:
		appendFunctionCallsDelta(stream, choice, data)
	}
}

func fn3(prevState, currentState CompletionChunkType, stream *ComplexEventStream, choice *openai.StreamChoiceDelta, data *InnerData) {
	if prevState == CompletionChunkNone && currentState == CompletionChunkNone {
		return
	}

	if prevState != currentState {
		finishChunkState(prevState, stream, data)
		if currentState != CompletionChunkNone {
			startChunkState(currentState, stream, choice, data)
		}
		return
	}

	appendChunkDelta(currentState, stream, choice, data)
}
