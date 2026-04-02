package skills

import (
	"encoding/json"

	"goclaw/pkg/model"
)

// SkillDef is kept as alias for backward compatibility.
type SkillDef = model.ToolDef

// FunctionDef is kept as alias for backward compatibility.
type FunctionDef = model.FunctionDef

// FileSkills keeps backward compatibility with existing callers.
// It mirrors the currently registered file skill definitions.
var FileSkills []SkillDef

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
