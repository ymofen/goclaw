package agent

import (
	"fmt"
)

type CompressSummarySchema struct {
	TaskOverview         string `json:"task_overview" description:"The user's core request and success criteria"`
	CurrentState         string `json:"current_state" description:"What has been completed so far"`
	ImportantDiscoveries string `json:"important_discoveries" description:"Technical constraints or requirements uncovered"`
	NextSteps            string `json:"next_steps" description:"Specific actions needed to complete the task"`
	ContextToPreserve    string `json:"context_to_preserve" description:"User preferences or style requirements"`
}

// CompressOption configures message compression strategy for memory management.
type CompressOption struct {
	// Prompt is the compression hint sent to the model for summarization.
	Prompt string

	// TokenCounter counts tokens in a string.
	// Used to determine if memory exceeds MaxTokens.
	TokenCounter func(s string) int

	// MaxTokens is the target maximum token count for compressed memory.
	MaxTokens int

	// TriggerTokens is the token count threshold that triggers compression.
	TriggerTokens int

	// PreserveRecentMessages is the number of recent messages to keep without compression.
	// These messages will be excluded from the compression request and added back as-is.
	PreserveRecentMessages int
}

// Validate checks if CompressOption is properly configured.
func (c *CompressOption) Validate() error {
	if c.TokenCounter == nil {
		return fmt.Errorf("TokenCounter is nil")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("MaxTokens must be > 0")
	}
	if c.Prompt == "" {
		return fmt.Errorf("Prompt cannot be empty")
	}
	if c.PreserveRecentMessages < 0 {
		return fmt.Errorf("PreserveRecentMessages must be >= 0")
	}
	return nil
}
