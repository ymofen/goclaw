package tests

import (
	"encoding/json"
	"fmt"
	"goclaw/pkg/agent"
	"goclaw/pkg/openai"
	"testing"
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
