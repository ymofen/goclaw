package agent

import (
	"fmt"
	"goclaw/pkg/model"
)

type CompressSummarySchema struct {
	TaskOverview         string `json:"task_overview" description:"The user's core request and success criteria"`
	CurrentState         string `json:"current_state" description:"What has been completed so far"`
	ImportantDiscoveries string `json:"important_discoveries" description:"Technical constraints or requirements uncovered"`
	NextSteps            string `json:"next_steps" description:"Specific actions needed to complete the task"`
	ContextToPreserve    string `json:"context_to_preserve" description:"User preferences or style requirements"`
}

var DefaultCompressionPrompt = "<system-hint>You have been working on the task described above " +
	"but have not yet completed it. " +
	"Now write a continuation summary that will allow you to resume " +
	"work efficiently in a future context window where the " +
	"conversation history will be replaced with this summary. " +
	"The summary is returned in text format, including fields: task_overview, current_state, important_discoveries, next_steps, context_to_preserve." +
	"</system-hint>"

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

	// KeepRecent is the number of recent messages to keep without compression.
	// These messages will be excluded from the compression request and added back as-is.
	KeepRecent int

	// Model is the AIModel used for compression.
	Model model.AIModel

	// RequestFormatter formats the memory into a request for the compression model.
	RequestFormatter model.AIModelRequestFormatter

	// OnBeginCompress is an optional callback that is called when compression starts.
	OnBeginCompress func(tokenNum int) bool

	// OnEndCompress is an optional callback that is called when compression ends.
	OnEndCompress func(content string)
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
	if c.KeepRecent < 0 {
		return fmt.Errorf("KeepRecent must be >= 0")
	}
	return nil
}

func CompressMemory(mem *model.Memory, compresOption *CompressOption, formatter model.AIModelRequestFormatter, aiModel model.AIModel) error {
	lst := mem.GetMessagesExcludingMark(model.MarkCompressed)
	toCompress, toCompressIds, _ := model.SplitMessagesForCompression(lst, compresOption.KeepRecent)

	if len(toCompress) == 0 {
		return nil
	}

	mc := &model.Memory{}
	mc.UpdateMessages(toCompress)
	mc.Add(model.Message{
		Role:    "user",
		Content: compresOption.Prompt,
		Kind:    "text",
	})

	formatter.SetMemory(mc)
	formatter.SetPrompt(compresOption.Prompt)

	tokenNum := 0
	if compresOption.TokenCounter != nil {
		body, err := formatter.GetRequest()
		if err != nil {
			return fmt.Errorf("failed to get compression request body: %v", err)
		}
		tokenNum = compresOption.TokenCounter(string(body))
		if tokenNum < compresOption.TriggerTokens {
			// No need to compress yet
			return nil
		}
	}

	if compresOption.OnBeginCompress != nil {
		if !compresOption.OnBeginCompress(tokenNum) {
			// Compression cancelled by callback
			return nil
		}
	}

	// 原始APIModel执行完成
	choices, err := aiModel.Execute(mc)
	if err != nil {
		return fmt.Errorf("compression failed: %v", err)
	}

	if len(choices) >= 2 {
		// 压缩完成内容应该在第二条
		content := choices[1].Content
		mem.UpdateCompressed(content)
		mem.UpdateMessagesMark(toCompressIds, model.MarkCompressed)
		if compresOption.OnEndCompress != nil {
			compresOption.OnEndCompress(content)
		}
		return nil
	}

	return fmt.Errorf("compression failed: no choices returned")
}
