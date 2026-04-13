package openai

import "goclaw/pkg/model"

// SSEStreamProcessor collects parsed SSE chunks, emits lightweight SSE events,
// and can build the final normalized message list for the full stream.
type SSEStreamProcessor struct {
	onMessage func(model.SSEMessage)
	chunks    []streamChunk
	pending   *model.SSEMessage
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
	}
	p.chunks = append(p.chunks, chunk)
}

// BuildMessages converts all collected chunks into normalized messages and emits
// a final stop SSE event.
func (p *SSEStreamProcessor) BuildMessages() ([]model.Message, string) {
	p.flushPending(true)
	messages, finishReason := buildMessagesFromChunks(p.chunks, nil)
	p.emitStop(finishReason)
	return messages, finishReason
}

func (p *SSEStreamProcessor) push(kind model.MessageKind, content string) {
	if content == "" {
		return
	}
	if p.pending != nil && p.pending.Kind != kind {
		p.flushPending(true)
	}
	if p.pending != nil && p.pending.Kind == kind {
		p.flushPending(false)
	}
	p.pending = &model.SSEMessage{Kind: kind, Content: content}
}

func (p *SSEStreamProcessor) emitStop(finishReason string) {
	if p.onMessage == nil {
		return
	}
	p.onMessage(model.SSEMessage{Kind: model.KindStop, Content: finishReason, DoneFlag: true})
}

func (p *SSEStreamProcessor) flushPending(done bool) {
	if p.pending == nil {
		return
	}
	msg := *p.pending
	msg.DoneFlag = done
	if p.onMessage != nil {
		p.onMessage(msg)
	}
	p.pending = nil
}
