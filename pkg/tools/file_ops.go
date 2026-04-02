package tools

import (
	"encoding/json"
	"fmt"
	"goclaw/pkg/model"
	"os"
	"strings"
)

func RegisterFileTools(mgr model.Toolkit) error {
	if err := mgr.Register(viewFileToolDef(), execViewFile); err != nil {
		return err
	}
	if err := mgr.Register(appendFileToolDef(), execAppendFile); err != nil {
		return err
	}
	if err := mgr.Register(editFileToolDef(), execEditFile); err != nil {
		return err
	}
	return nil
}

func viewFileToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "view_file",
			Description: "Read and return the full content of a file from disk.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the file to read.",
					},
				},
				"required": []string{"path"},
			}),
		},
	}
}

func appendFileToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "append_file",
			Description: "Append text content to the end of a file. Creates the file if it does not exist.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the file.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Text to append to the file.",
					},
				},
				"required": []string{"path", "content"},
			}),
		},
	}
}

func editFileToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "edit_file",
			Description: "Replace the first occurrence of old_str with new_str in a file. The file must already exist and old_str must appear exactly once.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the file.",
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

func execViewFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("view_file: invalid args: %w", err)
	}
	return ViewFile(args.Path)
}

func execAppendFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("append_file: invalid args: %w", err)
	}
	if err := AppendFile(args.Path, args.Content); err != nil {
		return "", err
	}
	return "ok", nil
}

func execEditFile(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Path   string `json:"path"`
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("edit_file: invalid args: %w", err)
	}
	if err := EditFile(args.Path, args.OldStr, args.NewStr); err != nil {
		return "", err
	}
	return "ok", nil
}

// ViewFile reads and returns the full content of a file.
func ViewFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("view_file: %w", err)
	}
	return string(data), nil
}

// AppendFile appends content to a file, creating it if it doesn't exist.
func AppendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("append_file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("append_file: %w", err)
	}
	return nil
}

// EditFile replaces the first occurrence of oldStr with newStr in a file.
func EditFile(path, oldStr, newStr string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("edit_file: %w", err)
	}
	original := string(data)
	count := strings.Count(original, oldStr)
	if count == 0 {
		return fmt.Errorf("edit_file: old_str not found in %s", path)
	}
	if count > 1 {
		return fmt.Errorf("edit_file: old_str appears %d times in %s; must appear exactly once", count, path)
	}
	updated := strings.Replace(original, oldStr, newStr, 1)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("edit_file: %w", err)
	}
	return nil
}
