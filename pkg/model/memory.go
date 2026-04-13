package model

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Memory stores ordered chat messages.
type Memory struct {
	Compressed string    `json:"compressed"`
	List       []Message `json:"list"`
}

// Add appends one message to memory.
func (m *Memory) Add(msg Message) {
	if len(msg.ID) == 0 {
		msg.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	m.List = append(m.List, msg)
}

func (m *Memory) AddMessages(msgs []Message) {
	for _, msg := range msgs {
		m.Add(msg)
	}
}

// SnapshotMessages returns a copy of the current list of messages in memory.
func (m *Memory) SnapshotMessages() []Message {
	cloned := make([]Message, len(m.List))
	copy(cloned, m.List)
	return cloned
}

func (m *Memory) ResetMarks(mark string) {
	for i := range m.List {
		m.List[i].Marks = nil
	}
}

// UpdateMessages replaces the current list of messages in memory with the provided list.
func (m *Memory) UpdateMessages(msgs []Message) {
	m.List = make([]Message, len(msgs))
	copy(m.List, msgs)
}

func (m *Memory) Save(filePath string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (m *Memory) Load(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, m)
	if err != nil {
		return err
	}

	for idx := 0; idx < len(m.List); idx++ {
		if len(m.List[idx].ID) == 0 {
			m.List[idx].ID = fmt.Sprintf("%d-%d", time.Now().Unix(), idx)
		}
	}

	return nil
}

func (m *Memory) UpdateCompressed(msg string) {
	m.Compressed = msg
}

// GetMessagesExcludingMark returns messages that don't have the specified mark
func (m *Memory) GetMessagesExcludingMark(excludeMark string) []Message {
	if excludeMark == "" {
		return m.SnapshotMessages()
	}

	result := []Message{}
	if len(m.Compressed) > 0 {
		result = append(result,
			Message{Role: "system",
				Content: m.Compressed,
				Kind:    KindText,
			})
	}

	for _, msg := range m.List {
		if msg.ExcludingMark(excludeMark) {
			result = append(result, msg)
		}
	}
	return result
}

// UpdateMessagesMark updates marks for specific messages
func (m *Memory) UpdateMessagesMark(msgIDs []string, newMark string) {
	msgIDMap := make(map[string]bool)
	for _, id := range msgIDs {
		msgIDMap[id] = true
	}

	for i := range m.List {
		if msgIDMap[m.List[i].ID] {
			m.List[i].AddMark(newMark)
		}
	}
}

func SaveMessagesToFile(lst []Message, filePath string) error {
	data, err := json.MarshalIndent(lst, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func LoadMessagesFromFile(filePath string) ([]Message, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var lst []Message
	if err := json.Unmarshal(data, &lst); err != nil {
		return nil, err
	}
	return lst, nil
}

// 先对消息进行分组，id相同分到一组，tool_call 和 tool_result 视为一组
// 然后再按分组结果保留最近 keepRecent 组
func SplitMessagesForCompression(lst []Message, keepRecent int) (toCompress []Message, compressIDs []string, toKeep []Message) {
	if len(lst) == 0 {
		return nil, nil, nil
	}

	type messageGroup struct {
		messages []Message
	}

	groups := make([]messageGroup, 0, len(lst))
	currentGroup := messageGroup{messages: make([]Message, 0, 4)}
	groupIDs := make(map[string]struct{})
	groupToolCallIDs := make(map[string]struct{})

	flushGroup := func() {
		if len(currentGroup.messages) == 0 {
			return
		}
		groups = append(groups, currentGroup)
		currentGroup = messageGroup{messages: make([]Message, 0, 4)}
		groupIDs = make(map[string]struct{})
		groupToolCallIDs = make(map[string]struct{})
	}

	canJoinCurrentGroup := func(msg Message) bool {
		if len(currentGroup.messages) == 0 {
			return true
		}

		if msg.ID != "" {
			if _, ok := groupIDs[msg.ID]; ok {
				return true
			}
		}

		if msg.Kind == KindToolResult && msg.ToolCallID != "" {
			if _, ok := groupToolCallIDs[msg.ToolCallID]; ok {
				return true
			}
		}

		return false
	}

	for _, msg := range lst {
		if !canJoinCurrentGroup(msg) {
			flushGroup()
		}

		currentGroup.messages = append(currentGroup.messages, msg)
		if msg.ID != "" {
			groupIDs[msg.ID] = struct{}{}
		}
		if msg.Kind == KindToolCall && msg.ToolCallID != "" {
			groupToolCallIDs[msg.ToolCallID] = struct{}{}
		}
	}
	flushGroup()

	if keepRecent < 0 {
		keepRecent = 0
	}
	if keepRecent > len(groups) {
		keepRecent = len(groups)
	}

	splitIndex := len(groups) - keepRecent
	seenIDs := make(map[string]struct{})

	for idx, group := range groups {
		if idx < splitIndex {
			toCompress = append(toCompress, group.messages...)
			for _, msg := range group.messages {
				if msg.ID == "" {
					continue
				}
				if _, ok := seenIDs[msg.ID]; ok {
					continue
				}
				seenIDs[msg.ID] = struct{}{}
				compressIDs = append(compressIDs, msg.ID)
			}
			continue
		}

		toKeep = append(toKeep, group.messages...)
	}

	return toCompress, compressIDs, toKeep
}
