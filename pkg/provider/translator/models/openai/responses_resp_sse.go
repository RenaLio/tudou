package openai

import "encoding/json"

type (
	ResponseCreatedEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseInProgressEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseQueuedEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseErrorEvent struct {
		Code           string  `json:"code"`
		Message        string  `json:"message"`
		Param          *string `json:"param"`
		SequenceNumber int     `json:"sequence_number"`
		Type           string  `json:"type"`
	}
	ResponseFailedEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseIncompleteEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseCompletedEvent struct {
		Response       *Response `json:"response"`
		SequenceNumber int       `json:"sequence_number"`
		Type           string    `json:"type"`
	}
	ResponseContentPartAddedEvent struct {
		ContentIndex   int             `json:"content_index"`
		ItemID         string          `json:"item_id"`
		OutputIndex    int             `json:"output_index"`
		Part           json.RawMessage `json:"part"` // ResponseOutputText |  ResponseOutputRefusal | openai.OnlyTypeTextObject
		SequenceNumber int             `json:"sequence_number"`
		Type           string          `json:"type"`
	}
	ResponseContentPartDoneEvent struct {
		ContentIndex   int             `json:"content_index"`
		ItemID         string          `json:"item_id"`
		OutputIndex    int             `json:"output_index"`
		Part           json.RawMessage `json:"part"` // ResponseOutputText |  ResponseOutputRefusal | openai.OnlyTypeTextObject
		SequenceNumber int             `json:"sequence_number"`
		Type           string          `json:"type"`
	}
)
type (
	ResponseOutputTextDeltaEvent struct {
		ContentIndex   int               `json:"content_index"`
		Delta          string            `json:"delta"`
		ItemID         string            `json:"item_id"`
		LogProbs       []json.RawMessage `json:"logprobs,omitempty"`
		OutputIndex    int               `json:"output_index"`
		SequenceNumber int               `json:"sequence_number"`
		Type           string            `json:"type"`
	}
	ResponseOutputTextDoneEvent struct {
		ContentIndex   int               `json:"content_index"`
		ItemID         string            `json:"item_id"`
		LogProbs       []json.RawMessage `json:"logprobs,omitempty"`
		OutputIndex    int               `json:"output_index"`
		SequenceNumber int               `json:"sequence_number"`
		Text           string            `json:"text"`
		Type           string            `json:"type"`
	}
	ResponseTextDeltaEvent                 = ResponseOutputTextDeltaEvent
	ResponseTextDoneEvent                  = ResponseOutputTextDoneEvent
	ResponseOutputTextAnnotationAddedEvent struct {
		Annotation      json.RawMessage `json:"annotation"` // Annotation
		AnnotationIndex int             `json:"annotation_index"`
		ContentIndex    int             `json:"content_index"`
		ItemID          string          `json:"item_id"`
		OutputIndex     int             `json:"output_index"`
		SequenceNumber  int             `json:"sequence_number"`
		Type            string          `json:"type"`
	}
)

type (
	ResponseOutputItemAddedEvent struct {
		Item           json.RawMessage `json:"item"` //	ResponseOutputItem
		OutputIndex    int             `json:"output_index"`
		SequenceNumber int             `json:"sequence_number"`
		Type           string          `json:"type"`
	}
	ResponseOutputItemDoneEvent struct {
		Item           json.RawMessage `json:"item"` // ResponseOutputItem
		OutputIndex    int             `json:"output_index"`
		SequenceNumber int             `json:"sequence_number"`
		Type           string          `json:"type"`
	}
)

