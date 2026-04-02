package tools

import (
	"encoding/json"

	"goclaw/pkg/model"
)

// ToolDef is kept as alias for backward compatibility.
type ToolDef = model.ToolDef

// FunctionDef is kept as alias for backward compatibility.
type FunctionDef = model.FunctionDef

// FileTools keeps backward compatibility with existing callers.
// It mirrors the currently registered file tool definitions.
var FileTools []ToolDef

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
