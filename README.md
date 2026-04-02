

# goclaw

## 语言切换 / Language Switch
- [中文](README.md)
- [English](README.en.md)

**关键字**：Agentic、AI助手、Go、大模型、自动化、工具集成、Skill插件、OpenAI、百炼、命令行、SSE、自动办公、智能文档、Python工具

## 项目简介

goclaw 是一个基于 Go 语言开发的智能 Agentic 命令行助手，集成了 OpenAI 大模型能力，支持多种工具（如 Shell 命令、Python 脚本、文件操作、Skill 插件等）自动调用，适用于自动化办公、智能文档处理、AI 辅助开发等场景。

## 主要功能
- 支持多轮对话与上下文记忆，自动记录交互历史
- 集成 OpenAI Chat API，支持流式（SSE）推理输出
- 动态加载静态技能（Skill），可扩展多种能力
- 内置 Shell 命令、Python 代码、文件操作等工具
- 工具调用结果自动反馈给大模型，实现 Agentic 闭环
- 所有交互自动写入 history.log 便于追溯

## 依赖环境
- Go 1.20 及以上
- OpenAI API Key（需配置 .env 文件）
- （可选）Python 环境（如需用到 Python 工具）


## 快速运行推荐（百炼模型）
1. 申请免费的百炼账号。
2. 在百炼控制台开启免费的模型。
3. 修改 `.env.sample` 文件中的 `api_key`、`model_id`、`url`，填写你的百炼信息，并将文件重命名为 `.env`。
4. 已配置好 [vscode/launch.json]，可直接按 F5 快速运行。

> 百炼控制台地址：[https://bailian.console.aliyun.com/cn-beijing?spm=5176.28197581.0.0.6c9829a4cj1K7Y&tab=model#/model-usage](https://bailian.console.aliyun.com/cn-beijing?spm=5176.28197581.0.0.6c9829a4cj1K7Y&tab=model#/model-usage)

---

## 安装与运行（OpenAI模式）
1. 克隆仓库：
   ```bash
   git clone https://github.com/ymofen/goclaw.git
   cd goclaw
   ```
2. 配置 OpenAI API：
   - 在项目根目录下创建 `.env` 文件，内容示例：
     ```json
     { "APIKey": "sk-xxx", "BaseURL": "https://api.openai.com/v1", "SSE": true }
     ```
3. 编译并运行：
   ```bash
   cd cmd/main
   go run main.go
   ```

## 示例用法
- 直接在 main.go 里修改 memory.Add 里的 Content 字段，如：
  - "google当前截个图，保存为google.png"
  - "将hello, docx, 写入到hello.docx文件中"
- 程序会自动推理、调用相关工具，并输出结果。

## 技能扩展
- 可在 static/skills 目录下添加自定义技能（Skill），支持多种格式和脚本。

## 贡献与交流
- 欢迎提交 Issue 或 PR 参与项目改进。
- 联系方式：见仓库主页。

