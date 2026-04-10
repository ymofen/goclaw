# OpenAI-Go v1.12.0 ResponseFormat 参数设置指南

## 概述

在 openai-go 官方 SDK v1.12.0 中，`response_format` 用于控制 API 返回的数据格式。该参数在 `ChatCompletionNewParams` 中配置。

## 关键 API 类型

### 1. ResponseFormat Union 类型
```go
type ChatCompletionNewParamsResponseFormatUnion struct {
    OfText       *shared.ResponseFormatTextParam       // 纯文本格式
    OfJSONSchema *shared.ResponseFormatJSONSchemaParam // JSON Schema 格式  
    OfJSONObject *shared.ResponseFormatJSONObjectParam // JSON 对象格式
    paramUnion
}
```

## 核心常量

### JSON 对象格式常量

```go
// 在官方 SDK 中定义的常量（参考 cmd/openai/main.go）
openai.ChatCompletionResponseFormatJSONObject
```

### 使用方式

**方式 1：使用内置常量（推荐）**
```go
ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject)
```

**方式 2：显式构造**
```go
ResponseFormat: openai.F(shared.ResponseFormatJSONObjectParam{
    Type: "json_object",
})
```

## 三种格式详解

### 1. JSON 对象格式 (`json_object`)
- **用途**：强制 API 返回有效的 JSON 对象
- **特点**：简单易用，自动验证返回值是 JSON
- **限制**：不支持自定义 schema，只保证返回有效 JSON

```go
params := openai.ChatCompletionNewParams{
    Model:     openai.ChatModel("gpt-4o"),
    MaxTokens: openai.Int(1024),
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("在消息中提及 JSON，例如：请以 JSON 格式回复"),
    },
    ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject),
}
```

**重要提示**：使用 `json_object` 时，消息中必须包含 "json" 相关的单词，提醒 API 返回 JSON 格式，否则会报错：
```
'messages' must contain the word 'json' in some form, to use 'response_format' of type 'json_object'
```

### 2. JSON Schema 格式 (`json_schema`)
- **用途**：使用 JSON Schema 严格定义返回结构
- **特点**：提供最高的结构控制，确保返回值符合 schema
- **支持**：gpt-4o 及更高版本模型

```go
jsonSchema := shared.ResponseFormatJSONSchemaJSONSchemaParam{
    Name: "user_schema",
    Description: "用户信息结构",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "name": map[string]interface{}{
                "type": "string",
            },
            "age": map[string]interface{}{
                "type": "integer",
            },
        },
        "required": []string{"name", "age"},
    },
    Strict: openai.Bool(true),
}

params := openai.ChatCompletionNewParams{
    Model:          openai.ChatModel("gpt-4o"),
    MaxTokens:      openai.Int(1024),
    Messages:       []openai.ChatCompletionMessageParamUnion{...},
    ResponseFormat: openai.F(shared.ResponseFormatJSONSchemaParam{
        JSONSchema: jsonSchema,
    }),
}
```

### 3. 纯文本格式 (`text`)
- **用途**：返回纯文本内容（默认）
- **特点**：不做任何格式限制

```go
ResponseFormat: openai.F(shared.ResponseFormatTextParam{
    Type: "text",
})
```

## 实际工作区示例

在你的 `cmd/openai/main.go` 中的用法：

```go
params := openai.ChatCompletionNewParams{
    Model:     openai.ChatModel(config.ModelID),
    MaxTokens: openai.Int(1024),
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("你好，请简要介绍一下你自己"),
    },
    // 使用 response_format 参数强制 API 返回 JSON 格式
    ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject),
}
```

## 完整的设置步骤

### 步骤 1：导入必要的包
```go
import (
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    "github.com/openai/openai-go/shared"
)
```

### 步骤 2：创建客户端
```go
opts := []option.RequestOption{
    option.WithAPIKey(apiKey),
    option.WithBaseURL(baseURL), // 如果使用自定义端点
}
client := openai.NewClient(opts...)
```

### 步骤 3：构造带有 ResponseFormat 的参数
```go
params := openai.ChatCompletionNewParams{
    Model:     openai.ChatModel("gpt-4o"),
    MaxTokens: openai.Int(1024),
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("你的提示词..."),
    },
    ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject),
}
```

### 步骤 4：调用 API
```go
response, err := client.Chat.Completions.New(ctx, params)
if err != nil {
    // 处理错误
}
```

## 常见错误及解决方案

### 错误 1: "must contain the word 'json'"
```
InternalError.Algo.InvalidParameter: 'messages' must contain the word 'json' 
in some form, to use 'response_format' of type 'json_object'
```

**解决**：在消息中提及 "json" 字样
```go
// ✅ 正确
openai.UserMessage("请以 JSON 格式回复...")

// ❌ 错误
openai.UserMessage("请回复...")
```

### 错误 2: ResponseFormat 类型不匹配
使用 `openai.F()` 包装器包装 ResponseFormat 值，以正确指定可选参数。

### 错误 3: 模型版本不支持
JSON Schema 格式仅在 gpt-4o 及以上版本支持。

## API v1.12.0 中的相关文件

- 主定义：`github.com/openai/openai-go` 包中的 `chatcompletion.go`
- 共享类型：`github.com/openai/openai-go/shared` 中的定义
- 常量：`github.com/openai/openai-go/shared/constant` 

## 参考资源

- 关键文件：
  - `d:\Go\go\pkg\mod\github.com\openai\openai-go@v1.12.0\chatcompletion.go`
  - `shared.ResponseFormatTextParam`
  - `shared.ResponseFormatJSONSchemaParam`
  - `shared.ResponseFormatJSONObjectParam`

- 你的项目中的例子：
  - `e:\workspace\ai\goclaw\cmd\openai\main.go`（第 71 行）

## 最佳实践总结

1. ✅ 使用 `openai.F()` 包装 ResponseFormat 参数
2. ✅ 使用 JSON 对象格式时，消息中必须包含 "json" 关键字
3. ✅ 优先使用内置常量而不是字符串字面量
4. ✅ 对于严格的数据结构需求，使用 JSON Schema 格式
5. ✅ 确保模型版本支持所选格式（JSON Schema 需要较新模型）
6. ✅ 始终正确处理 API 错误，检查是否满足格式要求
