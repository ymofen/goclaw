# goclaw

**Keywords**: Agentic, AI Assistant, Go, LLM, Automation, Tool Integration, Skill Plugin, OpenAI, Bailian, CLI, SSE, Office Automation, Intelligent Document, Python Tools

## Project Overview

goclaw is an Agentic command-line assistant written in Go, integrating LLM (OpenAI or Bailian) capabilities. It supports automatic invocation of various tools (Shell commands, Python scripts, file operations, skill plugins, etc.), suitable for office automation, intelligent document processing, and AI-assisted development.

## Features
- Multi-turn conversation with context memory, auto-logging all interactions
- Integrated OpenAI Chat API, supports streaming (SSE) reasoning output
- Dynamically loads static skills (Skill), extensible for various capabilities
- Built-in Shell, Python, file operation tools
- Tool call results are automatically fed back to the LLM, forming a closed Agentic loop
- All interactions are recorded in history.log for traceability

## Requirements
- Go 1.20 or above
- OpenAI API Key (for OpenAI mode, configure .env)
- (Optional) Python environment (if using Python tools)

## Quick Start (Bailian Model)
1. Register a free Bailian account.
2. Enable a free model in the Bailian console.
3. Edit `.env.sample` with your `api_key`, `model_id`, and `url`, then rename it to `.env`.
4. VSCode [launch.json] is pre-configured, just press F5 to run.

> Bailian Console: [https://bailian.console.aliyun.com/cn-beijing?spm=5176.28197581.0.0.6c9829a4cj1K7Y&tab=model#/model-usage](https://bailian.console.aliyun.com/cn-beijing?spm=5176.28197581.0.0.6c9829a4cj1K7Y&tab=model#/model-usage)

---

## Installation & Run (OpenAI Mode)
1. Clone the repo:
   ```bash
   git clone https://github.com/ymofen/goclaw.git
   cd goclaw
   ```
2. Configure OpenAI API:
   - Create a `.env` file in the root directory, e.g.:
     ```json
     { "APIKey": "sk-xxx", "BaseURL": "https://api.openai.com/v1", "SSE": true }
     ```
3. Build and run:
   ```bash
   cd cmd/main
   go run main.go
   ```

## Example Usage
- Edit the `Content` field in `memory.Add` in main.go, e.g.:
  - "Take a screenshot of Google and save as google.png"
  - "Write 'hello, docx' into hello.docx file"
- The program will automatically reason, call relevant tools, and output results.

## Skill Extension
- Add custom skills (Skill) in the `static/skills` directory, supporting various formats and scripts.

## Contributing & Contact
- PRs and issues are welcome!
- See repo homepage for contact info.

---

> This project is an experimental Agent framework for AI toolchain development and office automation. Feedback and contributions are welcome!

---

## Language Switch / 语言切换
- [English](README.en.md)
- [中文](README.md)
