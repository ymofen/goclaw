# OpenAI-Go v3.31.0 搜索结果总结

## 搜索完成

已成功获取并整理了 **openai-go v3.31.0** 的官方文档和实现方式。

---

## 📋 创建的文档

### 1. **[openai-go-v3.31.0-guide.md](docs/openai-go-v3.31.0-guide.md)** ⭐ 主文档
- **内容**: 完整的官方指南（12000+ 字）
- **包括**:
  - Client 创建的完整方式
  - Chat Completions 请求详解
  - ResponseFormat 参数详解
  - 5 个完整可运行示例
  - 常见错误和解决方案
  - 最佳实践
  - 官方资源链接

### 2. **[openai-go-v3.31.0-quick-ref.md](docs/openai-go-v3.31.0-quick-ref.md)** ⚡ 快速参考
- **内容**: 速查表（800+ 字）
- **包括**:
  - API 调用模式一行代码
  - 三种 ResponseFormat 快速对比
  - 消息构造快速参考
  - 常用参数值
  - 错误处理速查
  - 模型兼容性矩阵

### 3. **[openai_v3_examples.go](cmd/examples/openai_v3_examples.go)** 💻 代码示例
- **内容**: 10 个完全可运行的示例
- **包括**:
  1. 基本 Chat Completion
  2. 自定义 Client 配置
  3. JSON Object 格式
  4. JSON Schema 格式（推荐）
  5. 流式响应
  6. 多轮对话
  7. 系统提示词和参数
  8. 错误处理
  9. 三种 ResponseFormat 对比
  10. 内容类型处理

---

## 🎯 关键发现

### 1️⃣ Client 创建

```go
// 最简单方式
client := openai.NewClient()

// 自定义配置
client := openai.NewClient(
    option.WithAPIKey("sk-..."),
    option.WithBaseURL("https://..."),
)
```

### 2️⃣ Chat Completions 请求

```go
response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model:    openai.ChatModelGPT4o,
    Messages: []openai.ChatCompletionMessageParamUnion{...},
    ResponseFormat: ..., // 可选
})
```

### 3️⃣ ResponseFormat 三种方式

#### 方式 A: 纯文本（默认）
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
    OfText: &shared.ResponseFormatTextParam{},
}
```

#### 方式 B: JSON Object
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
    OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
}
```
- ✓ 需要消息中包含 "json" 关键字
- ✓ 大多数模型支持

#### 方式 C: JSON Schema（推荐）✨
```go
ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
    OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
        JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
            Name: "schema_name",
            Schema: map[string]any{...},
            Strict: openai.Bool(true),
        },
    },
}
```
- ✓ 最严格、最可靠
- ✓ 支持自定义 Schema
- ✓ 仅 gpt-4o 等新模型支持

---

## 📊 v3 vs v1 对比

| 方面 | v1 版本 | v3 版本 |
|------|---------|---------|
| **包导入** | `openai` | `openai/v3` |
| **ResponseFormat 类型** | 指针结构体 | Union 类型 |
| **JSON Schema** | 基础支持 | ✓ 完整支持 |
| **Structured Outputs** | 基础 | ✓ 完整实现 |
| **API 调用** | `CreateChatCompletion()` | `Chat.Completions.New()` |

---

## 🔍 从官方获取的信息

### 源代码参考
- **文件**: `chatcompletion.go` (~3400+ 行)
- **Union 定义**: L3279-3284
- **ResponseFormat 处理**: L3166-3180
- **初始化函数**: L3311-3370

### 官方示例
- **基本对话**: `examples/chat-completion/main.go`
- **结构化输出**: `examples/structured-outputs/main.go` ⭐
- **工具调用**: `examples/chat-completion-tool-calling/main.go`
- **流式响应**: `examples/responses-streaming/main.go`

### 版本信息
- **发布日期**: 2026-04-08
- **最低 Go 版本**: 1.22+
- **新功能**: 短期 Token 支持、Web Search 改进

---

## 🚀 快速开始

### 最简单的例子
```go
package main

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3"
)

func main() {
	client := openai.NewClient()
	response, _ := client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4o,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello!"),
			},
		},
	)
	fmt.Println(response.Choices[0].Message.Content)
}
```

### 使用 JSON Schema
```go
schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name: "person",
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]string{"type": "string"},
			"age": map[string]string{"type": "integer"},
		},
		"required": []string{"name", "age"},
	},
}

response, _ := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	Model: openai.ChatModelGPT4o,
	Messages: []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("生成一个人物"),
	},
	ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: schemaParam,
		},
	},
})
```

---

## ⚠️ 常见陷阱

1. **JSON Object 格式缺少 "json" 关键字**
   ```
   ❌ 错误：openai.UserMessage("返回数据")
   ✓ 正确：openai.UserMessage("返回 JSON 格式的数据")
   ```

2. **ResponseFormat Union 类型多字段非零**
   ```
   ❌ 错误：OfText 和 OfJSONObject 同时设置
   ✓ 正确：只设置其中一个
   ```

3. **JSON Schema 在旧模型中不可用**
   ```
   ❌ 错误：gpt-3.5-turbo + JSON Schema
   ✓ 正确：gpt-4o + JSON Schema
   ```

---

## 📚 官方资源链接

- 🔗 **GitHub 仓库**: https://github.com/openai/openai-go
- 📖 **Pkg.Dev 文档**: https://pkg.go.dev/github.com/openai/openai-go/v3
- 🎓 **OpenAI Platform**: https://platform.openai.com/docs
- 🔧 **API 参考**: https://platform.openai.com/docs/api-reference/chat
- 💡 **Structured Outputs 指南**: https://platform.openai.com/docs/guides/structured-outputs

---

## 📝 文件清单

| 文件 | 说明 | 大小 |
|------|------|------|
| `docs/openai-go-v3.31.0-guide.md` | 完整指南 | ⭐⭐⭐ |
| `docs/openai-go-v3.31.0-quick-ref.md` | 速查表 | ⭐⭐ |
| `cmd/examples/openai_v3_examples.go` | 代码示例 | ⭐⭐⭐ |

---

## 🎓 学习路径

### 初学者
1. 阅读 **quick-ref.md** - 5 分钟了解基本概念
2. 运行 **示例 1** - 基本对话
3. 阅读 **guide.md** 的前两章

### 中级开发者
1. 完整阅读 **guide.md**
2. 运行 **示例 3-5** - JSON 格式和流式
3. 实现自己的 Schema

### 高级开发者
1. 查阅 **quick-ref.md** 的矩阵
2. 研究官方 GitHub 示例
3. 参考 `chatcompletion.go` 源码

---

## ✅ 验证清单

使用这些文档后，您应该能够：

- [ ] 创建 OpenAI Client
- [ ] 发送基本 Chat Completions 请求
- [ ] 使用三种 ResponseFormat
- [ ] 定义和使用 JSON Schema
- [ ] 处理流式响应
- [ ] 进行多轮对话
- [ ] 正确处理错误
- [ ] 选择合适的模型

---

## 📞 获取帮助

### 常见问题
- 如何使用自定义 API？ → 见 **guide.md** 第 1 章
- JSON 输出为什么失败？ → 见 **guide.md** 第 7 章
- 如何实现结构化输出？ → 见 **示例 4**

### 官方支持
- GitHub Issues: https://github.com/openai/openai-go/issues
- OpenAI 文档: https://platform.openai.com/docs

---

**生成时间**: 2026-04-09  
**源版本**: openai-go v3.31.0  
**最后更新**: 2026-04-09

