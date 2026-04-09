package model

// AIModel defines a chat model that can execute one round and return normalized messages.
type AIModel interface {
	Execute() ([]Message, error)
}

// AIModelRequestFormatter serializes request payload for model providers.
type AIModelRequestFormatter interface {
	GetRequest() ([]byte, error)
	SetMemory(mem *Memory)
	SetPrompt(prompt string)
}

// AIModelEvent stores optional model lifecycle callbacks.
type AIModelEvent struct {
	OnSSEReplyHandler func(msg Message)
	OnRequestEvent    func(body []byte)
	OnResponse        func(eventType string, data []byte)
}
