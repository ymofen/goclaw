package agent

import (
	"encoding/json"
	"testing"

	"goclaw/pkg/model"
	"goclaw/pkg/openai"
)

// MockAIModel is a mock implementation of model.AIModel for testing.
type MockAIModel struct {
	ExecuteFunc func() ([]model.Message, error)
	CallCount   int
}

func (m *MockAIModel) Execute() ([]model.Message, error) {
	m.CallCount++
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc()
	}
	return []model.Message{}, nil
}

// MockToolkit is a mock implementation of model.Toolkit for testing.
type MockToolkit struct {
	ExecuteFunc func(name, argsJSON string) (string, error)
	CallCount   int
}

func (m *MockToolkit) Register(def model.ToolDef, handler model.ToolHandler) error {
	return nil
}

func (m *MockToolkit) Unregister(name string) error {
	return nil
}

func (m *MockToolkit) Execute(name, argsJSON string) (string, error) {
	m.CallCount++
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(name, argsJSON)
	}
	return "mock result", nil
}

func (m *MockToolkit) Definitions() []model.ToolDef {
	return []model.ToolDef{}
}

// MockFormatter is a mock implementation of model.AIModelRequestFormatter.
type MockFormatter struct {
}

func (m *MockFormatter) GetRequest() ([]byte, error) {
	return []byte("{}"), nil
}

// Test_GoAgent_SetMemory verifies memory can be set and retrieved.
func Test_GoAgent_SetMemory(t *testing.T) {
	agent := &GoAgent{}
	mem := model.Memory{
		List: []model.Message{
			{Role: "user", Kind: model.KindText, Content: "hello"},
		},
	}

	agent.SetMemory(mem)

	if len(agent.GetMemory().List) != 1 {
		t.Errorf("expected 1 message, got %d", len(agent.GetMemory().List))
	}
	if agent.GetMemory().List[0].Content != "hello" {
		t.Errorf("expected content 'hello', got %s", agent.GetMemory().List[0].Content)
	}
}

// Test_GoAgent_SetModel verifies model can be set.
func Test_GoAgent_SetModel(t *testing.T) {
	agent := &GoAgent{}
	mockModel := &MockAIModel{}

	agent.SetModel(mockModel)

	if agent.model != mockModel {
		t.Error("model not set correctly")
	}
}

// Test_GoAgent_SetToolkit verifies toolkit can be set.
func Test_GoAgent_SetToolkit(t *testing.T) {
	agent := &GoAgent{}
	mockToolkit := &MockToolkit{}

	agent.SetToolkit(mockToolkit)

	if agent.toolkit != mockToolkit {
		t.Error("toolkit not set correctly")
	}
}

// Test_GoAgent_SetCompressOption verifies CompressOption can be set.
func Test_GoAgent_SetCompressOption(t *testing.T) {
	agent := &GoAgent{}
	option := &CompressOption{
		Prompt:       "compress",
		TokenCounter: func(s string) int { return len(s) / 4 },
		MaxTokens:    1000,
	}

	agent.SetCompressOption(option)

	if agent.compressOption != option {
		t.Error("CompressOption not set correctly")
	}
}

// Test_GoAgent_AppendPrompt verifies prompts can be appended.
func Test_GoAgent_AppendPrompt(t *testing.T) {
	agent := &GoAgent{}

	agent.AppendPrompt("prompt1", "prompt2")
	if len(agent.prompts) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(agent.prompts))
	}

	agent.AppendPrompt("prompt3")
	if len(agent.prompts) != 3 {
		t.Errorf("expected 3 prompts, got %d", len(agent.prompts))
	}

	if agent.prompts[0] != "prompt1" || agent.prompts[1] != "prompt2" || agent.prompts[2] != "prompt3" {
		t.Errorf("prompts not appended correctly")
	}
}

// Test_GoAgent_Execute_NoModel verifies Execute fails when model is not set.
func Test_GoAgent_Execute_NoModel(t *testing.T) {
	agent := &GoAgent{
		toolkit: &MockToolkit{},
	}

	_, err := agent.Execute()
	if err == nil {
		t.Error("expected error when model is not set")
	}
}

// Test_GoAgent_Execute_NoToolkit verifies Execute fails when toolkit is not set.
func Test_GoAgent_Execute_NoToolkit(t *testing.T) {
	agent := &GoAgent{
		model: &MockAIModel{},
	}

	_, err := agent.Execute()
	if err == nil {
		t.Error("expected error when toolkit is not set")
	}
}

