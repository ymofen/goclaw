package openai

import (
	"goclaw/pkg/model"
)

// SSEStreamProcessor collects parsed SSE chunks, emits lightweight SSE events,
// and can build the final normalized message list for the full stream.
type SSEStreamProcessor struct {
	onMessage   func(model.SSEMessage)
	chunks      []streamChunk
	currentKind model.MessageKind
	segmentOpen bool
}

// NewSSEStreamProcessor creates a processor for chunk-by-chunk SSE handling.
func NewSSEStreamProcessor(onMessage func(model.SSEMessage)) *SSEStreamProcessor {
	return &SSEStreamProcessor{
		onMessage: onMessage,
		chunks:    make([]streamChunk, 0, 16),
	}
}

// AddChunk records one parsed chunk and emits lightweight streaming content.
func (p *SSEStreamProcessor) AddChunk(chunk streamChunk) {
	for _, choice := range chunk.Choices {
		if choice.Delta.ReasoningContent != "" {
			p.push(model.KindReasoning, choice.Delta.ReasoningContent)
		}
		if choice.Delta.Content != "" {
			p.push(model.KindText, choice.Delta.Content)
		}
		if choice.Message.Content != "" {
			p.push(model.KindText, choice.Message.Content)
		}
		if len(choice.Delta.ToolCalls) > 0 {
			p.push(model.KindToolCall, "")
		}
	}
	p.chunks = append(p.chunks, chunk)
}

// BuildMessages converts all collected chunks into normalized messages and emits
// a final stop SSE event.
func (p *SSEStreamProcessor) BuildMessages() ([]model.Message, string) {
	p.emitCurrentEnd()
	messages, finishReason := buildMessagesFromChunks(p.chunks, nil)
	p.emitStop(finishReason)
	return messages, finishReason
}

func (p *SSEStreamProcessor) push(kind model.MessageKind, content string) {
	if !p.segmentOpen {
		p.segmentOpen = true
		p.currentKind = kind
		p.emit(kind, content, model.SSEStepStart)
		return
	}
	if p.currentKind != kind {
		p.emitCurrentEnd()
		p.segmentOpen = true
		p.currentKind = kind
		p.emit(kind, content, model.SSEStepStart)
		return
	}
	p.emit(kind, content, 0)
}

func (p *SSEStreamProcessor) emit(kind model.MessageKind, content string, step uint8) {
	if p.onMessage == nil {
		return
	}
	p.onMessage(model.SSEMessage{Kind: kind, Content: content, Step: step})
}

func (p *SSEStreamProcessor) emitStop(finishReason string) {
	if p.onMessage == nil {
		return
	}
	p.onMessage(model.SSEMessage{Kind: model.KindStop, Content: finishReason, Step: model.SSEStepStart | model.SSEStepEnd})
}

func (p *SSEStreamProcessor) emitCurrentEnd() {
	if !p.segmentOpen {
		return
	}
	p.emit(p.currentKind, "", model.SSEStepEnd)
	p.segmentOpen = false
	p.currentKind = ""
}
