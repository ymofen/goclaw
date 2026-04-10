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

const MarkCompressed = "COMPRESSED"

// Message is a normalized message used by the app and model layer.
type Message struct {
	Role          string      `json:"role"`
	Kind          MessageKind `json:"kind"`
	Content       string      `json:"content"`
	ID            string      `json:"id,omitempty"` // 可能会重复
	Created       int64       `json:"created,omitempty"`
	Model         string      `json:"model,omitempty"`
	ToolCallID    string      `json:"tool_call_id,omitempty"`
	ToolName      string      `json:"tool_name,omitempty"`
	ToolArguments string      `json:"tool_arguments,omitempty"`
	Marks         []string    `json:"marks,omitempty"`
}

// ExcludingMark returns true if the given message does not have the specified mark
func (s *Message) ExcludingMark(excludeMark string) bool {
	if excludeMark == "" || &s == nil {
		return true
	}
	for _, mark := range s.Marks {
		if mark == excludeMark {
			return false
		}
	}
	return true
}

// AddMark adds the given mark to the message if it does not already exist.
// It returns true if the mark was added, or false if the mark already existed.
func (s *Message) AddMark(addMark string) bool {
	for _, mark := range s.Marks {
		if mark == addMark {
			return false
		}
	}
	s.Marks = append(s.Marks, addMark)
	return true
}
