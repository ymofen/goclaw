#
- 修改一下可以将下面文本进行解析
```
{
    "id": "kimi-k2-thinking-bailian",
    "name": "kimi-k2-thinking-bailian",
    "model_id": "kimi-k2-thinking",
    "api_key": "sk-8234d689595545eb83bea3985ae5e1ad",
    "url": "https://dashscope.aliyuncs.com/compatible-mode/v1"
}
```

- 支持流式输出

# 重构

- Message 消息
  - 存放目录 pkg/model
  - role
  - kind   类型 text|reasoning|stop
  - content string
- Memory 
  - 存放目录 pkg/model
  - list []Message
  - Add(msg Message)
- OpenAIChatFormatter OpenAIChat 请求格式化
  - 存放目录 pkg/openai
  - GetRequest() string  返回符合与 OpenAI 服务器请求消息格式格式
- OpenAIConfig 参数配置 
  - 存放目录 pkg/openai
  - ModelId, Url, SSE分帧, ApiKey  
- OpenAIModel 利用使用OpenAIConfig 与大模型服务器进行通讯
  - 存放目录 pkg/openai
  - Execute(mem Memory, formatter Formatter) (choices []Message, error)
  - OnSSEReply(msg Message);