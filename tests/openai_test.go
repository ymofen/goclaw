package tests

import (
	"encoding/json"
	"fmt"
	"goclaw/pkg/agent"
	"goclaw/pkg/model"
	"goclaw/pkg/openai"
	"goclaw/pkg/skills"
	"goclaw/pkg/tools"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGenerateOpenAISchema(t *testing.T) {

	schemaMap, err := openai.GenerateOpenAISchema(agent.CompressSummarySchema{}, "summary_schema", "Conversation summary structure")
	if err != nil {
		panic(err)
	}
	jsonBytes, err := json.MarshalIndent(schemaMap, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(jsonBytes))
}

func TestOpenAIThinking(t *testing.T) {
	envText, err := os.ReadFile("../.env")
	if err != nil {
		panic(fmt.Sprintf("failed to read .env: %v", err))
	}

	var config openai.OpenAIConfig
	if err := json.Unmarshal([]byte(envText), &config); err != nil {
		panic(err)
	}

	// Initialize history.log for recording all interactions
	nowIDString := time.Now().Format("20060102_150405")
	historyName := fmt.Sprintf("log/%s_history_openai.log", nowIDString)
	logFile, err := os.Create(historyName)
	if err != nil {
		fmt.Printf("failed to create %s: %v\n", historyName, err)
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
		Role:    "user",
		Kind:    model.KindText,
		Content: "今天美伊战争怎么样, 搜索引擎用Bing",
	})

	skillmgr := skills.NewFileSkillMgr()
	err = skillmgr.LoadSkillsFromDir("../static/skills")
	if err != nil {
		fmt.Printf("failed to load skills: %v\n", err)
	}

	prompts := []string{
		"You are a helpful assistant with access to file tools. Use them when the user asks about files.",
		skillmgr.GetPrompt(),
	}

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

	config.SSE = true

	formatter := &openai.OpenAIRequestFormatter{
		Config:  config,
		Toolkit: toolkit,
		Prompt:  strings.Join(prompts, "\n"),
		Memory:  &memory,
	}

	api := openai.NewOpenAIChatModel(config, model.AIModelEvent{}, formatter)
	var preKind model.MessageKind = ""
	api.SetOnRequestEvent(func(body []byte) {
		writeHistory(fmt.Sprintf("REQUEST: %s", string(body)))
	})

	api.SetOnResponseEvent(func(eventType string, data []byte) {
		writeHistory(fmt.Sprintf("RESPONSE [%s]: %s", eventType, string(data)))
	})

	api.SetOnSSEReply(func(msg model.SSEMessage) {
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
		case model.KindStop:
			// Reset SSE output state for next round.
			preKind = ""
		}
	})

	defer func() {
		memFile := fmt.Sprintf("log/%s_mem_openai.log", nowIDString)
		memory.Save(memFile)
	}()

	choices, err := api.Execute()
	if err != nil {
		fmt.Printf("execute failed: %v\n", err)
		return
	}

	if config.SSE {
		fmt.Println()
	} else {
		for _, msg := range choices {
			if msg.Content != "" {
				fmt.Println(msg.Content)
			}
		}
	}
	for _, msg := range choices {
		if msg.Kind != model.KindStop {
			memory.Add(msg)
		}
	}

}
