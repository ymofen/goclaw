package model

// MessageKind describes the semantic type of one message unit.
type MessageKind string

const (
	KindText       MessageKind = "text"
	KindReasoning  MessageKind = "reasoning"
	KindStop       MessageKind = "stop"
	KindToolCall   MessageKind = "tool_call"
	KindToolResult MessageKind = "tool_result"
)

// Message is a normalized message used by the app and model layer.
type Message struct {
	Role          string      `json:"role"`
	Kind          MessageKind `json:"kind"`
	Content       string      `json:"content"`
	ID            string      `json:"id,omitempty"`
	Created       int64       `json:"created,omitempty"`
	Model         string      `json:"model,omitempty"`
	ToolCallID    string      `json:"tool_call_id,omitempty"`
	ToolName      string      `json:"tool_name,omitempty"`
	ToolArguments string      `json:"tool_arguments,omitempty"`
}
