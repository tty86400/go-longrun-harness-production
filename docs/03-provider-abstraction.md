# 03. Provider Abstraction

## 为什么 provider 必须抽象

如果把 harness 直接写死到某个 SDK：

- 模型换一家，主循环要重写。
- review / summarize 不能轻易用不同模型。
- 迁移到私有部署或代理网关很痛苦。

所以这里把 provider 抽成统一接口：

```go
type Provider interface {
    Name() string
    Generate(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
```

主循环只依赖这个接口，不依赖任何厂商 SDK。

## 已实现 provider

### 1. openai_compatible

适合这类接口：

- `/v1/chat/completions`
- `messages` 数组
- 返回 `choices[0].message.content`

实现文件：`internal/provider/openai_compatible.go`

支持：

- 自定义 base URL / endpoint
- 认证头
- 超时
- JSON mode
- usage 解析
- raw response 审计

### 2. generic_json

适合不是 OpenAI 协议，但仍然是普通 JSON HTTP 接口的模型服务。

实现文件：`internal/provider/generic_json.go`

关键配置项：

- `input_mode`
  - `messages`
  - `prompt`
- `message_field_path`
- `prompt_field_path`
- `response_text_path`
- `response_id_path`
- `json_mode_field_path`
- `json_mode_value`

这让你可以把相同的 harness 主循环接到不同 JSON 协议的模型 API 上。

### 3. mock

实现文件：`internal/provider/mock.go`

它的用途是：

- 本地端到端自测
- CI 集成测试
- 学习主循环时避免外部 API 成本

## retry / backoff 为什么在 provider 层做

因为失败大多发生在这层：

- 网络抖动
- 429
- 5xx
- 代理网关偶发失败

如果把 retry 写在主循环，engine 会被很多 provider 细节污染；
放在 provider wrapper 里，engine 只看到“最终成功”或“最终失败”。

## strict JSON 的真实意义

这版 harness 没有把第一版做成 function calling，而是强制模型输出严格 JSON 协议。

这样做的好处：

- 更容易兼容任意模型 API。
- 对教学和调试更透明。
- 即使某些模型不支持原生工具调用，也能接入。

对应关键代码：

- `provider.ExtractJSONObjectText()`
- `decodeJSONStrict()`
- `validateDecision()`

## 什么时候需要自己写一个 provider

`generic_json` 已经覆盖了很多 JSON REST 场景，但这些情况仍然建议单独写 adapter：

- 需要签名认证而不只是简单 header
- 需要流式 SSE 特殊处理
- 输入消息结构不是简单 `messages` 或单 prompt
- 输出需要特殊拼接规则
- 需要强定制的 usage 统计

写法也很直接：

1. 新建一个 struct 实现 `Provider`。
2. 在 `factory.go` 里注册 kind。
3. 主循环完全不用改。

这就是抽象边界设计的价值。
