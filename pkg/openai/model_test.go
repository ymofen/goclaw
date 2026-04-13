package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goclaw/pkg/model"
)

func TestSSEParse(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "history_openai_SSE.txt")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open history file: %v", err)
	}
	defer file.Close()

	processor := NewSSEStreamProcessor(func(msg model.SSEMessage) {
		if msg.Step&model.SSEStepStart != 0 {
			fmt.Printf("=== Event Start: %s ===\n", msg.Kind)
		}
		if len(msg.Content) > 0 {
			fmt.Printf("%s", msg.Content)
		} else if msg.Kind == model.KindToolCall {
			fmt.Printf(".")
		}
		if msg.Step&model.SSEStepEnd != 0 {
			fmt.Printf("\n=== Event Done: %s ===\n", msg.Kind)
		}
		time.Sleep(time.Millisecond * 100)
	})

	var chunks []streamChunk
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "RESPONSE [data]:") {
			continue
		}

		payload := strings.TrimSpace(strings.SplitN(line, "RESPONSE [data]:", 2)[1])
		chunk, err := parseStreamChunk([]byte(payload))
		if err != nil {
			t.Fatalf("parse stream chunk: %v\nline: %s", err, line)
		}
		chunks = append(chunks, chunk)
		processor.AddChunk(chunk)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan history file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("no SSE chunks parsed from history file")
	}

	messages, finishReason := processor.BuildMessages()
	t.Logf("finish_reason: %s", finishReason)
	if finishReason != "tool_calls" {
		t.Fatalf("unexpected finish reason: %q", finishReason)
	}
	if len(messages) == 0 {
		t.Fatal("expected normalized messages")
	}

	pretty, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		t.Fatalf("marshal final messages: %v", err)
	}
	fmt.Printf("final messages:\n%s", string(pretty))

}

func parseStreamChunk(payload []byte) (streamChunk, error) {
	var chunk streamChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return streamChunk{}, err
	}
	return chunk, nil
}
