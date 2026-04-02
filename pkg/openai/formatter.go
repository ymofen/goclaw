package openai

import (
	"encoding/json"
	"fmt"

	"goclaw/pkg/model"
)

// OpenAIRequestFormatter formats messages as OpenAI-compatible chat payload.
type OpenAIRequestFormatter struct {
	Config  OpenAIConfig
	Toolkit model.Toolkit
	Prompt  string
	Memory  *model.Memory
}

// OpenAIChatFormatter is kept as a compatibility alias.
type OpenAIChatFormatter = OpenAIRequestFormatter

type chatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []any           `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []model.ToolDef `json:"tools,omitempty"`
}

// plainMessage is a standard role+content message.
type plainMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// assistantMessage is used for assistant history, including thinking and tool calls.
type assistantMessage struct {
	Role             string         `json:"role"`
	Content          *string        `json:"content"`
	ReasoningContent *string        `json:"reasoning_content,omitempty"`
	ToolCalls        []toolCallItem `json:"tool_calls,omitempty"`
}

type toolCallItem struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolResultMessage delivers the result of a tool call back to the model.
type toolResultMessage struct {
	Role       string `json:"role"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

type assistantMessageAccumulator struct {
	reasoning string
	content   string
	toolCalls []toolCallItem
}

func (a *assistantMessageAccumulator) empty() bool {
	return a.reasoning == "" && a.content == "" && len(a.toolCalls) == 0
}

func (a *assistantMessageAccumulator) toMessage() assistantMessage {
	msg := assistantMessage{Role: "assistant"}
	if a.content != "" {
		content := a.content
		msg.Content = &content
	}
	if a.reasoning != "" {
		reasoning := a.reasoning
		msg.ReasoningContent = &reasoning
	}
	if len(a.toolCalls) > 0 {
		msg.ToolCalls = append(msg.ToolCalls, a.toolCalls...)
	}
	return msg
}

// GetRequest serializes formatter state into request JSON.
func (f OpenAIRequestFormatter) GetRequest() ([]byte, error) {
	if f.Memory == nil {
		return nil, fmt.Errorf("memory is nil")
	}

	messages := make([]any, 0, len(f.Memory.List)+1)
	if f.Prompt != "" {
		messages = append(messages, plainMessage{
			Role:    "system",
			Content: f.Prompt,
		})
	}

	assistantAcc := &assistantMessageAccumulator{}
	flushAssistant := func() {
		if assistantAcc.empty() {
			return
		}
		messages = append(messages, assistantAcc.toMessage())
		assistantAcc = &assistantMessageAccumulator{}
	}

	for _, msg := range f.Memory.List {
		switch msg.Kind {
		case model.KindStop:
			// stop markers are internal only
			continue

		case model.KindReasoning:
			if msg.Role != "assistant" {
				flushAssistant()
				if msg.Content == "" {
					continue
				}
				messages = append(messages, plainMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
				continue
			}
			assistantAcc.reasoning += msg.Content

		case model.KindToolCall:
			if msg.ToolCallID == "" || msg.ToolName == "" {
				return nil, fmt.Errorf("KindToolCall message missing ToolCallID or ToolName")
			}
			assistantAcc.toolCalls = append(assistantAcc.toolCalls, toolCallItem{
				ID:   msg.ToolCallID,
				Type: "function",
				Function: toolCallFunction{
					Name:      msg.ToolName,
					Arguments: msg.ToolArguments,
				},
			})

		case model.KindToolResult:
			flushAssistant()
			if msg.ToolCallID == "" {
				return nil, fmt.Errorf("KindToolResult message missing ToolCallID")
			}
			messages = append(messages, toolResultMessage{
				Role:       "tool",
				ToolCallID: msg.ToolCallID,
				Content:    msg.Content,
			})

		default:
			if msg.Role == "assistant" {
				assistantAcc.content += msg.Content
				continue
			}
			flushAssistant()
			if msg.Content == "" {
				continue
			}
			messages = append(messages, plainMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	flushAssistant()

	req := chatCompletionRequest{
		Model:    f.Config.ModelID,
		Messages: messages,
		Stream:   f.Config.SSE,
	}
	if f.Toolkit != nil {
		req.Tools = f.Toolkit.Definitions()
	}
	return json.Marshal(req)
}
