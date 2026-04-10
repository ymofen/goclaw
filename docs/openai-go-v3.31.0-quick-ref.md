# OpenAI-Go v3.31.0 快速参考

## 核心 API 调用模式

```go
// 步骤 1：创建 Client
client := openai.NewClient()

// 步骤 2：准备参数
params := openai.ChatCompletionNewParams{
	Model:          openai.ChatModelGPT4o,
	Messages:       []openai.ChatCompletionMessageParamUnion{...},
	ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{...},
}

// 步骤 3：发送请求
response, err := client.Chat.Completions.New(ctx, params)

// 步骤 4：获取响应
content := response.Choices[0].Message.Content
```

---

## Response Format 三种用法速查表

### 1️⃣ 文本（默认）
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfText: &shared.ResponseFormatTextParam{},
}
```
- 场景：普通对话
- 限制：无

### 2️⃣ JSON Object
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
}
```
- 场景：需要 JSON 输出
- 限制：消息中需包含 "json" 
- 模型：大多数模型支持

### 3️⃣ JSON Schema（推荐）✨
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
	OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
		JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:   "schema_name",
			Schema: map[string]any{...},
			Strict: openai.Bool(true),
		},
	},
}
```
- 场景：严格的结构化输出
- 限制：仅新模型支持
- 优势：类型安全、API 保证输出符合 schema

---

## 消息构造快速参考

```go
// 用户消息
openai.UserMessage("你的问题")

// 系统消息
openai.SystemMessage("系统提示")

// 助手消息
openai.AssistantMessage("回复内容")

// 开发者消息
openai.DeveloperMessage("开发者消息")

// 自定义消息
openai.ChatCompletionMessageParamUnion{
	OfUser: &openai.ChatCompletionUserMessageParam{
		Content: openai.ChatCompletionUserMessageParamContentUnion{
			OfString: openai.String("内容"),
		},
	},
}
```

---

## 常用模型常量

```go
openai.ChatModelGPT5_2        // 最新模型
openai.ChatModelGPT4o         // 推荐用于 JSON Schema
openai.ChatModelGPT4o2024_08_06
openai.ChatModelGPT4Turbo
openai.ChatModelGPT35Turbo
```

---

## 必需参数 vs 可选参数

### 必需参数 ✓
```go
Model:    openai.ChatModelGPT4o,           // string
Messages: []openai.ChatCompletionMessageParamUnion{...}, // 数组
```

### 常用可选参数
```go
MaxTokens:      openai.Int(1000),          // param.Opt[int64]
Temperature:    openai.Float(0.7),         // param.Opt[float64]
TopP:           openai.Float(0.9),         // param.Opt[float64]
Seed:           openai.Int(42),            // param.Opt[int64]
ResponseFormat: ...,                        // Union 类型
Tools:          []openai.ChatCompletionToolUnionParam{...},
Stop:           openai.ChatCompletionNewParamsStopUnion{...},
```

---

## 流式 vs 非流式

### 非流式（一次性获取）
```go
response, err := client.Chat.Completions.New(ctx, params)
if err != nil {
	panic(err)
}
content := response.Choices[0].Message.Content
```

### 流式（实时接收）
```go
stream := client.Chat.Completions.NewStreaming(ctx, params)
defer stream.Close()

for stream.Next(ctx) {
	chunk := stream.Current()
	if len(chunk.Choices) > 0 {
		// 处理流数据
		content := chunk.Choices[0].Delta.Content
	}
}

if stream.Err() != nil {
	panic(stream.Err())
}
```

---

## 常见参数值

### 模型选择
```go
gppt-5.2           // 最新推理模型
gpt-4o             // 视觉和推理
gpt-4-turbo        // 快速 + 上下文长
gpt-3.5-turbo      // 成本低效
```

### 温度选择
```go
0.0    // 最确定性（数学计算）
0.7    // 平衡（一般应用）
1.5+   // 最创意（创意写作）
```

### Stop 序列
```go
Stop: openai.ChatCompletionNewParamsStopUnion{
	OfString: openai.String("\n"),  // 遇到换行停止
}
```

---

## 错误处理最佳实践

```go
response, err := client.Chat.Completions.New(ctx, params)
if err != nil {
	// 检查 API 错误
	var apierr *openai.Error
	if errors.As(err, &apierr) {
		statusCode := apierr.StatusCode
		rawReq := string(apierr.DumpRequest(true))
		rawRes := string(apierr.DumpResponse(true))
		
		log.Printf("API Error: %d\nRequest: %s\nResponse: %s", 
			statusCode, rawReq, rawRes)
	}
	return err
}
```

---

## Context 设置

```go
// 带超时的 Context（推荐）
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 或与值绑定
ctx := context.WithValue(context.Background(), "request_id", "123")
```

---

## JSON Schema 定义模板

```go
schema := openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name: "response_schema",
	Description: openai.String("描述"),
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"field1": map[string]string{
				"type":        "string",
				"description": "字段描述",
			},
			"field2": map[string]string{
				"type":        "integer",
				"description": "整数字段",
			},
		},
		"required":             []string{"field1", "field2"},
		"additionalProperties": false,
	},
	Strict: openai.Bool(true),  // 严格模式
}
```

---

## 常见问题速查

| 问题 | 解决方案 |
|------|---------|
| 如何使用自定义 API？ | `option.WithBaseURL("url")` |
| 如何设置超时？ | `context.WithTimeout()` / `option.WithRequestTimeout()` |
| JSON 输出为什么失败？ | 消息中需包含 "json" 关键字 |
| JSON Schema 不支持？ | 检查模型版本，需 gpt-4o 或更新 |
| 如何添加自定义 Header？ | `option.WithHeader("key", "value")` |
| 如何重试失败请求？ | `option.WithMaxRetries(n)` |

---

## 模型兼容性矩阵

| 功能 | gpt-3.5-turbo | gpt-4-turbo | gpt-4o | gpt-5.2 |
|------|--|--|--|--|
| 基本对话 | ✓ | ✓ | ✓ | ✓ |
| JSON Object | ✓ | ✓ | ✓ | ✓ |
| JSON Schema | ✗ | ✓ | ✓ | ✓ |
| 视觉 | ✗ | ✓ | ✓ | ✓ |
| 音频 | ✗ | ✗ | ✓ | ✓ |
| 工具调用 | ✓ | ✓ | ✓ | ✓ |

---

## 一行代码示例

```go
// 最简单的调用
resp, _ := openai.NewClient().Chat.Completions.New(context.Background(), 
    openai.ChatCompletionNewParams{
        Model: openai.ChatModelGPT4o,
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.UserMessage("Hello"),
        },
    })
fmt.Println(resp.Choices[0].Message.Content)
```

---

## 导入路径

```go
import (
	"github.com/openai/openai-go/v3"              // 主包
	"github.com/openai/openai-go/v3/option"       // 选项
	"github.com/openai/openai-go/v3/shared"       // 共享类型
)
```

---

## 版本检查

```go
// go.mod 中应包含
require github.com/openai/openai-go/v3 v3.31.0
```

获取最新版本：
```bash
go get -u github.com/openai/openai-go/v3@v3.31.0
```

---

**最后更新**: 2026-04-08 | **版本**: v3.31.0
