package model

import "encoding/json"

// ToolHandler executes one tool with parsed JSON arguments.
type ToolHandler func(args json.RawMessage) (string, error)

// ToolDef is the OpenAI-compatible tool definition sent in requests.
type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes one callable tool function.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Toolkit defines a tool registration and dispatch abstraction.
type Toolkit interface {
	Register(def ToolDef, handler ToolHandler) error
	Unregister(name string) error
	Execute(name, argsJSON string) (string, error)
	Definitions() []ToolDef
}
