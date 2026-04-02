package model

// Memory stores ordered chat messages.
type Memory struct {
	List []Message
}

// Add appends one message to memory.
func (m *Memory) Add(msg Message) {
	m.List = append(m.List, msg)
}
