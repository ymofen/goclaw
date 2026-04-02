package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"goclaw/pkg/model"
)

// RegisterExecutePyCode registers the Python code execution tool with the toolkit.
func RegisterExecutePyCode(mgr model.Toolkit) error {
	return mgr.Register(executePyCodeToolDef(), execPythonCode)
}

func executePyCodeToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "execute_python_code",
			Description: "Execute Python code in a temporary file and capture the return code, standard output, and error. The code output must be printed to get results.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"code": map[string]any{
						"type":        "string",
						"description": "The Python code to be executed.",
					},
					"timeout": map[string]any{
						"type":        "number",
						"description": "The maximum time (in seconds) allowed for the code to run (default: 300).",
						"default":     300,
					},
				},
				"required": []string{"code"},
			}),
		},
	}
}

// execPythonCode executes Python code in a temporary file and returns returncode, stdout, and stderr.
// Supports timeout with automatic process termination.
func execPythonCode(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Code    string  `json:"code"`
		Timeout float64 `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("execute_python_code: invalid arguments: %w", err)
	}

	if args.Code == "" {
		return "", fmt.Errorf("execute_python_code: code is required")
	}

	if args.Timeout <= 0 {
		args.Timeout = 300
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "goclaw_py_*")
	if err != nil {
		return "", fmt.Errorf("execute_python_code: failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary Python file
	tempFile := filepath.Join(tempDir, "tmp_code.py")
	if err := os.WriteFile(tempFile, []byte(args.Code), 0o600); err != nil {
		return "", fmt.Errorf("execute_python_code: failed to write temp file: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
	defer cancel()

	// Execute Python code
	cmd := exec.CommandContext(ctx, "python", "-u", tempFile)
	cmd.Dir, _ = os.Getwd()

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"PYTHONUTF8=1",
		"PYTHONIOENCODING=utf-8",
	)

	// Capture stdout and stderr separately
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	returncode := 0
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			returncode = exitErr.ExitCode()
		}
	}

	stdoutStr := strings.TrimSpace(stdoutBuf.String())
	stderrStr := strings.TrimSpace(stderrBuf.String())

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		returncode = -1
		timeoutMsg := fmt.Sprintf("TimeoutError: The code execution exceeded the timeout of %.0f seconds.", args.Timeout)
		if stderrStr != "" {
			stderrStr += "\n" + timeoutMsg
		} else {
			stderrStr = timeoutMsg
		}
	}

	result := fmt.Sprintf(
		"<returncode>%d</returncode><stdout>%s</stdout><stderr>%s</stderr>",
		returncode,
		stdoutStr,
		stderrStr,
	)

	return result, nil
}
