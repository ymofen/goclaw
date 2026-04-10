package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"goclaw/pkg/model"
)

// OpenAIChatModel executes chat requests against an OpenAI-compatible API.
type OpenAIChatModel struct {
	Config    OpenAIConfig
	Client    *http.Client
	Event     model.AIModelEvent
	Formatter model.AIModelRequestFormatter
}

// OpenAIModel is kept as a compatibility alias.
type OpenAIModel = OpenAIChatModel

var _ model.AIModel = (*OpenAIChatModel)(nil)

// NewOpenAIChatModel creates an OpenAI chat model instance.
func NewOpenAIChatModel(cfg OpenAIConfig, event model.AIModelEvent, formatter model.AIModelRequestFormatter) *OpenAIChatModel {
	return &OpenAIChatModel{
		Config:    cfg,
		Event:     event,
		Formatter: formatter,
	}
}

// SetOnSSEReply registers callback for each SSE incremental message.
func (m *OpenAIChatModel) SetOnSSEReply(handler func(msg model.Message)) {
	m.Event.OnSSEReplyHandler = handler
}

// OnSSEReply dispatches one message callback when handler is registered.
func (m *OpenAIChatModel) OnSSEReply(msg model.Message) {
	if m.Event.OnSSEReplyHandler != nil {
		m.Event.OnSSEReplyHandler(msg)
	}
}

// SetOnRequestEvent registers callback for request body events.
func (m *OpenAIChatModel) SetOnRequestEvent(handler func(body []byte)) {
	m.Event.OnRequestEvent = handler
}

// OnRequestEvent dispatches request body when handler is registered.
func (m *OpenAIChatModel) OnRequestEvent(body []byte) {
	if m.Event.OnRequestEvent != nil {
		m.Event.OnRequestEvent(body)
	}
}

// SetOnResponseEvent registers callback for response events.
func (m *OpenAIChatModel) SetOnResponseEvent(handler func(eventType string, data []byte)) {
	m.Event.OnResponse = handler
}

// OnResponseEvent dispatches response event when handler is registered.
func (m *OpenAIChatModel) OnResponseEvent(eventType string, data []byte) {
	if m.Event.OnResponse != nil {
		m.Event.OnResponse(eventType, data)
	}
}

type chatCompletionResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []streamToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

type streamToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type streamToolCall struct {
	Index    int                    `json:"index"`
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function streamToolCallFunction `json:"function"`
}

type streamChunk struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          string           `json:"content"`
			ReasoningContent string           `json:"reasoning_content"`
			ToolCalls        []streamToolCall `json:"tool_calls"`
		} `json:"delta"`
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []streamToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// apiErrorResponse represents API error in SSE stream
type apiErrorResponse struct {
	Error struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
	} `json:"error"`
}

// Execute sends request and returns normalized messages in received order.
func (m *OpenAIChatModel) Execute() ([]model.Message, error) {
	if m.Formatter == nil {
		return nil, fmt.Errorf("formatter is nil")
	}

	body, err := m.Formatter.GetRequest()
	if err != nil {
		return nil, err
	}

	// Fire request event before sending
	m.OnRequestEvent(body)

	endpoint := strings.TrimRight(m.Config.URL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.Config.APIKey)
	if m.Config.SSE {
		req.Header.Set("Accept", "text/event-stream")
	}

	client := m.Client
	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, readErr
		}
		m.OnResponseEvent("response_body", raw)
		return nil, fmt.Errorf("request failed: %s\n%s", resp.Status, string(raw))
	}

	if m.Config.SSE {
		return m.executeSSE(resp.Body)
	}

	// For non-SSE, read full response body and fire event
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	m.OnResponseEvent("response_body", raw)

	// Parse and return
	return m.executeNonSSEWithBody(raw)
}

// toolCallAcc accumulates streaming fragments for one tool call slot.
type toolCallAcc struct {
	id   string
	name string
	args strings.Builder
}

func appendOrMergeSSEMessage(messages []model.Message, msg model.Message) []model.Message {
	if msg.Kind == model.KindToolCall {
		return append(messages, msg)
	}

	if len(messages) == 0 {
		return append(messages, msg)
	}

	last := &messages[len(messages)-1]
	if last.Role == msg.Role && last.Kind == msg.Kind && last.ID == msg.ID && last.Created == msg.Created && last.Model == msg.Model {
		last.Content += msg.Content
		return messages
	}

	return append(messages, msg)
}

