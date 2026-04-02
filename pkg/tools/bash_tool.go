package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"goclaw/pkg/model"
)

// RegisterBashTool registers the bash command execution tool with the toolkit.
func RegisterBashTool(mgr model.Toolkit) error {
	return mgr.Register(bashToolDef(), execBashCommand)
}

// RegisterShellCommandTool registers the shell_command execution tool with separate stdout/stderr capture.
func RegisterShellCommandTool(mgr model.Toolkit) error {
	return mgr.Register(shellCommandToolDef(), execShellCommand)
}

func bashToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "bash",
			Description: "Run a shell command and return its output. On Windows, uses PowerShell; on Unix-like systems, uses bash.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute.",
					},
				},
				"required": []string{"command"},
			}),
		},
	}
}

// execBashCommand executes a shell command and returns its output.
// On Windows, it uses PowerShell; on Unix-like systems, it uses bash.
func execBashCommand(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("bash: invalid arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("bash: command is required")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", args.Command)
	} else {
		cmd = exec.Command("bash", "-c", args.Command)
	}

	// Set working directory to current directory
	cmd.Dir, _ = os.Getwd()

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Even if command fails, return the output with the error message
		return fmt.Sprintf("command failed: %v\noutput:\n%s", err, string(output)), nil
	}

	return strings.TrimSpace(string(output)), nil
}

func shellCommandToolDef() ToolDef {
	return ToolDef{
		Type: "function",
		Function: FunctionDef{
			Name:        "shell_command",
			Description: "Execute a shell command with timeout support. Returns separate stdout, stderr, and return code.",
			Parameters: mustRaw(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute.",
					},
					"timeout": map[string]any{
						"type":        "integer",
						"description": "Maximum execution time in seconds (default: 300).",
						"default":     300,
					},
				},
				"required": []string{"command"},
			}),
		},
	}
}

// execShellCommand executes a shell command and returns returncode, stdout, and stderr.
// On Windows, uses PowerShell; on Unix-like systems, uses sh.
// Supports timeout with automatic process termination.
func execShellCommand(argsJSON json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("shell_command: invalid arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("shell_command: command is required")
	}

	if args.Timeout <= 0 {
		args.Timeout = 300
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-Command", args.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", args.Command)
	}

	cmd.Dir, _ = os.Getwd()

	// Capture stdout and stderr separately
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	returncode := 0
	err := cmd.Run()
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
		if stderrStr != "" {
			stderrStr += fmt.Sprintf("\nTimeoutError: The command execution exceeded the timeout of %d seconds.", args.Timeout)
		} else {
			stderrStr = fmt.Sprintf("TimeoutError: The command execution exceeded the timeout of %d seconds.", args.Timeout)
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
