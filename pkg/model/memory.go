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

// SnapshotMessages returns a copy of the current list of messages in memory.
func (m *Memory) SnapshotMessages() []Message {
	cloned := make([]Message, len(m.List))
	copy(cloned, m.List)
	return cloned
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
			Message{Role: "user",
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

func SplitMessagesForCompression(lst []Message, keepRecent int) (toCompress []Message, compressIDs []string, toKeep []Message) {
	cutIndex := len(lst)

	if keepRecent > 0 {
		nKeep := 0
		accumulatedToolCallIDs := make(map[string]bool)
		// 从后往前遍历
		for i := len(lst) - 1; i >= 0; i-- {
			msg := lst[i]

			// Python 版本: tool_result 时 add
			if msg.Kind == "tool_result" && msg.ToolCallID != "" {
				accumulatedToolCallIDs[msg.ToolCallID] = true
			}

			// Python 版本: tool_use 时 remove
			if msg.Kind == "tool_call" && msg.ToolCallID != "" {
				if accumulatedToolCallIDs[msg.ToolCallID] {
					delete(accumulatedToolCallIDs, msg.ToolCallID)
				}
			}

			// 当没有未配对的 tool_call 时，计数加1
			if len(accumulatedToolCallIDs) == 0 {
				nKeep++
				if nKeep >= keepRecent {
					cutIndex = i
					break
				}
			}
		}

	}

	if cutIndex <= len(lst) {
		var chk map[string]struct{} = make(map[string]struct{})
		toCompress = lst[:cutIndex]
		for j := 0; j < len(toCompress); j++ {
			if _, ok := chk[toCompress[j].ID]; ok {
				continue
			}
			chk[toCompress[j].ID] = struct{}{}
			if len(toCompress[j].ID) > 0 {
				compressIDs = append(compressIDs, toCompress[j].ID)
			}
		}
		return toCompress, compressIDs, lst[cutIndex:]
	}
	return []Message{}, compressIDs, lst
}
