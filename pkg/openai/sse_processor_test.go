package openai

import (
	"encoding/json"
	"testing"

	"goclaw/pkg/model"
)

func TestSSEStreamProcessor(t *testing.T) {
	var events []model.SSEMessage
	processor := NewSSEStreamProcessor(func(msg model.SSEMessage) {
		events = append(events, msg)
	})

	chunk1 := mustChunk(t, `{"id":"chat-1","created":1,"model":"demo","choices":[{"delta":{"reasoning_content":"think","content":"hello"},"finish_reason":null}]}`)
	chunk2 := mustChunk(t, `{"id":"chat-1","created":1,"model":"demo","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`)

	processor.AddChunk(chunk1)
	processor.AddChunk(chunk2)

	if len(events) != 1 {
		t.Fatalf("unexpected SSE event count: %d", len(events))
	}
	if events[0].Kind != model.KindReasoning || events[0].Content != "think" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if !events[0].DoneFlag {
		t.Fatalf("reasoning event should be marked done when the kind changes: %+v", events[0])
	}

	messages, finishReason := processor.BuildMessages()
	if finishReason != "tool_calls" {
		t.Fatalf("unexpected finish reason: %q", finishReason)
	}
	if len(messages) == 0 {
		t.Fatal("expected normalized messages")
	}
	if messages[len(messages)-1].Kind != model.KindStop {
		t.Fatalf("expected stop message, got: %+v", messages[len(messages)-1])
	}
	if messages[len(messages)-1].Content != "tool_calls" {
		t.Fatalf("unexpected stop content: %q", messages[len(messages)-1].Content)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected event count after finalize: %d", len(events))
	}
	if events[1].Kind != model.KindText || events[1].Content != "hello" || !events[1].DoneFlag {
		t.Fatalf("unexpected text completion event: %+v", events[1])
	}
	if len(events) != 3 {
		t.Fatalf("unexpected event count after finalize: %d", len(events))
	}
	if events[2].Kind != model.KindStop || events[2].Content != "tool_calls" || !events[2].DoneFlag {
		t.Fatalf("unexpected stop event: %+v", events[2])
	}
}

func mustChunk(t *testing.T, raw string) streamChunk {
	t.Helper()
	var chunk streamChunk
	if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
		t.Fatalf("unmarshal chunk: %v", err)
	}
	return chunk
}
