# OpenAI-Go v3.31.0 完全指南

> 官方文档来源：[github.com/openai/openai-go](https://github.com/openai/openai-go) v3.31.0 版本
> 发布日期：2026-04-08

## 目录
1. [Client 创建](#client-创建)
2. [Chat Completions 请求](#chat-completions-请求)
3. [Response Format 参数](#response-format-参数)
4. [完整的可运行示例](#完整的可运行示例)
5. [常见错误和解决方案](#常见错误和解决方案)

---

## Client 创建

### 基础方式

```go
package main

import (
	"context"
	"os"
	
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	// 方式 1：使用环境变量 OPENAI_API_KEY（推荐）
	client := openai.NewClient()
	
	// 方式 2：显式指定 API Key
	client := openai.NewClient(
		option.WithAPIKey("sk-..."),
	)
	
	// 方式 3：指定自定义 Base URL（兼容的 OpenAI 服务）
	client := openai.NewClient(
		option.WithAPIKey("sk-..."),
		option.WithBaseURL("https://your-custom-api.com/v1"),
	)
}
```

### Client 可用的选项

| 选项 | 说明 | 示例 |
|------|------|------|
| `WithAPIKey(key)` | 设置 API 密钥 | `option.WithAPIKey("sk-...")` |
| `WithBaseURL(url)` | 设置基础 URL | `option.WithBaseURL("https://api.openai.com/v1")` |
| `WithHTTPClient(client)` | 自定义 HTTP 客户端 | `option.WithHTTPClient(customClient)` |
| `WithHeader(key, value)` | 添加自定义 Header | `option.WithHeader("X-Custom", "value")` |
| `WithMaxRetries(n)` | 设置重试次数（默认2） | `option.WithMaxRetries(5)` |
| `WithRequestTimeout(duration)` | 设置单次请求超时 | `option.WithRequestTimeout(20*time.Second)` |

---

## Chat Completions 请求

### 基础语法

```go
ctx := context.Background()

response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	// 必需字段
	Model:    openai.ChatModelGPT4o,
	Messages: []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("你好，请介绍一下你自己"),
	},
})

if err != nil {
	panic(err)
}

// 获取响应内容
content := response.Choices[0].Message.Content
println(content)
```

### ChatCompletionNewParams 主要字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `Model` | string | ✓ | 模型名称（如 `gpt-4o`） |
| `Messages` | `[]ChatCompletionMessageParamUnion` | ✓ | 消息列表 |
| `MaxTokens` | `param.Opt[int64]` | | 最大输出 token 数 |
| `Temperature` | `param.Opt[float64]` | | 温度（0-2） |
| `TopP` | `param.Opt[float64]` | | Top P 采样 |
| `Seed` | `param.Opt[int64]` | | 随机种子 |
| `ResponseFormat` | `ChatCompletionNewParamsResponseFormatUnion` | | **响应格式** |
| `Tools` | `[]ChatCompletionToolUnionParam` | | 工具/函数 |
| `Stop` | `ChatCompletionNewParamsStopUnion` | | 停止序列 |

### 消息类型

```go
// 用户消息
openai.UserMessage("你的问题")

// 助手消息
openai.AssistantMessage("回复内容")

// 系统消息
openai.SystemMessage("系统提示词")

// 开发者消息
openai.DeveloperMessage("开发者消息")
```

---

## Response Format 参数

### 架构概览

`ResponseFormat` 是一个 **Union 类型**，只能设置一个字段：

```go
type ChatCompletionNewParamsResponseFormatUnion struct {
	OfText       *shared.ResponseFormatTextParam       // 纯文本格式
	OfJSONObject *shared.ResponseFormatJSONObjectParam // JSON 对象格式
	OfJSONSchema *shared.ResponseFormatJSONSchemaParam // JSON Schema 格式（推荐）
}
```

### 方式 1：纯文本格式（默认）

```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfText: &shared.ResponseFormatTextParam{},
}
```

**特征：**
- 这是默认行为
- 模型返回普通文本
- 不需要特殊提示

---

### 方式 2：JSON 对象格式

```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
}
```

**重要要求：**
- ✓ 可用于绝大多数模型
- ✓ 消息中必须包含 "json" 字样
- ✓ 返回有效的 JSON 对象
- ✗ 不提供 Schema 约束

**完整示例：**

```go
response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	Model: openai.ChatModelGPT4o,
	Messages: []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(`请返回 JSON 格式的用户信息：
{
  "name": "用户名",
  "age": 25,
  "email": "email@example.com"
}`),
	},
	ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
	},
})
```

---

### 方式 3：JSON Schema 格式（推荐）

这是最新、最严格的方式，支持自定义 Schema。

#### 基础结构

```go
schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        "response_schema",  // Schema 名称
	Description: openai.String("描述你期望的输出结构"),
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]string{
				"type":        "string",
				"description": "用户名",
			},
			"age": map[string]string{
				"type":        "integer",
				"description": "年龄",
			},
			"email": map[string]string{
				"type":        "string",
				"description": "邮箱地址",
			},
		},
		"required": []string{"name", "age", "email"},
		"additionalProperties": false,
	},
	Strict: openai.Bool(true), // 严格模式
}

response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	Model: openai.ChatModelGPT4o2024_08_06,
	Messages: []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("生成一个用户信息对象"),
	},
	ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: schemaParam,
		},
	},
})
```

#### 类型签名

```go
// 完整的 JSON Schema 参数类型
type ResponseFormatJSONSchemaJSONSchemaParam struct {
	Name        string         `json:"name" api:"required"`              // Schema 名称
	Description param.Opt[string] `json:"description,omitzero"`           // 描述
	Schema      map[string]any `json:"schema,omitzero" api:"required"`   // JSON Schema
	Strict      param.Opt[bool]   `json:"strict,omitzero"`                 // 严格模式
}
```

---

## 完整的可运行示例

### 示例 1：基本的 Chat Completion

```go
package main

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Write a haiku about programming"),
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(response.Choices[0].Message.Content)
}
```

---

### 示例 2：使用 JSON Object Format

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type UserInfo struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(`生成一个 JSON 格式的用户信息，包含 name、age、email 字段。
回复必须是有效的 JSON 对象。`),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		},
	})

	if err != nil {
		panic(err)
	}

	// 解析 JSON
	var userInfo UserInfo
	contentJson := response.Choices[0].Message.Content
	err = json.Unmarshal([]byte(contentJson), &userInfo)
	if err != nil {
		panic(err)
	}

	fmt.Printf("User Info: %+v\n", userInfo)
}
```

---

### 示例 3：使用 JSON Schema Format（推荐）

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type Person struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	// 定义 Schema
	schema := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "person",
		Description: openai.String("Information about a person"),
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]string{
					"type":        "string",
					"description": "Person's full name",
				},
				"age": map[string]string{
					"type":        "integer",
					"description": "Person's age in years",
				},
				"email": map[string]string{
					"type":        "string",
					"description": "Person's email address",
				},
			},
			"required":             []string{"name", "age", "email"},
			"additionalProperties": false,
		},
		Strict: openai.Bool(true),
	}

	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Generate a random person's information"),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schema,
			},
		},
	})

	if err != nil {
		panic(err)
	}

	// 解析结构化输出
	var person Person
	content := response.Choices[0].Message.Content
	err = json.Unmarshal([]byte(content), &person)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Person: %+v\n", person)
}
```

---

### 示例 4：流式响应 + Response Format

```go
package main

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	// 创建流式响应
	stream := client.Chat.Completions.NewStreaming(
		ctx,
		openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4o,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("输出 JSON 格式：{\"message\": \"你的问候\"}"),
			},
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
			},
		},
	)
	defer stream.Close()

	fmt.Println("Streaming response:")
	for stream.Next(ctx) {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	if stream.Err() != nil {
		panic(stream.Err())
	}
	fmt.Println()
}
```

---

### 示例 5：带工具调用 + Response Format

```go
package main

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("What's the weather in New York?"),
		},
		Tools: []openai.ChatCompletionToolUnionParam{
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "get_weather",
				Description: openai.String("Get current weather"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]string{
							"type": "string",
						},
					},
					"required": []string{"location"},
				},
			}),
		},
		// JSON Schema 可与工具一起使用
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name: "tool_response",
					Schema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"status": map[string]string{
								"type": "string",
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %+v\n", response)
}
```

---

## 常见错误和解决方案

### 错误 1：使用 JSON Object 但消息中无 "json" 关键字

**错误信息：**
```
400 Bad Request: { "error": { "message": "... must instruct model to produce JSON ..." } }
```

**原因：** 使用 `OfJSONObject` 时，需要在消息中明确告诉模型返回 JSON。

**解决方案：**
```go
// ❌ 错误
openai.UserMessage("返回用户信息"),

// ✓ 正确
openai.UserMessage("以 JSON 格式返回用户信息"),
// 或
openai.UserMessage("响应必须是有效的 JSON 对象"),
```

---

### 错误 2：JSON Schema Strict Mode 失败

**错误信息：**
```
模型输出不符合 schema
```

**原因：** Strict 模式下，模型必须严格遵守 schema，某些复杂逻辑可能无法完成。

**解决方案：**
```go
// 设置 Strict: false 以获得更灵活的响应
Strict: openai.Bool(false),

// 或优化 schema 定义，使其不过于严格
```

---

### 错误 3：ResponseFormat Union 多字段非零

**错误信息：**
```
Union 类型错误
```

**原因：** `ResponseFormatUnion` 中只能有一个字段非零。

**解决方案：**
```go
// ❌ 错误：两个字段都设置了
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfText: &shared.ResponseFormatTextParam{},
	OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
}

// ✓ 正确：只设置一个
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
}
```

---

### 错误 4：模型不支持 JSON Schema

**错误信息：**
```
模型不支持此 response_format
```

**原因：** 某些较旧模型不支持 Structured Outputs（JSON Schema）。

**支持的模型：**
- ✓ `gpt-4o-2024-08-06` 及更新版本
- ✓ `gpt-4-turbo-2024-04-09` 及更新版本
- ✗ 较旧的 `gpt-3.5-turbo` 版本

**解决方案：**
```go
// 使用支持的模型
Model: openai.ChatModelGPT4o,
// Model: openai.ChatModelGPT4o2024_08_06,
// Model: openai.ChatModelGPT4Turbo,
```

---

## 关键差异：v1 vs v3

| 功能 | v1 版本 | v3 版本 |
|------|---------|---------|
| Client 创建 | `openai.NewClient()` | `openai.NewClient()` |
| Chat API | `client.CreateChatCompletion()` | `client.Chat.Completions.New()` |
| Response Format | 指针类型 | Union 类型 |
| JSON Schema | 不支持 | ✓ 支持（推荐）|
| Structured Outputs | 基础支持 | ✓ 完整支持 |

---

## 最佳实践

1. **始终使用 Context**：设置合理的超时
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

2. **错误处理**：使用 `errors.As` 检查 API 错误
   ```go
   var apierr *openai.Error
   if errors.As(err, &apierr) {
       println(string(apierr.DumpRequest(true)))
   }
   ```

3. **JSON Schema 优于 JSON Object**：更严格、更可靠
   ```go
   // 推荐
   ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
       OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{...},
   }
   ```

4. **流式响应用于大输出**
   ```go
   stream := client.Chat.Completions.NewStreaming(ctx, params)
   ```

5. **验证模型支持性**：在使用新功能前检查模型兼容性

---

## 官方资源

- 📖 [官方 GitHub 仓库](https://github.com/openai/openai-go)
- 📚 [API 参考文档](https://pkg.go.dev/github.com/openai/openai-go/v3)
- 🔗 [OpenAI Platform 文档](https://platform.openai.com/docs)
- 💡 [Structured Outputs 指南](https://platform.openai.com/docs/guides/structured-outputs)

---

## 版本信息

- **Library Version**: v3.31.0
- **Release Date**: 2026-04-08
- **Go Version**: 1.22+
- **Latest Changes**: 支持短期 Token、改进 Web Search 等

