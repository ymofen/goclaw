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

#
- OpenAIModel 
  - 请求事件，可以记录请求的所有数据包
  - 响应事件，可以记录收到的所有数据包
- demo中使用事件记录交互数据记录到history.log中

# tools 
- 提供一个工具管理类，
  - 可以提交注册工具函数
  - 提供执行工具接口


# 
- Toolkit 工具接口管理类
  - Register
  - Unregister
  - Execute
  - Definitions() []ToolDef
  
- AIModelRequestFormatter 接口 移到model中
  - GetRequest() string  序列化
    
- AIModelEvent 类 存放在model目录中
  - OnSSEReplyHandler func(msg model.Message)
  - OnRequestEvent func(body []byte)
  - OnResponse func(handler func(body []byte)
- AIModel AIModel接口，存放在model目录中
  - Execute() (choices []Message, error)

- OpenAIChatModel 
  - 创建是传入ModelEvent和OpenAIConfig
  - 实现AIModel接口  

- OpenAIRequestFormatter
  - Toolkit model.Toolkit
  - Prompt 提示词
  - Memory 记忆



