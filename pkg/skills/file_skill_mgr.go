package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"goclaw/pkg/model"
)

// FileSkillMgr implements model.SkillMgr and manages a collection of FileSkill instances.
type FileSkillMgr struct {
	mu     sync.RWMutex
	skills map[string]*FileSkill
}

// NewFileSkillMgr creates an empty FileSkillMgr.
func NewFileSkillMgr() *FileSkillMgr {
	return &FileSkillMgr{
		skills: make(map[string]*FileSkill),
	}
}

// RegisterSkill adds or replaces a skill. If a skill with the same name already
// exists it is overwritten without error.
func (m *FileSkillMgr) RegisterSkill(skill *FileSkill) error {
	if skill == nil {
		return fmt.Errorf("skills: skill is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills[skill.GetName()] = skill
	return nil
}

// UnregisterSkill removes the skill identified by its name.
func (m *FileSkillMgr) UnregisterSkill(skill *FileSkill) {
	if skill == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.skills, skill.GetName())
}

// GetPrompt returns a prompt that tells the model about all registered skills:
// which skills are available, their absolute directory paths, and their full
// descriptions so the model knows how to load files within each skill.
func (m *FileSkillMgr) GetPrompt() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Skills\n\n")
	sb.WriteString("The following skills are loaded. Each skill's files are located at the absolute path shown below.\n")
	sb.WriteString("To access any file inside a skill, use the absolute directory path as the base.\n")

	for _, s := range m.skills {
		sb.WriteString("\n---\n")
		sb.WriteString(fmt.Sprintf("## Skill: %s\n", s.GetName()))
		sb.WriteString(fmt.Sprintf("Directory: %s\n", s.GetDir()))
		sb.WriteString(s.GetRawMeta())
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

// RegisterToolFunction registers the GetSkillContent tool into toolkit.
// The function accepts a JSON object with a single `name` string parameter
// and returns the full SKILL.md content for the matching skill.
func (m *FileSkillMgr) RegisterToolFunction(toolkit model.Toolkit) {
	// nothing need to register

	// def := model.ToolDef{
	// 	Type: "function",
	// 	Function: model.FunctionDef{
	// 		Name:        "GetSkillContent",
	// 		Description: "获取指定技能的详细内容（SKILL.md 文件内容）",
	// 		Parameters: mustRaw(map[string]any{
	// 			"type": "object",
	// 			"properties": map[string]any{
	// 				"name": map[string]any{
	// 					"type":        "string",
	// 					"description": "技能名称",
	// 				},
	// 			},
	// 			"required": []string{"name"},
	// 		}),
	// 	},
	// }

	// handler := func(args json.RawMessage) (string, error) {
	// 	var params struct {
	// 		Name string `json:"name"`
	// 	}
	// 	if err := json.Unmarshal(args, &params); err != nil {
	// 		return "", fmt.Errorf("GetSkillContent: invalid args: %w", err)
	// 	}
	// 	return m.GetSkillContent(params.Name)
	// }

	// // Errors (e.g. already registered) are intentionally ignored; the caller is
	// // responsible for not calling RegisterToolFunction multiple times.
	// _ = toolkit.Register(def, handler)
}

// GetSkillContent returns the full SKILL.md content for the named skill.
func (m *FileSkillMgr) GetSkillContent(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.skills[name]
	if !ok {
		return "", fmt.Errorf("skills: skill %q not found", name)
	}
	return s.GetPrompt(), nil
}

// LoadSkillsFromDir scans the immediate subdirectories of dir, calls
// NewFileSkill for each, and registers them. Directories that do not contain a
// SKILL.md file are silently skipped.
func (m *FileSkillMgr) LoadSkillsFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("skills: cannot read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(dir, entry.Name())
		skill, err := NewFileSkill(subDir)
		if err != nil {
			// No SKILL.md or unreadable — skip
			continue
		}
		if err := m.RegisterSkill(skill); err != nil {
			return err
		}
	}
	return nil
}

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
