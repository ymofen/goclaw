package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"goclaw/pkg/agent"
	"goclaw/pkg/model"
	"goclaw/pkg/openai"
	"goclaw/pkg/skills"
	"goclaw/pkg/tools"
)

func main() {
	envText, err := os.ReadFile(".env")
	if err != nil {
		panic(fmt.Sprintf("failed to read .env: %v", err))
	}

	var config openai.OpenAIConfig
	if err := json.Unmarshal([]byte(envText), &config); err != nil {
		panic(err)
	}

	if !config.SSE {
		config.SSE = true
	}

	//config.SSE = true

	// Initialize history.log for recording all interactions
	logFile, err := os.Create("history.log")
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

	memory := model.Memory{}
	memory.Add(model.Message{
		Role: "user",
		Kind: model.KindText,
		// Content: "将hello, docx, 写入到hello.docx文件中",
		Content: "今天美伊战争怎么样, 搜索引擎用Bing",
	})

	skillmgr := skills.NewFileSkillMgr()
	err = skillmgr.LoadSkillsFromDir("static/skills")
	if err != nil {
		fmt.Printf("failed to load skills: %v\n", err)
	}

	prompts := []string{
		"You are a helpful assistant with access to file tools. Use them when the user asks about files.",
		skillmgr.GetPrompt(),
	}

	// return

	toolkit := tools.NewManager()

	if err := tools.RegisterShellCommandTool(toolkit); err != nil {
		fmt.Printf("failed to register shell command tool: %v\n", err)
		return
	}

	if err := tools.RegisterExecutePyCode(toolkit); err != nil {
		fmt.Printf("failed to register execute py code tool: %v\n", err)
		return
	}

	if err := tools.RegisterFileTools(toolkit); err != nil {
		fmt.Printf("failed to register file tools: %v\n", err)
		return
	}

	formatter := &openai.OpenAIRequestFormatter{
		Config:  config,
		Toolkit: toolkit,
		Prompt:  strings.Join(prompts, "\n"),
		Memory:  &memory,
	}

	api := openai.NewOpenAIChatModel(config, model.AIModelEvent{}, formatter)

	// Register event handlers for logging
	api.SetOnRequestEvent(func(body []byte) {
		writeHistory(fmt.Sprintf("REQUEST: %s", string(body)))
	})

	api.SetOnResponseEvent(func(eventType string, data []byte) {
		writeHistory(fmt.Sprintf("RESPONSE [%s]: %s", eventType, string(data)))
	})

	goAgent := &agent.GoAgent{}
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

	// Initialize GoAgent with configured dependencies

	goAgent.SetMemory(&memory)
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
		TriggerTokens: 1024,
		OnBeginCompress: func(tokenNum int) bool {
			fmt.Printf(">>> Compression triggered! Token count: %d\n", tokenNum)
			return true
		},
		OnEndCompress: func(content string) {
			fmt.Printf("Compress Done length:%d\n", len(content))
		},
	}
	goAgent.SetCompressOption(compressOpt)

	// Execute the agentic loop
	choices, err := goAgent.Execute()
	if err != nil {
		memory.Save("mem.log")
		fmt.Printf("agent execution failed: %v\n", err)
		return
	}

	memory.Save("mem.log")

	// Print final results
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

// truncate shortens s to at most n runes for display purposes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