// Test_GoAgent_Execute_SimpleCompletion tests a simple model response with no tool calls.
func Test_GoAgent_Execute_SimpleCompletion(t *testing.T) {
	agent := &GoAgent{
		model:     &MockAIModel{},
		toolkit:   &MockToolkit{},
		formatter: &MockFormatter{},
	}

	agent.model.(*MockAIModel).ExecuteFunc = func() ([]model.Message, error) {
		return []model.Message{
			{Role: "assistant", Kind: model.KindText, Content: "response"},
			{Role: "assistant", Kind: model.KindStop, Content: ""},
		}, nil
	}

	choices, err := agent.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(choices) != 2 {
		t.Errorf("expected 2 messages, got %d", len(choices))
	}

	// Only text message should be added to memory (stop messages are not added)
	if len(agent.GetMemory().List) != 1 {
		t.Errorf("expected 1 message in memory, got %d", len(agent.GetMemory().List))
	}
}

// Test_GoAgent_Execute_WithToolCall tests agentic loop with tool execution.
func Test_GoAgent_Execute_WithToolCall(t *testing.T) {
	agent := &GoAgent{
		toolkit:   &MockToolkit{},
		formatter: &MockFormatter{},
	}

	callCount := 0
	agent.model = &MockAIModel{
		ExecuteFunc: func() ([]model.Message, error) {
			callCount++
			if callCount == 1 {
				// First call: return tool call
				toolArgs := map[string]string{"file": "test.txt"}
				argsJSON, _ := json.Marshal(toolArgs)
				return []model.Message{
					{
						Role:          "assistant",
						Kind:          model.KindToolCall,
						ToolName:      "write_file",
						ToolCallID:    "call_001",
						ToolArguments: string(argsJSON),
					},
				}, nil
			}
			// Second call: return final response
			return []model.Message{
				{Role: "assistant", Kind: model.KindText, Content: "done"},
				{Role: "assistant", Kind: model.KindStop, Content: ""},
			}, nil
		},
	}

	agent.toolkit.(*MockToolkit).ExecuteFunc = func(name, argsJSON string) (string, error) {
		return "file written", nil
	}

	choices, err := agent.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called model twice
	if callCount != 2 {
		t.Errorf("expected model.Execute() called twice, got %d times", callCount)
	}

	// Should have executed tool once
	if agent.toolkit.(*MockToolkit).CallCount != 1 {
		t.Errorf("expected toolkit.Execute() called once, got %d times", agent.toolkit.(*MockToolkit).CallCount)
	}

	// Final response should be in choices
	if len(choices) < 1 || choices[0].Content != "done" {
		t.Errorf("expected final response in choices")
	}

	// Memory should contain: tool call, tool result, and final response
	if len(agent.GetMemory().List) < 3 {
		t.Errorf("expected at least 3 messages in memory, got %d", len(agent.GetMemory().List))
	}
}

// Test_CompressOption_Validate tests CompressOption validation.
func Test_CompressOption_Validate(t *testing.T) {
	// Test missing TokenCounter
	option := &CompressOption{
		Prompt:    "compress",
		MaxTokens: 100,
	}
	err := option.Validate()
	if err == nil {
		t.Error("expected error for missing TokenCounter")
	}

	// Test invalid MaxTokens
	option = &CompressOption{
		Prompt:       "compress",
		TokenCounter: func(s string) int { return len(s) },
		MaxTokens:    0,
	}
	err = option.Validate()
	if err == nil {
		t.Error("expected error for invalid MaxTokens")
	}

	// Test empty Prompt
	option = &CompressOption{
		TokenCounter: func(s string) int { return len(s) },
		MaxTokens:    100,
		Prompt:       "",
	}
	err = option.Validate()
	if err == nil {
		t.Error("expected error for empty Prompt")
	}

	// Test negative PreserveRecentMessages
	option = &CompressOption{
		Prompt:                 "compress",
		TokenCounter:           func(s string) int { return len(s) },
		MaxTokens:              100,
		PreserveRecentMessages: -1,
	}
	err = option.Validate()
	if err == nil {
		t.Error("expected error for negative PreserveRecentMessages")
	}

	// Test valid option
	option = &CompressOption{
		Prompt:                 "compress",
		TokenCounter:           func(s string) int { return len(s) },
		MaxTokens:              100,
		PreserveRecentMessages: 2,
	}
	err = option.Validate()
	if err != nil {
		t.Errorf("unexpected error for valid option: %v", err)
	}
}

// Test_GoAgent_Compress_NoOption tests Compress when no CompressOption is set.
func Test_GoAgent_Compress_NoOption(t *testing.T) {
	agent := &GoAgent{}
	agent.memory.Add(model.Message{
		Role:    "user",
		Kind:    model.KindText,
		Content: "hello",
	})

	originalLen := len(agent.memory.List)
	err := agent.Compress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not change memory when no option is set
	if len(agent.memory.List) != originalLen {
		t.Errorf("expected same memory size when no option set, got %d", len(agent.memory.List))
	}
}

