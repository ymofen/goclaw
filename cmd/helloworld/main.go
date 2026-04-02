package main

import (
	"encoding/json"
	"fmt"

	"goclaw/pkg/model"
	"goclaw/pkg/openai"
)

const configText = `{
	"id": "kimi-k2-thinking-bailian",
	"name": "kimi-k2-thinking-bailian",
	"model_id": "kimi-k2-thinking",
	"api_key": "sk-8234d689595545eb83bea3985ae5e1ad",
	"url": "https://dashscope.aliyuncs.com/compatible-mode/v1"
}`

func main() {
	var config openai.OpenAIConfig
	if err := json.Unmarshal([]byte(configText), &config); err != nil {
		panic(err)
	}
	if !config.SSE {
		config.SSE = true
	}

	memory := model.Memory{}
	memory.Add(model.Message{Role: "user", Kind: model.KindText, Content: "介绍一下你自己"})

	formatter := &openai.OpenAIRequestFormatter{
		Config: config,
		Prompt: "You are a helpful assistant.",
		Memory: &memory,
	}
	api := openai.NewOpenAIChatModel(config, model.AIModelEvent{}, formatter)
	api.SetOnSSEReply(func(msg model.Message) {
		switch msg.Kind {
		case model.KindReasoning, model.KindText:
			if msg.Content != "" {
				fmt.Print(msg.Content)
			}
		}
	})

	choices, err := api.Execute()
	if err != nil {
		fmt.Printf("execute failed: %v\n", err)
		return
	}

	if config.SSE {
		fmt.Println()
		return
	}

	for _, msg := range choices {
		if msg.Kind == model.KindText && msg.Content != "" {
			fmt.Println(msg.Content)
		}
	}
}
