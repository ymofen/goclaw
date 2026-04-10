# OpenAI-Go ResponseFormat 快速参考

## 🎯 最常用的方法（JSON 对象格式）

```go
package main

import (
    "context"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

func main() {
    // 创建客户端
    client := openai.NewClient(
        option.WithAPIKey("your-api-key"),
    )

    // 创建请求参数
    params := openai.ChatCompletionNewParams{
        Model:     openai.ChatModel("gpt-4o"),
        MaxTokens: openai.Int(1024),
        Messages: []openai.ChatCompletionMessageParamUnion{
            // ⚠️ 重要：提及 "json" 关键字
            openai.UserMessage("请以 JSON 格式回复，告诉我你的名字和版本"),
        },
        // ✅ 设置 JSON 对象格式
        ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject),
    }

    // 调用 API
    response, err := client.Chat.Completions.New(context.Background(), params)
    if err != nil {
        panic(err)
    }

    // 获取结果（已是 JSON 格式）
    if len(response.Choices) > 0 {
        println(response.Choices[0].Message.Content)
    }
}
```

## 📚 三种格式对比

| 格式 | 常量/构造 | 用途 | 模型要求 | 优点 | 缺点 |
|------|---------|------|---------|------|------|
| **json_object** | `openai.ChatCompletionResponseFormatJSONObject` | 返回有效 JSON | gpt-4o+ | 简单易用 | 无结构控制，需消息提及 json |
| **json_schema** | `shared.ResponseFormatJSONSchemaParam{}` | 严格 JSON Schema | gpt-4o+ | 完全控制输出结构 | 配置复杂 |
| **text** | `shared.ResponseFormatTextParam{Type: "text"}` | 纯文本（默认） | 所有 | 最灵活 | 无格式保证 |

## ⚡ 三行代码模板

### JSON 对象格式
```go
ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject)
```

### JSON Schema 格式
```go
ResponseFormat: openai.F(shared.ResponseFormatJSONSchemaParam{
    JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{...}
})
```

### 纯文本格式
```go
ResponseFormat: openai.F(shared.ResponseFormatTextParam{Type: "text"})
```

## 🔴 常见错误

### 错误 1: 忘记在消息中提及 "json"
```
InternalError.Algo.InvalidParameter: 'messages' must contain the word 'json'
```
**修复**：在 UserMessage 中包含 "json"
```go
✅ openai.UserMessage("以 JSON 格式回复")
❌ openai.UserMessage("直接告诉我结果")
```

### 错误 2: ResponseFormat 为 nil
```go
❌ ResponseFormat: nil  // 类型检查错误

✅ ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject)
```

### 错误 3: 使用字符串而不是常量
```go
❌ ResponseFormat: openai.F("json_object")  // 类型错误

✅ ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject)
```

## 🔗 与你的项目相关

### 当前代码位置
- [cmd/openai/main.go](../../cmd/openai/main.go#L70-L71) - 已有示例

### 已有的 Formatter 支持
- [pkg/openai/formatter.go](../../pkg/openai/formatter.go) - ResponseFormat map[string]interface{}
- [pkg/openaisdk/types.go](../../pkg/openaisdk/types.go) - ChatCompletionRequest.ResponseFormat

### 最佳实践集成建议
在你的 Formatter 中支持 ResponseFormat：
```go
type RequestFormatter struct {
    ResponseFormat openai.ChatCompletionNewParamsResponseFormatUnion
}

func (f *RequestFormatter) GetRequest() ([]byte, error) {
    params := openai.ChatCompletionNewParams{
        Model:          f.Model,
        Messages:       f.Messages,
        ResponseFormat: openai.F(f.ResponseFormat), // 集成
    }
    // ...
}
```

## 📖 完整文档参考

- **主文档**: [response_format_guide.md](./response_format_guide.md)
- **代码示例**: [response_format_examples.go](../../cmd/openai/response_format_examples.go)
- **官方 SDK**: GitHub - openai/openai-go v1.12.0

## 📋 核心 API 查询表

```go
// 创建一个包含所有支持类型的请求
ctx := context.Background()
client := openai.NewClient(option.WithAPIKey("key"))

// 方式 A: JSON 对象（最简单）
params1 := openai.ChatCompletionNewParams{
    Messages: []openai.ChatCompletionMessageParamUnion{...},
    ResponseFormat: openai.F(openai.ChatCompletionResponseFormatJSONObject),
}

// 方式 B: JSON Schema（最严格）
params2 := openai.ChatCompletionNewParams{
    Messages: []openai.ChatCompletionMessageParamUnion{...},
    ResponseFormat: openai.F(shared.ResponseFormatJSONSchemaParam{
        JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
            Name:   "my_schema",
            Schema: map[string]interface{}{...},
        },
    }),
}

// 方式 C: 纯文本（默认）
params3 := openai.ChatCompletionNewParams{
    Messages: []openai.ChatCompletionMessageParamUnion{...},
    ResponseFormat: openai.F(shared.ResponseFormatTextParam{
        Type: "text",
    }),
}

// 调用 API
response, _ := client.Chat.Completions.New(ctx, params1)
```

## ✅ 完整检查清单

- [ ] 导入了 `github.com/openai/openai-go` 和 `github.com/openai/openai-go/shared`
- [ ] 使用 `openai.F()` 包装 ResponseFormat
- [ ] 使用了官方常量而不是字符串
- [ ] 如果使用 `json_object`，消息中包含 "json" 关键字
- [ ] 错误处理检查 API 返回的 `error` 字段
- [ ] 测试过流式和非流式场景