// Test_GoAgent_Compress_WithOption tests Compress with valid CompressOption and model.
func Test_GoAgent_Compress_WithOption(t *testing.T) {
	agent := &GoAgent{}

	// Add some messages to memory
	agent.memory.Add(model.Message{
		Role:    "system",
		Kind:    model.KindText,
		Content: "you are helpful",
	})
	agent.memory.Add(model.Message{
		Role:    "user",
		Kind:    model.KindText,
		Content: "hello",
	})
	agent.memory.Add(model.Message{
		Role:    "assistant",
		Kind:    model.KindText,
		Content: "hi there",
	})

	// Set up compression option
	agent.SetCompressOption(&CompressOption{
		Prompt:       "Summarize the conversation",
		TokenCounter: func(s string) int { return len(s) },
		MaxTokens:    100,
	})

	// Use real OpenAI formatter for compression
	mockToolkit := &MockToolkit{}
	formatter := &openai.OpenAIRequestFormatter{
		Config: openai.OpenAIConfig{
			ModelID: "test",
			URL:     "http://test",
			APIKey:  "test",
			SSE:     false,
		},
		Toolkit: mockToolkit,
		Prompt:  "",
		Memory:  agent.memory,
	}
	agent.SetRequestFormatter(formatter)

	// Mock model to return compression summary
	agent.SetModel(&MockAIModel{
		ExecuteFunc: func() ([]model.Message, error) {
			return []model.Message{
				{Role: "assistant", Kind: model.KindText, Content: "conversation summary"},
			}, nil
		},
	})

	err := agent.Compress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compressed memory (in agent.memory) should have system message + summary
	if len(agent.memory.List) < 2 {
		t.Errorf("expected at least 2 messages in compressed memory, got %d", len(agent.memory.List))
	}

	// Should have system message
	systemFound := false
	for _, msg := range agent.memory.List {
		if msg.Role == "system" {
			systemFound = true
			break
		}
	}
	if !systemFound {
		t.Error("system message should be preserved")
	}
}

// Test_GoAgent_Compress_InvalidOption tests Compress with invalid CompressOption.
func Test_GoAgent_Compress_InvalidOption(t *testing.T) {
	agent := &GoAgent{}

	// Set invalid compression option (missing TokenCounter)
	agent.SetCompressOption(&CompressOption{
		Prompt:    "compress",
		MaxTokens: 100,
	})

	err := agent.Compress()
	if err == nil {
		t.Error("expected error for invalid compression option")
	}
}

// Test_GoAgent_Compress_PreserveRecentMessages tests that recent messages are preserved during compression.
func Test_GoAgent_Compress_PreserveRecentMessages(t *testing.T) {
	agent := &GoAgent{}

	// Add multiple messages to memory
	agent.memory.Add(model.Message{
		Role:    "system",
		Kind:    model.KindText,
		Content: "you are helpful",
	})
	agent.memory.Add(model.Message{
		Role:    "user",
		Kind:    model.KindText,
		Content: "first question",
	})
	agent.memory.Add(model.Message{
		Role:    "assistant",
		Kind:    model.KindText,
		Content: "first answer",
	})
	agent.memory.Add(model.Message{
		Role:    "user",
		Kind:    model.KindText,
		Content: "second question",
	})
	agent.memory.Add(model.Message{
		Role:    "assistant",
		Kind:    model.KindText,
		Content: "second answer",
	})

	// Set compression option to preserve last 2 messages
	agent.SetCompressOption(&CompressOption{
		Prompt:                 "Summarize the conversation",
		TokenCounter:           func(s string) int { return len(s) },
		MaxTokens:              100,
		PreserveRecentMessages: 2,
	})

	// Use real OpenAI formatter
	formatter := &openai.OpenAIRequestFormatter{
		Config: openai.OpenAIConfig{
			ModelID: "test",
			URL:     "http://test",
			APIKey:  "test",
			SSE:     false,
		},
		Toolkit: &MockToolkit{},
		Prompt:  "",
		Memory:  agent.memory,
	}
	agent.SetRequestFormatter(formatter)

	// Mock model to return compression summary
	agent.SetModel(&MockAIModel{
		ExecuteFunc: func() ([]model.Message, error) {
			return []model.Message{
				{Role: "assistant", Kind: model.KindText, Content: "[SUMMARY] User asked questions, assistant answered"},
			}, nil
		},
	})

	err := agent.Compress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected structure in agent.memory after compression:
	// 1. system message
	// 2. summary from compression
	// 3. last 2 messages (recent user and assistant)
	if len(agent.memory.List) < 4 {
		t.Errorf("expected at least 4 messages in compressed memory, got %d", len(agent.memory.List))
	}

	// Verify system message is present
	if agent.memory.List[0].Role != "system" {
		t.Error("first message should be system message")
	}

	// Verify last 2 messages are preserved (user "second question" and assistant "second answer")
	lastTwoIdx := len(agent.memory.List) - 2
	if lastTwoIdx >= 0 {
		if agent.memory.List[lastTwoIdx].Content != "second question" {
			t.Errorf("expected second-to-last message to be 'second question', got %q", agent.memory.List[lastTwoIdx].Content)
		}
		if agent.memory.List[len(agent.memory.List)-1].Content != "second answer" {
			t.Errorf("expected last message to be 'second answer', got %q", agent.memory.List[len(agent.memory.List)-1].Content)
		}
	}
}

