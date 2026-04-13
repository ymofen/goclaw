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

	if len(events) != 3 {
		t.Fatalf("unexpected SSE event count: %d", len(events))
	}
	if events[0].Kind != model.KindReasoning || events[0].Content != "think" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[0].Step != model.SSEStepStart {
		t.Fatalf("reasoning event should be start-only when it begins: %+v", events[0])
	}
	if events[1].Kind != model.KindReasoning || events[1].Content != "" || events[1].Step != model.SSEStepEnd {
		t.Fatalf("unexpected reasoning end marker: %+v", events[1])
	}
	if events[2].Kind != model.KindText || events[2].Content != "hello" || events[2].Step != model.SSEStepStart {
		t.Fatalf("unexpected text start event: %+v", events[2])
	}

	processor.AddChunk(chunk2)

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
	if len(events) != 5 {
		t.Fatalf("unexpected event count after finalize: %d", len(events))
	}
	if events[3].Kind != model.KindText || events[3].Content != "" || events[3].Step != model.SSEStepEnd {
		t.Fatalf("unexpected text end marker: %+v", events[3])
	}
	if events[4].Kind != model.KindStop || events[4].Content != "tool_calls" || events[4].Step != model.SSEStepStart|model.SSEStepEnd {
		t.Fatalf("unexpected stop event: %+v", events[4])
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