func (m *OpenAIChatModel) executeSSE(body io.Reader) ([]model.Message, error) {
	messages := make([]model.Message, 0, 16)
	lastChunkID := ""
	var lastChunkCreated int64
	lastChunkModel := ""
	// toolAccs accumulates streamed tool_calls by their index.
	toolAccs := map[int]*toolCallAcc{}
	toolMessageIndex := map[int]int{}

	err := ProcessReplyStream(body, func(dataType PayloadType, payload []byte) error {
		// Fire response event for each SSE payload
		m.OnResponseEvent(string(dataType), payload)

		if dataType != PayloadTypeData {
			return nil
		}

		// 先检查是否为 API 错误响应（可能同时有 id 和 error 字段）
		var apiErr apiErrorResponse
		if err := json.Unmarshal(payload, &apiErr); err == nil && apiErr.Error.Code != "" {
			// 这是一个 API 错误，返回格式化的错误
			return fmt.Errorf("API error [%s]: %s (type: %s)", apiErr.Error.Code, apiErr.Error.Message, apiErr.Error.Type)
		}

		// 否则按正常的 SSE 数据块解析
		var chunk streamChunk
		if err := json.Unmarshal(payload, &chunk); err != nil {
			// 既不是错误响应也无法解析为数据块，忽略并继续
			return nil
		}
		if chunk.ID != "" {
			lastChunkID = chunk.ID
		}
		if chunk.Created != 0 {
			lastChunkCreated = chunk.Created
		}
		if chunk.Model != "" {
			lastChunkModel = chunk.Model
		}

		for _, choice := range chunk.Choices {
			role := choice.Message.Role
			if role == "" {
				role = "assistant"
			}

			// Merge adjacent reasoning fragments in arrival order.
			if choice.Delta.ReasoningContent != "" {
				msg := model.Message{Role: role, Kind: model.KindReasoning, Content: choice.Delta.ReasoningContent, ID: chunk.ID, Created: chunk.Created, Model: chunk.Model}
				messages = appendOrMergeSSEMessage(messages, msg)
				m.OnSSEReply(msg)
			}

			// Merge adjacent text fragments in arrival order.
			if choice.Delta.Content != "" {
				msg := model.Message{Role: role, Kind: model.KindText, Content: choice.Delta.Content, ID: chunk.ID, Created: chunk.Created, Model: chunk.Model}
				messages = appendOrMergeSSEMessage(messages, msg)
				m.OnSSEReply(msg)
			}

			// Some providers place content on message instead of delta.
			if choice.Message.Content != "" {
				msg := model.Message{Role: role, Kind: model.KindText, Content: choice.Message.Content, ID: chunk.ID, Created: chunk.Created, Model: chunk.Model}
				messages = appendOrMergeSSEMessage(messages, msg)
				m.OnSSEReply(msg)
			}

			mergeToolCalls := func(toolCalls []streamToolCall) {
				for _, tc := range toolCalls {
					acc, ok := toolAccs[tc.Index]
					if !ok {
						acc = &toolCallAcc{}
						toolAccs[tc.Index] = acc
					}
					if tc.ID != "" {
						acc.id = tc.ID
					}
					if tc.Function.Name != "" {
						acc.name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						acc.args.WriteString(tc.Function.Arguments)
					}

					msgIndex, exists := toolMessageIndex[tc.Index]
					if !exists {
						messages = append(messages, model.Message{
							Role:          role,
							Kind:          model.KindToolCall,
							ID:            chunk.ID,
							Created:       chunk.Created,
							Model:         chunk.Model,
							ToolCallID:    acc.id,
							ToolName:      acc.name,
							ToolArguments: acc.args.String(),
						})
						toolMessageIndex[tc.Index] = len(messages) - 1
						continue
					}

					messages[msgIndex].Role = role
					messages[msgIndex].ID = chunk.ID
					messages[msgIndex].Created = chunk.Created
					messages[msgIndex].Model = chunk.Model
					messages[msgIndex].ToolCallID = acc.id
					messages[msgIndex].ToolName = acc.name
					messages[msgIndex].ToolArguments = acc.args.String()
				}
			}

			// Merge tool call fragments back into their first-seen position.
			mergeToolCalls(choice.Delta.ToolCalls)
			mergeToolCalls(choice.Message.ToolCalls)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	stop := model.Message{Role: "assistant", Kind: model.KindStop, Content: "", ID: lastChunkID, Created: lastChunkCreated, Model: lastChunkModel}
	messages = append(messages, stop)
	m.OnSSEReply(stop)

	return messages, nil
}

func (m *OpenAIChatModel) executeNonSSEWithBody(raw []byte) ([]model.Message, error) {
	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode failed: %v\nraw response: %s", err, string(raw))
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response: %s", string(raw))
	}

	messages := make([]model.Message, 0, len(parsed.Choices)+1)
	for _, c := range parsed.Choices {
		role := c.Message.Role
		if role == "" {
			role = "assistant"
		}
		if c.Message.Content != "" {
			messages = append(messages, model.Message{Role: role, Kind: model.KindText, Content: c.Message.Content, ID: parsed.ID, Created: parsed.Created, Model: parsed.Model})
		}
		for _, tc := range c.Message.ToolCalls {
			messages = append(messages, model.Message{
				Role:          role,
				Kind:          model.KindToolCall,
				ID:            parsed.ID,
				Created:       parsed.Created,
				Model:         parsed.Model,
				ToolCallID:    tc.ID,
				ToolName:      tc.Function.Name,
				ToolArguments: tc.Function.Arguments,
			})
		}
	}
	messages = append(messages, model.Message{Role: "assistant", Kind: model.KindStop, Content: "", ID: parsed.ID, Created: parsed.Created, Model: parsed.Model})
	return messages, nil
}

func (m *OpenAIChatModel) executeNonSSE(body io.Reader) ([]model.Message, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return m.executeNonSSEWithBody(raw)
}