type (
	ResponseReasoningSummaryPartAddedEvent struct {
		ItemID         string          `json:"item_id"`
		OutputIndex    int             `json:"output_index"`
		Part           *TypeTextObject `json:"part"`
		SequenceNumber int             `json:"sequence_number"`
		SummaryIndex   int             `json:"summary_index"`
		Type           string          `json:"type"`
	}
	ResponseReasoningSummaryPartDoneEvent struct {
		ItemID         string          `json:"item_id"`
		OutputIndex    int             `json:"output_index"`
		Part           *TypeTextObject `json:"part"`
		SequenceNumber int             `json:"sequence_number"`
		SummaryIndex   int             `json:"summary_index"`
		Type           string          `json:"type"`
	}
	ResponseReasoningSummaryTextDeltaEvent struct {
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		SummaryIndex   int    `json:"summary_index"`
		Type           string `json:"type"`
	}
	ResponseReasoningSummaryTextDoneEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		SummaryIndex   int    `json:"summary_index"`
		Text           string `json:"text"`
		Type           string `json:"type"`
	}
	ResponseReasoningTextDeltaEvent struct {
		ContentIndex   int    `json:"content_index"`
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseReasoningTextDoneEvent struct {
		ContentIndex   int    `json:"content_index"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Text           string `json:"text"`
		Type           string `json:"type"`
	}
)

type (
	ResponseFunctionCallArgumentsDeltaEvent struct {
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseFunctionCallArgumentsDoneEvent struct {
		Arguments      string `json:"arguments"`
		ItemID         string `json:"item_id"`
		Name           string `json:"name"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseCustomToolCallInputDeltaEvent struct {
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseCustomToolCallInputDoneEvent struct {
		Input          string `json:"input"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)

type (
	ResponseAudioDeltaEvent struct {
		// delta: string
		// A chunk of Base64 encoded response audio bytes.
		Delta string `json:"delta"`

		// sequence_number: number
		// A sequence number for this chunk of the stream response.
		SequenceNumber int `json:"sequence_number"`

		// type: "response.audio.delta"
		// The type of the event. Always response.audio.delta.
		Type string `json:"type"`
	}
	ResponseAudioDoneEvent struct {
		// sequence_number: number
		// The sequence number of the delta.
		SequenceNumber int `json:"sequence_number"`

		// type: "response.audio.done"
		// The type of the event. Always response.audio.done.
		Type string `json:"type"`
	}
	ResponseAudioTranscriptDeltaEvent struct {
		// delta: string
		// The partial transcript of the audio response.
		Delta string `json:"delta"`

		// sequence_number: number
		// The sequence number of this event.
		SequenceNumber int `json:"sequence_number"`

		// type: "response.audio.transcript.delta"
		// The type of the event. Always response.audio.transcript.delta.
		Type string `json:"type"`
	}
	ResponseAudioTranscriptDoneEvent struct {
		// sequence_number: number
		// The sequence number of this event.
		SequenceNumber int `json:"sequence_number"`

		// type: "response.audio.transcript.done"
		// The type of the event. Always response.audio.transcript.done.
		Type string `json:"type"`
	}
)

type (
	ResponseCodeInterpreterCallCodeDeltaEvent struct {
		// delta: string
		// The partial code snippet being streamed by the code interpreter.
		Delta string `json:"delta"`

		// item_id: string
		// The unique identifier of the code interpreter tool call item.
		ItemID string `json:"item_id"`

		// output_index: number
		// The index of the output item in the response for which the code is being streamed.
		OutputIndex int `json:"output_index"`

		// sequence_number: number
		// The sequence number of this event, used to order streaming events.
		SequenceNumber int `json:"sequence_number"`

		// type: "response.code_interpreter_call_code.delta"
		// The type of the event. Always response.code_interpreter_call_code.delta.
		Type string `json:"type"`
	}
	// ResponseCodeInterpreterCallCodeDoneEvent 表示代码解释器代码片段完成事件
	// 当代码解释器流式传输的代码片段最终确定时发出此事件
	ResponseCodeInterpreterCallCodeDoneEvent struct {
		// Code 最终确定的代码片段
		// The final code snippet output by the code interpreter.
		Code string `json:"code"`

		// ItemID 代码解释器工具调用项的唯一标识符
		// The unique identifier of the code interpreter tool call item.
		ItemID string `json:"item_id"`

		// OutputIndex 响应中输出项的索引
		// The index of the output item in the response for which the code is finalized.
		OutputIndex int `json:"output_index"`

		// SequenceNumber 事件序列号，用于排序流式事件
		// The sequence number of this event, used to order streaming events.
		SequenceNumber int `json:"sequence_number"`

		// Type 事件类型，固定值为 "response.code_interpreter_call_code.done"
		// The type of the event. Always response.code_interpreter_call_code.done.
		Type string `json:"type"`
	}
	ResponseCodeInterpreterCallInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseCodeInterpreterCallInterpretingEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseCodeInterpreterCallCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)

type (
	ResponseFileSearchCallCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseFileSearchCallInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseFileSearchCallSearchingEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)

type (
	ResponseRefusalDeltaEvent struct {
		ContentIndex   int    `json:"content_index"`
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseRefusalDoneEvent struct {
		ContentIndex   int    `json:"content_index"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		Refusal        string `json:"refusal"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)

type (
	ResponseWebSearchCallInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseWebSearchCallSearchingEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseWebSearchCallCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)

type (
	ResponseImageGenerationCallInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseImageGenerationCallGeneratingEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseImageGenerationCallPartialImageEvent struct {
		ItemID            string `json:"item_id"`
		OutputIndex       int    `json:"output_index"`
		PartialImageB64   string `json:"partial_image_b64"`
		PartialImageIndex int    `json:"partial_image_index"`
		SequenceNumber    int    `json:"sequence_number"`
		Type              string `json:"type"`
	}
	ResponseImageGenerationCallCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseImageGenCallInProgressEvent   = ResponseImageGenerationCallInProgressEvent
	ResponseImageGenCallGeneratingEvent   = ResponseImageGenerationCallGeneratingEvent
	ResponseImageGenCallPartialImageEvent = ResponseImageGenerationCallPartialImageEvent
	ResponseImageGenCallCompletedEvent    = ResponseImageGenerationCallCompletedEvent
)

type (
	ResponseMCPCallArgumentsDeltaEvent struct {
		Delta          string `json:"delta"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPCallArgumentsDoneEvent struct {
		Arguments      string `json:"arguments"`
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPCallInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPCallCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPCallFailedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPListToolsInProgressEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPListToolsCompletedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
	ResponseMCPListToolsFailedEvent struct {
		ItemID         string `json:"item_id"`
		OutputIndex    int    `json:"output_index"`
		SequenceNumber int    `json:"sequence_number"`
		Type           string `json:"type"`
	}
)
