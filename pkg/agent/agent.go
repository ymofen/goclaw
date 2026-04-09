package agent

import (
	"fmt"

	"goclaw/pkg/model"
	"goclaw/pkg/openai"
)

// GoAgent implements the agentic loop with message memory, model execution, and tool integration.
type GoAgent struct {
	memory         *model.Memory
	model          model.AIModel
	formatter      model.AIModelRequestFormatter
	toolkit        model.Toolkit
	compressOption *CompressOption
	prompts        []string
}

// SetMemory configures the memory for the agent.
func (a *GoAgent) SetMemory(mem *model.Memory) {
	a.memory = mem
}

// SetModel configures the AIModel for the agent.
func (a *GoAgent) SetModel(m model.AIModel) {
	a.model = m
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

// AppendPrompt appends one or more prompts to the agent's system prompt list.
func (a *GoAgent) AppendPrompt(prompts ...string) {
	a.prompts = append(a.prompts, prompts...)
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
func (a *GoAgent) Compress() error {
	if a.compressOption == nil {
		return nil
	}

	if err := a.compressOption.Validate(); err != nil {
		return fmt.Errorf("invalid CompressOption: %w", err)
	}

	if a.model == nil {
		return fmt.Errorf("model is not set for compression")
	}

	// Get snapshot and separate by role
	msgs := a.memory.SnapshotMessages()
	systemMsgs := []model.Message{}
	nonSystemMsgs := []model.Message{}

	for _, msg := range msgs {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else {
			nonSystemMsgs = append(nonSystemMsgs, msg)
		}
	}

	// Determine split point for preservation
	preserveCount := a.compressOption.PreserveRecentMessages
	if preserveCount < 0 {
		preserveCount = 0
	}

	splitIdx := len(nonSystemMsgs) - preserveCount
	if splitIdx < 0 {
		splitIdx = 0
	}

	msgsToCompress := nonSystemMsgs[:splitIdx]
	msgsToPreserve := nonSystemMsgs[splitIdx:]

	// Find the first index in original list where preserved messages start
	// This helps us identify which system messages come after
	var preservedStartIdx int
	if len(msgsToPreserve) > 0 {
		// Find the index of the first preserved message in original list
		firstPreservedContent := msgsToPreserve[0].Content
		firstPreservedRole := msgsToPreserve[0].Role

		for i, msg := range msgs {
			if msg.Role == firstPreservedRole && msg.Content == firstPreservedContent {
				preservedStartIdx = i
				break
			}
		}
	} else {
		preservedStartIdx = len(msgs) // All messages will be compressed
	}

	// Keep only system messages that appear at or after the preserved messages start
	var recentSystemMsgs []model.Message
	for _, msg := range systemMsgs {
		for i, originalMsg := range msgs {
			if i >= preservedStartIdx &&
				originalMsg.Role == msg.Role &&
				originalMsg.Content == msg.Content {
				recentSystemMsgs = append(recentSystemMsgs, msg)
				break
			}
		}
	}

	// Build compression request memory
	compressionMem := model.Memory{
		List: []model.Message{
			{
				Role:    "system",
				Kind:    model.KindText,
				Content: a.compressOption.Prompt,
			},
		},
	}

	// Add messages to compress
	msgsCombined := ""
	for _, msg := range msgsToCompress {
		msgsCombined += fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content)
	}

	compressionMem.Add(model.Message{
		Role: "user",
		Kind: model.KindText,
		Content: fmt.Sprintf("Please summarize the following conversation to fit within %d tokens:\n\n%s",
			a.compressOption.MaxTokens, msgsCombined),
	})

	// Save original memory, temporarily switch to compression memory
	originalMem := a.memory
	a.memory = &compressionMem
	a.updateFormatter(a.memory, "")
	defer func() {
		// Restore original memory in case model stores it
		a.memory = originalMem
	}()

	// Get compression summary from model
	summaries, err := a.model.Execute()

	if err != nil {
		return fmt.Errorf("model compression failed: %w", err)
	}

	// Build final compressed memory: recent system messages + summary + preserved recent
	compressed := model.Memory{}

	// Add recent system messages only (after preserved message start)
	for _, msg := range recentSystemMsgs {
		compressed.Add(msg)
	}

	// Add compression summary
	for _, summary := range summaries {
		if summary.Kind == model.KindText && summary.Content != "" {
			compressed.Add(model.Message{
				Role:    "assistant",
				Kind:    model.KindText,
				Content: summary.Content,
			})
		}
	}

	// Add preserved recent messages
	for _, msg := range msgsToPreserve {
		compressed.Add(msg)
	}

	// Update agent's memory with compressed result
	a.memory = &compressed
	return nil
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

	// // Compress memory if CompressOption is set
	// if err := a.Compress(); err != nil {
	// 	return nil, fmt.Errorf("memory compression failed: %w", err)
	// }

	// // Update formatter with current memory and composite prompt
	// compositePrompt := strings.Join(a.prompts, "\n")
	// a.updateFormatter(a.memory, compositePrompt)

	// Call model to get response
	choices, err := a.model.Execute()
	if err != nil {
		return nil, fmt.Errorf("model execution failed: %w", err)
	}

	// Process choices and perform agentic loop
	return a.processChoices(choices, a.memory)
}

// updateFormatter updates the formatter's memory and prompt.
// Currently supports OpenAIChatFormatter via type assertion.
func (a *GoAgent) updateFormatter(mem *model.Memory, prompt string) {
	// Try to update as OpenAIChatFormatter
	if formatter, ok := a.formatter.(*openai.OpenAIRequestFormatter); ok {
		formatter.Memory = mem
		formatter.Prompt = prompt
		return
	}
	// Silence other types; they may manage their own state
}

// processChoices handles the agentic loop: collect tool calls, execute them, and recurse if needed.
func (a *GoAgent) processChoices(choices []model.Message, memToUse *model.Memory) ([]model.Message, error) {
	// Collect tool calls and add non-stop messages to memory
	var toolCalls []model.Message
	for _, msg := range choices {
		if msg.Kind != model.KindStop {
			memToUse.Add(msg)
		}
		if msg.Kind == model.KindToolCall {
			toolCalls = append(toolCalls, msg)
		}
	}

	// No tool calls → return current choices
	if len(toolCalls) == 0 {
		return choices, nil
	}

	// Execute tool calls and feed results back
	for _, tc := range toolCalls {
		result, err := a.toolkit.Execute(tc.ToolName, tc.ToolArguments)
		if err != nil {
			result = fmt.Sprintf("error: %v", err)
		}

		// Add tool result to memory
		memToUse.Add(model.Message{
			Role:       "tool",
			Kind:       model.KindToolResult,
			ToolCallID: tc.ToolCallID,
			Content:    result,
		})
	}

	// Recursively call model again with updated memory
	nextChoices, err := a.model.Execute()
	if err != nil {
		return nil, fmt.Errorf("model execution failed after tool calls: %w", err)
	}

	// Continue processing
	return a.processChoices(nextChoices, memToUse)
}
