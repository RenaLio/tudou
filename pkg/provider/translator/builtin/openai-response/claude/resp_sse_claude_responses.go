package claude

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/translator/helpers"

	"github.com/RenaLio/tudou/pkg/provider/common"
	claudemodel "github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	oaimodel "github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
	"github.com/RenaLio/tudou/pkg/provider/types"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

func ConvertClaudeStreamToOpenAIResponses(ctx context.Context, req *types.Request, resp *types.Response) (*types.Response, error) {
	cp := common.CloneResponse(resp)
	st := types.NewComplexEventStream(ctx, resp.Stream)
	cp.Stream = st
	cp.IsStream = true
	cp.Format = types.FormatOpenAIResponses
	go runClaudeToResponsesStream(st, req)
	return cp, nil
}

type responseToolState struct {
	outputIndex int
	itemId      string
	callId      string
	name        string
	args        string
}
type responseTextState struct {
	index   int
	itemID  string
	text    string
	started bool
}
type oaiResponsesState struct {
	seq      int
	curIndex int
	usage    *oaimodel.ResponseUsage

	text        responseTextState
	summaryText responseTextState

	tools              map[int]*responseToolState
	effectiveToolIndex int
	outputs            []json.RawMessage

	stopReason   string
	curBlockType string
}

func runClaudeToResponsesStream(st *types.ComplexEventStream, req *types.Request) {
	defer st.CloseCh()

	var srcReq oaimodel.CreateResponseRequest
	_ = common.UnmarshalJSON(req.Payload, &srcReq)

	response := oaimodel.Response{
		CreatedAt:            time.Now().Unix(),
		Instructions:         mustMarshal(srcReq.Instructions),
		Metadata:             srcReq.Metadata,
		Model:                srcReq.Model,
		Object:               "response",
		ParallelToolCalls:    srcReq.ParallelToolCalls,
		Temperature:          srcReq.Temperature,
		ToolChoice:           srcReq.ToolChoice,
		Tools:                srcReq.Tools,
		TopP:                 srcReq.TopP,
		Background:           srcReq.Background,
		CompletedAt:          0,
		MaxOutputTokens:      srcReq.MaxOutputTokens,
		MaxToolCalls:         srcReq.MaxToolCalls,
		PreviousResponseID:   srcReq.PreviousResponseID,
		Prompt:               srcReq.Prompt,
		PromptCacheKey:       srcReq.PromptCacheKey,
		PromptCacheRetention: srcReq.PromptCacheRetention,
		Reasoning:            srcReq.Reasoning,
		SafetyIdentifier:     srcReq.SafetyIdentifier,
		ServiceTier:          srcReq.ServiceTier,
		Text:                 srcReq.Text,
		TopLogprobs:          srcReq.TopLogprobs,
		Truncation:           srcReq.Truncation,

		Error:             nil,
		IncompleteDetails: nil,

		Id:     "",
		Status: "",
		Output: nil,
		Usage:  nil,
	}

	state := &oaiResponsesState{
		tools:              map[int]*responseToolState{},
		usage:              &oaimodel.ResponseUsage{},
		outputs:            make([]json.RawMessage, 0),
		effectiveToolIndex: -1,
	}

	for {
		select {
		case <-st.Done():
			return
		default:
		}
		ev, err := st.Pull()
		if err != nil {
			if err == io.EOF {
				return
			}
			st.SetErr(err)
			plog.Debug("recv error", "err", err)
			return
		}

		chunk := ev.Content
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

		switch getType(json.RawMessage(line)) {
		case "message_start":
			msgId := gjson.Get(line, "message.id").String()
			response.Id = firstNonEmpty(msgId, RandomId())
			response.CreatedAt = time.Now().Unix()
			var usage claudemodel.Usage
			_ = json.Unmarshal(json.RawMessage(gjson.Get(line, "message.usage").Raw), &usage)
			state.usage = helpers.ClaudeUsageToResponseUsage(&usage)
			emitResponsesEvent(st, "response.created", oaimodel.ResponseCreatedEvent{Type: "response.created", SequenceNumber: nextSeq(state), Response: &response})
			emitResponsesEvent(st, "response.in_progress", oaimodel.ResponseInProgressEvent{Type: "response.in_progress", SequenceNumber: nextSeq(state), Response: &response})
		case "message_delta":
			var usage claudemodel.Usage
			_ = json.Unmarshal(json.RawMessage(gjson.Get(line, "usage").Raw), &usage)
			respUsage := helpers.ClaudeUsageToResponseUsage(&usage)
			state.usage = helpers.MergeResponseUsage(state.usage, respUsage)
			response.Usage = state.usage
			stopReason := gjson.Get(line, "delta.stop_reason").String()
			state.stopReason = stopReason
		case "message_stop":
			response.CompletedAt = time.Now().Unix()
			response.Output = state.outputs
			response.Usage = state.usage
			response.Status = "completed"
			emitResponsesEvent(st, "response.completed", oaimodel.ResponseCompletedEvent{Type: "response.completed", SequenceNumber: nextSeq(state), Response: &response})
			return
		case "content_block_start":
			// output_item_add
			// content_part_add | ...
			outputAddEvent := oaimodel.ResponseOutputItemAddedEvent{
				Item:           nil,
				OutputIndex:    state.curIndex,
				SequenceNumber: nextSeq(state),
				Type:           "response.output_item.added",
			}
			switch gjson.Get(line, "content_block.type").String() {
			case "text":
				// reset
				state.text = responseTextState{}
				state.curBlockType = "text"
				// start
				itemId := "msg_" + RandomId()
				textOutputItem := oaimodel.ResponseOutputMessage{
					Type:    "message",
					Id:      itemId,
					Content: make(json.RawMessage, 0),
					Role:    "assistant",
					Status:  "in_progress",
				}
				outputAddEvent.Item = mustMarshal(textOutputItem)
				emitResponsesEvent(st, "response.output_item.added", outputAddEvent)
				state.text.itemID = itemId
				state.text.started = true
				state.text.text = ""
				state.text.index = state.curIndex
				contentPartAddEvent := oaimodel.ResponseContentPartAddedEvent{
					ContentIndex:   0,
					ItemID:         itemId,
					OutputIndex:    state.curIndex,
					Part:           mustMarshal(oaimodel.ResponseOutputText{Type: "output_text", Text: ""}),
					SequenceNumber: nextSeq(state),
					Type:           "response.content_part.added",
				}
				emitResponsesEvent(st, "response.content_part.added", contentPartAddEvent)
			case "thinking":
				// reset
				state.summaryText = responseTextState{}
				state.curBlockType = "thinking"
				// start
				itemId := "rs_" + RandomId()
				reasoningItem := oaimodel.ResponseReasoningItem{
					Type:    "reasoning",
					Id:      itemId,
					Summary: make([]oaimodel.TypeTextObject, 0),
					Status:  "in_progress",
				}
				outputAddEvent.Item = mustMarshal(reasoningItem)
				emitResponsesEvent(st, "response.output_item.added", outputAddEvent)
				state.summaryText.itemID = itemId
				state.summaryText.started = true
				state.summaryText.text = ""
				state.summaryText.index = state.curIndex
				contentPartAddEvent := oaimodel.ResponseReasoningSummaryPartAddedEvent{
					ItemID:         itemId,
					OutputIndex:    state.curIndex,
					Part:           &oaimodel.TypeTextObject{Type: "summary_text", Text: ""},
					SequenceNumber: nextSeq(state),
					SummaryIndex:   0,
					Type:           "response.summary_part.added",
				}
				emitResponsesEvent(st, "response.summary_part.added", contentPartAddEvent)
			case "tool_use":
				// reset
				state.effectiveToolIndex = -1
				state.curBlockType = "tool_use"
				// start
				itemId := "fc_" + RandomId()
				toolCallId := gjson.Get(line, "content_block.id").String()
				state.effectiveToolIndex = state.curIndex
				toolName := gjson.Get(line, "content_block.name").String()
				state.tools[state.curIndex] = &responseToolState{
					outputIndex: state.curIndex,
					itemId:      itemId,
					callId:      toolCallId,
					name:        toolName,
					args:        "",
				}
				fcItem := oaimodel.ResponseFunctionCall{
					Type:      "function_call",
					Arguments: "",
					CallID:    toolCallId,
					Name:      toolName,
					Id:        itemId,
					Status:    "in_progress",
				}
				outputAddEvent.Item = mustMarshal(fcItem)
				emitResponsesEvent(st, "response.output_item.added", outputAddEvent)
			}
		case "content_block_delta":
			switch gjson.Get(line, "delta.type").String() {
			case "text_delta":
				textDelta := gjson.Get(line, "delta.text").String()
				state.text.text += textDelta
				outputTextDeltaEvent := oaimodel.ResponseOutputTextDeltaEvent{
					ContentIndex:   0,
					Delta:          textDelta,
					ItemID:         state.text.itemID,
					LogProbs:       nil,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					Type:           "response.output_text.delta",
				}
				emitResponsesEvent(st, "response.output_text.delta", outputTextDeltaEvent)
			case "input_json_delta":
				inputJsonDelta := gjson.Get(line, "delta.partial_json").String()
				state.tools[state.curIndex].args += inputJsonDelta
				fcArgsDeltaEvent := oaimodel.ResponseFunctionCallArgumentsDeltaEvent{
					Delta:          inputJsonDelta,
					ItemID:         state.tools[state.curIndex].itemId,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					Type:           "response.function_call_arguments.delta",
				}
				emitResponsesEvent(st, "response.function_call_arguments.delta", fcArgsDeltaEvent)
			case "thinking_delta":
				summaryTextDelta := gjson.Get(line, "delta.thinking").String()
				state.summaryText.text += summaryTextDelta
				summaryPartDeltaEvent := oaimodel.ResponseReasoningSummaryTextDeltaEvent{
					Delta:          summaryTextDelta,
					ItemID:         state.summaryText.itemID,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					SummaryIndex:   0,
					Type:           "response.reasoning_summary_text.delta",
				}
				emitResponsesEvent(st, "response.reasoning_summary_text.delta", summaryPartDeltaEvent)
			}
		case "content_block_stop":
			// output_text_done
			// content_part_done | ...
			// output_item_done | ...
			outputDoneEvent := oaimodel.ResponseOutputItemDoneEvent{
				Item:           nil,
				OutputIndex:    state.curIndex,
				SequenceNumber: 0,
				Type:           "response.output_item.done",
			}
			contentPartDoneEvent := oaimodel.ResponseContentPartDoneEvent{
				ContentIndex:   0,
				ItemID:         state.text.itemID,
				OutputIndex:    state.curIndex,
				Part:           nil,
				SequenceNumber: 0,
				Type:           "response.content_part.done",
			}
			switch state.curBlockType {
			case "text":
				textDone := oaimodel.ResponseOutputTextDoneEvent{
					ContentIndex:   0,
					ItemID:         state.text.itemID,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					Text:           state.text.text,
					Type:           "response.output_text.done",
				}
				emitResponsesEvent(st, "response.output_text.done", textDone)
				outputText := oaimodel.ResponseOutputText{Type: "output_text", Text: state.text.text}
				contentPartDoneEvent.Part = mustMarshal(outputText)
				contentPartDoneEvent.SequenceNumber = nextSeq(state)
				emitResponsesEvent(st, "response.content_part.done", contentPartDoneEvent)
				messageOutput := oaimodel.ResponseOutputMessage{
					Type:    "message",
					Id:      state.text.itemID,
					Content: mustMarshal([]oaimodel.ResponseOutputText{outputText}),
					Role:    "assistant",
					Status:  "completed",
				}
				state.outputs = append(state.outputs, mustMarshal(messageOutput))
				outputDoneEvent.Item = mustMarshal(messageOutput)
				outputDoneEvent.SequenceNumber = nextSeq(state)
				emitResponsesEvent(st, "response.output_item.done", outputDoneEvent)
				state.curIndex++
			case "thinking":
				summaryTextDone := oaimodel.ResponseReasoningSummaryTextDoneEvent{
					ItemID:         state.summaryText.itemID,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					SummaryIndex:   0,
					Text:           state.summaryText.text,
					Type:           "esponse.reasoning_summary_text.done",
				}
				emitResponsesEvent(st, "response.reasoning_summary_text.done", summaryTextDone)
				summary := oaimodel.TypeTextObject{Type: "summary_text", Text: state.summaryText.text}
				//contentPartDoneEvent.Part = mustMarshal(summary)
				//contentPartDoneEvent.SequenceNumber = nextSeq(state)
				summaryPartDoneEvent := oaimodel.ResponseReasoningSummaryPartDoneEvent{
					ItemID:         state.summaryText.itemID,
					OutputIndex:    state.curIndex,
					Part:           &summary,
					SequenceNumber: nextSeq(state),
					SummaryIndex:   0,
					Type:           "response.summary_part.done",
				}
				emitResponsesEvent(st, "response.summary_part.done", summaryPartDoneEvent)
				reasoningOutput := oaimodel.ResponseReasoningItem{
					Type:    "reasoning",
					Id:      state.summaryText.itemID,
					Summary: []oaimodel.TypeTextObject{summary},
					Status:  "completed",
				}
				state.outputs = append(state.outputs, mustMarshal(reasoningOutput))
				outputDoneEvent.Item = mustMarshal(reasoningOutput)
				outputDoneEvent.SequenceNumber = nextSeq(state)
				emitResponsesEvent(st, "response.output_item.done", outputDoneEvent)
				state.curIndex++
			case "tool_use":
				fcArgsDone := oaimodel.ResponseFunctionCallArgumentsDoneEvent{
					Arguments:      state.tools[state.curIndex].args,
					ItemID:         state.tools[state.curIndex].itemId,
					Name:           state.tools[state.curIndex].name,
					OutputIndex:    state.curIndex,
					SequenceNumber: nextSeq(state),
					Type:           "response.function_call_arguments.done",
				}
				emitResponsesEvent(st, "response.function_call_arguments.done", fcArgsDone)
				fcOutput := oaimodel.ResponseFunctionCall{
					Type:      "function_call",
					Arguments: state.tools[state.curIndex].args,
					CallID:    state.tools[state.curIndex].callId,
					Name:      state.tools[state.curIndex].name,
					Id:        state.tools[state.curIndex].itemId,
					Status:    "completed",
				}
				state.outputs = append(state.outputs, mustMarshal(fcOutput))
				outputDoneEvent.Item = mustMarshal(fcOutput)
				outputDoneEvent.SequenceNumber = nextSeq(state)
				emitResponsesEvent(st, "response.output_item.done", outputDoneEvent)
				state.curIndex++
			}
		}
	}
}

func finishAndReset(state *oaiResponsesState) {
	state.curBlockType = ""
	state.text = responseTextState{}
	state.summaryText = responseTextState{}
	state.tools = map[int]*responseToolState{}
	state.effectiveToolIndex = -1
}

func emitResponsesEvent(st *types.ComplexEventStream, event string, data any) {
	var err error
	b := mustMarshal(data)
	plog.Debug("trans_push_line: ", string(b))
	err = st.Send([]byte("event: " + event + "\n"))
	if err != nil {
		st.SetErr(err)
		return
	}
	err = st.Send([]byte("data: " + string(b) + "\n\n"))
	if err != nil {
		st.SetErr(err)
	}
	return
}
func nextSeq(state *oaiResponsesState) int { state.seq++; return state.seq }
