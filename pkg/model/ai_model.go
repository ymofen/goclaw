package model

// AIModel defines a chat model that can execute one round and return normalized messages.
type AIModel interface {
	Execute(mem *Memory) ([]Message, error)
	SetRequestFormatter(formatter AIModelRequestFormatter)
}

// AIModelRequestFormatter serializes request payload for model providers.
type AIModelRequestFormatter interface {
	GetRequest() ([]byte, error)
	SetMemory(mem *Memory)
	SetPrompt(prompt string)
}

// SSEMessage is the lightweight streaming payload used by SSE callbacks.
type SSEMessage struct {
	Kind    MessageKind
	Content string
	Step    uint8
}

const (
	SSEStepStart uint8 = 1 << iota
	SSEStepEnd
)

// AIModelEvent stores optional model lifecycle callbacks.
type AIModelEvent struct {
	OnSSEReplyHandler func(msg SSEMessage)
	OnRequestEvent    func(body []byte)
	OnResponse        func(eventType string, data []byte)
}
