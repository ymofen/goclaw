package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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
		Content: "google当前截个图，保存为google.png",
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

	var preKind model.MessageKind = ""
	api.SetOnSSEReply(func(msg model.Message) {
		switch msg.Kind {
		case model.KindReasoning, model.KindText:
			if msg.Content != "" {
				if preKind != msg.Kind {
					if preKind != "" {
						fmt.Println()
					}
					fmt.Printf("[%s]\n", msg.Kind)
					preKind = msg.Kind
				}
				fmt.Print(msg.Content)
			}
		case model.KindToolCall:
			fmt.Printf("\n[tool_call] %s(%s)\n", msg.ToolName, msg.ToolArguments)
			writeHistory(fmt.Sprintf("MESSAGE [tool_call] %s(%s)", msg.ToolName, msg.ToolArguments))
			preKind = msg.Kind
		case model.KindStop:
			// Reset SSE output state for next round.
			preKind = ""
		}

	})

	// Agentic loop: keep calling until no more tool calls are returned.
	for {
		choices, err := api.Execute()
		if err != nil {
			fmt.Printf("execute failed: %v\n", err)
			return
		}

		// Collect tool calls from this round's choices.
		var toolCalls []model.Message
		for _, msg := range choices {
			if msg.Kind != model.KindStop {
				memory.Add(msg)
			}
			if msg.Kind == model.KindToolCall {
				toolCalls = append(toolCalls, msg)
			}
		}

		if config.SSE {
			fmt.Println()
		} else {
			// Non-SSE: print text choices now.
			for _, msg := range choices {
				if msg.Kind == model.KindText && msg.Content != "" {
					fmt.Println(msg.Content)
				}
			}
		}

		// No tool calls → conversation is done.
		if len(toolCalls) == 0 {
			break
		}

		// Execute each tool call and feed results back into memory.
		for _, tc := range toolCalls {
			result, err := toolkit.Execute(tc.ToolName, tc.ToolArguments)
			if err != nil {
				result = fmt.Sprintf("error: %v", err)
			}
			fmt.Printf("[tool_result] %s → %s\n", tc.ToolName, truncate(result, 120))
			writeHistory(fmt.Sprintf("TOOL_RESULT [%s]: %s", tc.ToolName, truncate(result, 120)))
			memory.Add(model.Message{
				Role:       "tool",
				Kind:       model.KindToolResult,
				ToolCallID: tc.ToolCallID,
				Content:    result,
			})
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
