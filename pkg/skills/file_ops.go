package skills

import (
	"encoding/json"
	"fmt"
	"goclaw/pkg/model"
	"os"
	"strings"
)

func RegisterFileSkills(mgr model.Toolkit) error {
	if err := mgr.Register(viewSkillFileDef(), execViewSkillFile); err != nil {
		return err
	}
	if err := mgr.Register(appendSkillFileDef(), execAppendSkillFile); err != nil {
		return err
	}
	if err := mgr.Register(editSkillFileDef(), execEditSkillFile); err != nil {
		return err
	}
	return nil
}

func viewSkillFileDef() SkillDef {
	return SkillDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "view_skill_file",
			Description: "Read and return the full content of a skill file from disk.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the skill file to read.",
					},
				},
				"required": []string{"path"},
			}),
		},
	}
}

func appendSkillFileDef() SkillDef {
	return SkillDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "append_skill_file",
			Description: "Append text content to the end of a skill file. Creates the file if it does not exist.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the skill file.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Text to append to the skill file.",
					},
				},
				"required": []string{"path", "content"},
			}),
		},
	}
}

func editSkillFileDef() SkillDef {
	return SkillDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "edit_skill_file",
			Description: "Replace the first occurrence of old_str with new_str in a skill file. The file must already exist and old_str must appear exactly once.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the skill file.",
					},
					"old_str": map[string]any{
						"type":        "string",
						"description": "Exact string to find and replace (must appear exactly once).",
					},
					"new_str": map[string]any{
						"type":        "string",
						"description": "Replacement string.",
					},
				},
				"required": []string{"path", "old_str", "new_str"},
			}),
		},
	}
}

func execViewSkillFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("view_skill_file: invalid args: %w", err)
	}
	return ViewSkillFile(args.Path)
}

func execAppendSkillFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("append_skill_file: invalid args: %w", err)
	}
	if err := AppendSkillFile(args.Path, args.Content); err != nil {
		return "", err
	}
	return "ok", nil
}

func execEditSkillFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path   string `json:"path"`
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("edit_skill_file: invalid args: %w", err)
	}
	if err := EditSkillFile(args.Path, args.OldStr, args.NewStr); err != nil {
		return "", err
	}
	return "ok", nil
}

// ViewSkillFile reads and returns the full content of a skill file.
func ViewSkillFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("view_skill_file: %w", err)
	}
	return string(data), nil
}

// AppendSkillFile appends content to a skill file, creating it if it doesn't exist.
func AppendSkillFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("append_skill_file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("append_skill_file: %w", err)
	}
	return nil
}

// EditSkillFile replaces the first occurrence of oldStr with newStr in a skill file.
func EditSkillFile(path, oldStr, newStr string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("edit_skill_file: %w", err)
	}
	original := string(data)
	count := strings.Count(original, oldStr)
	if count == 0 {
		return fmt.Errorf("edit_skill_file: old_str not found in %s", path)
	}
	if count > 1 {
		return fmt.Errorf("edit_skill_file: old_str appears %d times in %s; must appear exactly once", count, path)
	}
	updated := strings.Replace(original, oldStr, newStr, 1)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("edit_skill_file: %w", err)
	}
	return nil
}