// Test_GoAgent_Compress_PreserveMoreThanAvailable tests PreserveRecentMessages when it exceeds available messages.
func Test_GoAgent_Compress_PreserveMoreThanAvailable(t *testing.T) {
	agent := &GoAgent{}

	// Add only 2 non-system messages
	agent.memory.Add(model.Message{
		Role:    "system",
		Kind:    model.KindText,
		Content: "system",
	})
	agent.memory.Add(model.Message{
		Role:    "user",
		Kind:    model.KindText,
		Content: "msg1",
	})
	agent.memory.Add(model.Message{
		Role:    "assistant",
		Kind:    model.KindText,
		Content: "msg2",
	})

	// Try to preserve 5 messages (more than available)
	agent.SetCompressOption(&CompressOption{
		Prompt:                 "Summarize",
		TokenCounter:           func(s string) int { return len(s) },
		MaxTokens:              100,
		PreserveRecentMessages: 5,
	})

	// Use real OpenAI formatter
	formatter := &openai.OpenAIRequestFormatter{
		Config: openai.OpenAIConfig{
			ModelID: "test",
			URL:     "http://test",
			APIKey:  "test",
			SSE:     false,
		},
		Toolkit: &MockToolkit{},
		Prompt:  "",
		Memory:  agent.memory,
	}
	agent.SetRequestFormatter(formatter)

	agent.SetModel(&MockAIModel{
		ExecuteFunc: func() ([]model.Message, error) {
			return []model.Message{
				{Role: "assistant", Kind: model.KindText, Content: "summary"},
			}, nil
		},
	})

	err := agent.Compress()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have system + summary + all 2 original messages preserved in agent.memory
	// (nothing to compress since preserve count exceeds available)
	if len(agent.memory.List) < 3 {
		t.Errorf("expected at least 3 messages, got %d", len(agent.memory.List))
	}
}

// Test_GoAgent_PromptComposition tests that prompts are properly appended in order.
// (Actual prompt composition is tested in integration tests with real formatters)
func Test_GoAgent_PromptComposition(t *testing.T) {
	agent := &GoAgent{}
	agent.AppendPrompt("prompt1")
	agent.AppendPrompt("prompt2", "prompt3")

	if len(agent.prompts) != 3 {
		t.Errorf("expected 3 prompts, got %d", len(agent.prompts))
	}

	// Verify order
	expectedOrder := []string{"prompt1", "prompt2", "prompt3"}
	for i, expected := range expectedOrder {
		if agent.prompts[i] != expected {
			t.Errorf("expected prompt[%d] = %q, got %q", i, expected, agent.prompts[i])
		}
	}
}

// Helper function to run all tests
func TestAll(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"SetMemory", Test_GoAgent_SetMemory},
		{"SetModel", Test_GoAgent_SetModel},
		{"SetToolkit", Test_GoAgent_SetToolkit},
		{"SetCompressOption", Test_GoAgent_SetCompressOption},
		{"AppendPrompt", Test_GoAgent_AppendPrompt},
		{"Execute_NoModel", Test_GoAgent_Execute_NoModel},
		{"Execute_NoToolkit", Test_GoAgent_Execute_NoToolkit},
		{"Execute_SimpleCompletion", Test_GoAgent_Execute_SimpleCompletion},
		{"Execute_WithToolCall", Test_GoAgent_Execute_WithToolCall},
		{"CompressOption_Validate", Test_CompressOption_Validate},
		{"Compress_NoOption", Test_GoAgent_Compress_NoOption},
		{"Compress_WithOption", Test_GoAgent_Compress_WithOption},
		{"Compress_InvalidOption", Test_GoAgent_Compress_InvalidOption},
		{"Compress_PreserveRecentMessages", Test_GoAgent_Compress_PreserveRecentMessages},
		{"Compress_PreserveMoreThanAvailable", Test_GoAgent_Compress_PreserveMoreThanAvailable},
		{"PromptComposition", Test_GoAgent_PromptComposition},
	}

	for _, test := range tests {
		t.Run(test.name, test.fn)
	}
}
