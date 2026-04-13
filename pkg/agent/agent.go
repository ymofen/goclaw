package agent

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"goclaw/pkg/model"
)

// GoAgent implements the agentic loop with message memory, model execution, and tool integration.
type GoAgent struct {
	memory         *model.Memory
	model          model.AIModel
	formatter      model.AIModelRequestFormatter
	toolkit        model.Toolkit
	compressOption *CompressOption
	maxIterations  int
	prompts        []string
	compressFlag   atomic.Bool
}

// SetMemory configures the memory for the agent.
func (a *GoAgent) SetMemory(mem *model.Memory) {
	a.memory = mem
}

// SetModel configures the AIModel for the agent.
func (a *GoAgent) SetModel(m model.AIModel) {
	a.model = m
}

func (a *GoAgent) CompressFlag() bool {
	return a.compressFlag.Load()
}

// SetRequestFormatter configures the request formatter for the agent.
func (a *GoAgent) SetRequestFormatter(formatter model.AIModelRequestFormatter) {
	a.formatter = formatter
}

// SetToolkit configures the toolkit for the agent.
func (a *GoAgent) SetToolkit(toolkit model.Toolkit) {
	a.toolkit = toolkit
}

// SetCompressOption configures message compression strategy.
func (a *GoAgent) SetCompressOption(option *CompressOption) {
	a.compressOption = option
}

func (a *GoAgent) SetMaxIterations(n int) {
	a.maxIterations = n
}

// AppendPrompt appends one or more prompts to the agent's system prompt list.
func (a *GoAgent) AppendPrompt(prompts ...string) {
	a.prompts = append(a.prompts, prompts...)
}

func (a *GoAgent) SetPrompts(prompts ...string) {
	a.prompts = append(a.prompts[:0], prompts...)
}

// GetMemory returns a reference to the agent's memory.
func (a *GoAgent) GetMemory() *model.Memory {
	return a.memory
}

// Compress compresses the agent's memory using the configured CompressOption.
// The CompressOption.Prompt acts as the system message for compression.
// Recent messages (specified by PreserveRecentMessages) are excluded from compression.
// Only system messages after the preserved recent messages are kept.
// Directly updates a.memory with the compressed messages.
// Returns nil if no compression is configured or if compression succeeds, otherwise returns an error.
func (a *GoAgent) CompressIfNeed() error {
	if a.compressOption == nil {
		return nil
	}

	model := a.compressOption.Model
	if model == nil {
		model = a.model
	}
	if a.model == nil {
		return fmt.Errorf("no model configured for compression")
	}

	a.compressFlag.Store(true)
	defer a.compressFlag.Store(false)

	formatter := a.compressOption.RequestFormatter
	if formatter == nil {
		formatter = a.formatter
	}
	model.SetRequestFormatter(formatter)

	return CompressMemory(a.memory, a.compressOption, formatter, model)
}

// Execute runs one round of the agentic loop:
// 1. Compress memory if needed (using Compress() method)
// 2. Call model.Execute() to get response
// 3. Collect tool calls from response
// 4. If tool calls exist, execute them and feed results back into memory
// 5. Return final choices from this round
//
// The loop continues: if tool_call is present, we execute and loop again.
// If no tool_call returned, the conversation round is complete.
func (a *GoAgent) Execute() ([]model.Message, error) {
	// Validate prerequisites
	if a.model == nil {
		return nil, fmt.Errorf("model is not set")
	}
	if a.toolkit == nil {
		return nil, fmt.Errorf("toolkit is not set")
	}

	if a.maxIterations <= 0 {
		a.maxIterations = 10 // default max iterations to prevent infinite loops
	}

	// Compress memory if needed
	err := a.CompressIfNeed()
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	var sessionMessages []model.Message

	prompts := strings.Join(a.prompts, "\n")

	for i := 0; i < a.maxIterations; i++ {

		// Call model to get response
		a.formatter.SetPrompt(prompts)
		choices, err := a.model.Execute(a.memory)
		if err != nil {
			return nil, fmt.Errorf("model execution failed: %w", err)
		}
		messages, n, _ := a.processChoices(choices)
		a.memory.AddMessages(messages)
		sessionMessages = append(sessionMessages, messages...)
		if n == 0 {
			return sessionMessages, nil
		}
	}
	return sessionMessages, fmt.Errorf("max iterations reached")
}

// updateFormatter updates the formatter's memory and prompt.
// Currently supports OpenAIChatFormatter via type assertion.
func (a *GoAgent) updateFormatter(mem *model.Memory, prompt string) {
	// Silence other types; they may manage their own state
}

// processChoices handles the agentic loop: collect tool calls, execute them, and recurse if needed.
func (a *GoAgent) processChoices(choices []model.Message) (messages []model.Message, toolCallCount int, stopReason string) {
	// Collect tool calls and add non-stop messages to memory
	toolCalls := []model.Message{}
	for _, msg := range choices {
		if msg.Kind != model.KindStop {
			stopReason = msg.Content
			messages = append(messages, msg)
		}
		if msg.Kind == model.KindToolCall {
			toolCalls = append(toolCalls, msg)
		}
	}

	// No tool calls → return current choices
	if len(toolCalls) == 0 {
		return messages, 0, stopReason
	}

	toolCallId := fmt.Sprintf("tc-%d", time.Now().UnixNano())

	// Execute tool calls and feed results back
	for _, tc := range toolCalls {
		result, err := a.toolkit.Execute(tc.ToolName, tc.ToolArguments)
		if err != nil {
			result = fmt.Sprintf("error: %v", err)
		}

		// Add tool result to memory
		messages = append(messages, model.Message{
			ID:         toolCallId,
			Role:       "tool",
			Kind:       model.KindToolResult,
			ToolCallID: tc.ToolCallID,
			Content:    result,
		})
	}

	return messages, len(toolCalls), stopReason
}
