# Story 8.1: LLM 实时对话集成

Status: ready-for-dev

## Story

As a Gitdex 用户,
I want 在 Chat 视图中输入自然语言即可获得 LLM 的实时流式回复,
So that 我可以通过对话获取建议、执行操作、理解仓库状态。

## 验收标准

1. 用户在 Composer 提交自然语言 → Chat 视图显示 LLM 流式逐字回复
2. LLM 配置缺失 → 友好错误提示，引导前往 Settings
3. 流式回复中 Esc/Ctrl+C → 中断请求，保留已收到内容
4. 多轮对话 → 上下文窗口自动管理（滑动窗口 20 条或 4K tokens）
5. 切换 Provider → 下次对话使用新 Provider，无需重启

## 任务 / 子任务

- [ ] T1: 创建会话管理器 `internal/llm/chat/session.go` (AC: #4)
  - [ ] T1.1: `Session` struct — messages []ChatMessage, maxTokens, maxMessages
  - [ ] T1.2: `AddMessage()` — 追加消息并触发窗口裁剪
  - [ ] T1.3: `GetContext()` — 返回当前窗口内的消息列表
  - [ ] T1.4: `Clear()` — 清空会话
- [ ] T2: 接入 LLM 到 app.go (AC: #1, #2)
  - [ ] T2.1: `app.Model` 新增 `llmProvider adapter.Provider` 字段
  - [ ] T2.2: `app.Model` 新增 `chatSession *chat.Session` 字段
  - [ ] T2.3: `handleSubmit` 非命令路径: 构造 ChatRequest → 调用 StreamChatCompletion
  - [ ] T2.4: 配置缺失检测: provider == nil 时返回错误提示
- [ ] T3: 流式消息类型 (AC: #1)
  - [ ] T3.1: `StreamChunkMsg{Content string, Done bool}` 定义
  - [ ] T3.2: `StreamErrorMsg{Error error}` 定义
- [ ] T4: ChatView 流式渲染 (AC: #1)
  - [ ] T4.1: 处理 StreamChunkMsg — 追加到最后一条 assistant 消息
  - [ ] T4.2: 处理 StreamChunkMsg{Done: true} — 标记完成
  - [ ] T4.3: 流式中渲染光标指示器 `▌`
- [ ] T5: 错误处理 (AC: #2)
  - [ ] T5.1: API 超时 → "LLM 请求超时，请检查网络连接"
  - [ ] T5.2: 401/403 → "API Key 无效，请在 Settings 中检查"
  - [ ] T5.3: 网络断开 → "无法连接到 LLM 服务"
- [ ] T6: 中断机制 (AC: #3)
  - [ ] T6.1: `app.Model` 新增 `streamCancel context.CancelFunc`
  - [ ] T6.2: Esc 在流式中 → 调用 cancel()
  - [ ] T6.3: 中断后追加 "（已中断）" 标记
- [ ] T7: Provider 热切换 (AC: #5)
  - [ ] T7.1: `handleConfigSave` 时重建 Provider: `adapter.NewProviderFromConfig(...)`

## Dev Notes

### 已有基础

- `internal/llm/adapter/provider.go` 定义了 `Provider` 接口:
  ```go
  type Provider interface {
      ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
      StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
  }
  ```
- `internal/llm/adapter/factory.go` 的 `NewProviderFromConfig(provider, model, apiKey, endpoint)` 可直接使用
- `internal/tui/views/chat.go` 已有 Message 类型（role: user/system/assistant/info/error）和 AppendMessage
- `internal/tui/app/app.go` 的 `handleSubmit` 当前对非命令输入返回静态占位回复

### 流式渲染架构

```
用户提交 → handleSubmit → 创建 assistant 占位消息 → 启动 goroutine:
  StreamChatCompletion(ctx, req) → chan ChatResponse
  for chunk := range chan {
      发送 StreamChunkMsg{Content: chunk.Content}
  }
  发送 StreamChunkMsg{Done: true}
```

ChatView.Update 处理 StreamChunkMsg 时追加内容到最后一条 assistant 消息。

### 会话管理器设计

```go
type Session struct {
    messages   []adapter.ChatMessage
    maxMessages int  // 默认 20
    maxTokens   int  // 默认 4096
    systemPrompt string
}
```

systemPrompt 固定前缀:
```
你是 Gitdex，一个仓库运维智能助手。你可以帮助用户理解仓库状态、执行 Git 操作、管理 GitHub 协作对象。
当前仓库: {owner}/{repo}
当前分支: {branch}
```

### Project Structure Notes

- 新建 `internal/llm/chat/` 包，与已有 `internal/llm/adapter/` 平级
- 消息类型添加到 `internal/tui/views/messages.go`（已有 FileContentMsg 等定义）
- Provider 生命周期由 `app.Model` 管理，不使用全局单例

### References

- [Source: internal/llm/adapter/provider.go] — Provider 接口定义
- [Source: internal/llm/adapter/factory.go] — NewProviderFromConfig 工厂
- [Source: internal/tui/views/chat.go] — ChatView、Message 类型
- [Source: internal/tui/app/app.go#handleSubmit] — 当前提交处理逻辑
- [Reference: openai-agents-python/src/agents/models/] — 流式模型调用模式
- [Reference: symphony/SPEC.md] — Agent Runner 生命周期

## Dev Agent Record

### Agent Model Used

（待实现时填写）

### Completion Notes List

### File List
