package tests

import (
	"encoding/json"
	"fmt"
	"goclaw/pkg/agent"
	"goclaw/pkg/model"
	"goclaw/pkg/openai"
	"goclaw/pkg/tools"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAgentCompact(t *testing.T) {
	m := &model.Memory{}
	m.Load("../log/mem_raw_0.log")

	envText, err := os.ReadFile("../.env")
	if err != nil {
		panic(fmt.Sprintf("failed to read .env: %v", err))
	}

	var config openai.OpenAIConfig
	if err := json.Unmarshal([]byte(envText), &config); err != nil {
		panic(err)
	}

	logFile, err := os.Create("../log/compact_history.log")
	if err != nil {
		fmt.Printf("failed to create history.log: %v\n", err)
		return
	}
	defer logFile.Close()

	logMutex := &sync.Mutex{}
	writeHistory := func(entry string) {
		logMutex.Lock()
		defer logMutex.Unlock()
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		fmt.Fprintf(logFile, "[%s] %s\n", timestamp, entry)
		logFile.Sync()
	}

	prompts := []string{
		"You're a helpful assistant. ",
	}

	config.SSE = true

	toolkit := tools.NewManager()
	formatter := &openai.OpenAIRequestFormatter{
		Config:  config,
		Toolkit: toolkit,
		Prompt:  strings.Join(prompts, "\n"),
		Memory:  m,
	}

	api := openai.NewOpenAIChatModel(config, model.AIModelEvent{}, formatter)

	// Register event handlers for logging
	api.SetOnRequestEvent(func(body []byte) {
		writeHistory(fmt.Sprintf("REQUEST: %s", string(body)))
	})

	api.SetOnResponseEvent(func(eventType string, data []byte) {
		writeHistory(fmt.Sprintf("RESPONSE [%s]: %s", eventType, string(data)))
	})

	// Initialize GoAgent with configured dependencies
	goAgent := &agent.GoAgent{}
	goAgent.SetPrompts(prompts...)
	goAgent.SetMemory(m)
	goAgent.SetModel(api)
	goAgent.SetToolkit(toolkit)
	goAgent.SetRequestFormatter(formatter)
	compressOpt := &agent.CompressOption{
		Prompt:     agent.DefaultCompressionPrompt,
		KeepRecent: 1,
		TokenCounter: func(s string) int {
			// Simple token counter: 1 token per 4 characters (for testing purposes)
			return len(s) / 4
		},
		TriggerTokens: 512,
	}
	goAgent.SetCompressOption(compressOpt)

	api.SetOnSSEReply(func(msg model.SSEMessage) {
		if msg.Step&model.SSEStepStart != 0 {
			if goAgent.CompressFlag() {
				fmt.Printf("=== [Compress SSE] Event Start: %s ===\n", msg.Kind)
			} else {
				fmt.Printf("=== Event Start: %s ===\n", msg.Kind)
			}
		}
		if len(msg.Content) > 0 {
			fmt.Printf("%s", msg.Content)
		} else if msg.Kind == model.KindToolCall {
			fmt.Printf(".")
		}
		if msg.Step&model.SSEStepEnd != 0 {
			fmt.Printf("\n=== Event Done: %s ===\n", msg.Kind)
		}
	})

	m.Add(model.Message{
		Role:    "user",
		Content: "你好",
		Kind:    "text",
	})

	choices, err := goAgent.Execute()
	if err != nil {
		fmt.Printf("agent execution failed: %v\n", err)
		return
	}

	if config.SSE {
		fmt.Println()
	} else {
		// Non-SSE: print text choices
		for _, msg := range choices {
			if msg.Kind == model.KindText && msg.Content != "" {
				fmt.Println(msg.Content)
			}
		}
	}
}

func TestCompactFunction(t *testing.T) {
	m := &model.Memory{}
	m.Load("../log/mem.log")

	envText, err := os.ReadFile("../.env")
	if err != nil {
		panic(fmt.Sprintf("failed to read .env: %v", err))
	}

	var config openai.OpenAIConfig
	if err := json.Unmarshal([]byte(envText), &config); err != nil {
		panic(err)
	}

	logFile, err := os.Create("../log/compact_history.log")
	if err != nil {
		fmt.Printf("failed to create history.log: %v\n", err)
		return
	}
	defer logFile.Close()

	logMutex := &sync.Mutex{}
	writeHistory := func(entry string) {
		logMutex.Lock()
		defer logMutex.Unlock()
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		fmt.Fprintf(logFile, "[%s] %s\n", timestamp, entry)
		logFile.Sync()
	}

	prompts := []string{
		"You're a helpful assistant. Summarize the conversation so far in a way that will allow you to resume work efficiently in a future context window where the conversation history will be replaced with this summary.",
	}

	var compression_prompt = "<system-hint>You have been working on the task described above " +
		"but have not yet completed it. " +
		"Now write a continuation summary that will allow you to resume " +
		"work efficiently in a future context window where the " +
		"conversation history will be replaced with this summary. " +
		"The summary is returned in text format, including fields: task_overview, current_state, important_discoveries, next_steps, context_to_preserve." +
		"</system-hint>"

	var compressFunction = func(id string, keepRecent int) {
		lst := m.GetMessagesExcludingMark(model.MarkCompressed)
		toCompress, toCompressIds, toKeep := model.SplitMessagesForCompression(lst, keepRecent)
		model.SaveMessagesToFile(toCompress, fmt.Sprintf("../log/compact_%s_to_compress.log", id))
		model.SaveMessagesToFile(toKeep, fmt.Sprintf("../log/compact_%s_to_keep.log", id))

		if len(toCompress) == 0 {
			fmt.Printf("nothing need compress")
			return
		}

		mc := &model.Memory{}
		mc.UpdateMessages(toCompress)
		mc.Add(model.Message{
			Role:    "user",
			Content: compression_prompt,
			Kind:    "text",
		})

		config.SSE = true

		toolkit := tools.NewManager()
		formatter := &openai.OpenAIRequestFormatter{
			Config:  config,
			Toolkit: toolkit,
			Prompt:  strings.Join(prompts, "\n"),
			Memory:  mc,
		}

		api := openai.NewOpenAIChatModel(config, model.AIModelEvent{}, formatter)

		// Register event handlers for logging
		api.SetOnRequestEvent(func(body []byte) {
			writeHistory(fmt.Sprintf("%s REQUEST: %s", id, string(body)))
		})

		api.SetOnResponseEvent(func(eventType string, data []byte) {
			writeHistory(fmt.Sprintf("%s RESPONSE [%s]: %s", id, eventType, string(data)))
		})
		api.SetOnSSEReply(func(msg model.SSEMessage) {
			if msg.Step&model.SSEStepStart != 0 {
				fmt.Printf("=== Event Start: %s ===\n", msg.Kind)
			}
			if len(msg.Content) > 0 {
				fmt.Printf("%s", msg.Content)
			} else if msg.Kind == model.KindToolCall {
				fmt.Printf(".")
			}
			if msg.Step&model.SSEStepEnd != 0 {
				fmt.Printf("\n=== Event Done: %s ===\n", msg.Kind)
			}
		})

		// Initialize GoAgent with configured dependencies
		goAgent := &agent.GoAgent{}
		goAgent.SetMemory(mc)
		goAgent.SetModel(api)
		goAgent.SetToolkit(toolkit)
		goAgent.SetRequestFormatter(formatter)

		// Execute the agentic loop
		choices, err := goAgent.Execute()
		if err != nil {
			fmt.Printf("agent execution failed: %v\n", err)
			return
		}

		if len(choices) >= 1 {
			// 倒数第1个是结果
			content := choices[len(choices)-1].Content
			m.UpdateCompressed(content)
			m.UpdateMessagesMark(toCompressIds, model.MarkCompressed)

			m.Save(fmt.Sprintf("../log/compact_%s.log", id))
		} else {
			fmt.Printf("unexpected number of choices: %d\n", len(choices))
		}

		// Non-SSE: print text choices
		for _, msg := range choices {
			if msg.Kind == model.KindText && msg.Content != "" {
				fmt.Println(msg.Content)
			}
		}
	}

	compressFunction("comp1", 4)
	compressFunction("comp2", 0)

}
