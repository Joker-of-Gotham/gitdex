# Story 8.2: 剪贴板与右键粘贴修复

Status: ready-for-dev

## Story

As a Gitdex 用户,
I want 在终端中使用右键粘贴、Ctrl+V 粘贴、以及选择文本右键复制,
So that 我可以在 TUI 中自由地复制粘贴内容。

## 验收标准

1. 终端右键粘贴 → Composer 正确接收粘贴内容（含多行和特殊字符）
2. Content 区域选择文本 → 文本正确复制到系统剪贴板（不被 TUI 拦截）
3. 应用启动 → 自动启用 bracketed paste 支持
4. 跨终端一致性 — Windows Terminal、PowerShell、iTerm、gnome-terminal

## 任务 / 子任务

- [ ] T1: 启用 BracketedPaste (AC: #3)
  - [ ] T1.1: `cmd/gitdex/main.go` 传入 `tea.WithBracketedPaste()` 选项
  - [ ] T1.2: 验证 `tea.PasteMsg` 在 `app.Update` 中正确路由
- [ ] T2: 完善 Composer PasteMsg 处理 (AC: #1)
  - [ ] T2.1: 确认 `composer.Update` 中 `tea.PasteMsg` 分支存在且正确
  - [ ] T2.2: 多行粘贴: 保留换行符，不截断
  - [ ] T2.3: 特殊字符: Unicode、emoji、制表符等正确插入
  - [ ] T2.4: 中间位置粘贴: 在光标位置插入，不覆盖现有内容
- [ ] T3: Content 区域鼠标兼容 (AC: #2)
  - [ ] T3.1: 使用 `tea.WithMouseCellMotion()` 而非 `tea.WithMouseAllMotion()`
  - [ ] T3.2: 确保终端原生文本选择不被 TUI 事件拦截
  - [ ] T3.3: 不在 Content 区域注册 mouseDown/mouseUp 事件（保留给终端选择）
- [ ] T4: 跨平台验证 (AC: #4)
  - [ ] T4.1: Windows Terminal + PowerShell 测试
  - [ ] T4.2: bash/zsh + macOS Terminal 测试
  - [ ] T4.3: gnome-terminal / Konsole 测试

## Dev Notes

### 已有基础

- `internal/tui/components/composer.go` 已有 `tea.PasteMsg` 处理分支:
  ```go
  case tea.PasteMsg:
      c.insertText(string(msg.Content))
  ```
- `internal/tui/components/components_test.go` 已有粘贴测试（多行、特殊字符、中间位置、空字符串、非焦点）

### 关键修改点

1. **`cmd/gitdex/main.go`**: 当前 `tea.NewProgram(model)` 调用需添加选项:
   ```go
   p := tea.NewProgram(model,
       tea.WithBracketedPaste(),
       tea.WithMouseCellMotion(),
   )
   ```

2. **Bubble Tea v2 PasteMsg 机制**: 终端发送 `\e[200~...内容...\e[201~`，Bubble Tea 解析为 `tea.PasteMsg{Content: "..."}`。不启用 `tea.WithBracketedPaste()` 时，粘贴内容会被逐字符作为 `KeyPressMsg` 处理。

3. **鼠标模式选择**:
   - `tea.WithMouseCellMotion()` — 仅跟踪按住移动，不拦截终端原生选择
   - `tea.WithMouseAllMotion()` — 跟踪所有鼠标移动，会拦截终端原生选择
   - 选择 `CellMotion` 可以保留复制能力

### Project Structure Notes

- 修改范围极小: `cmd/gitdex/main.go` + 可能的 `composer.go` 微调
- 此 Story 不需要新建任何文件
- 与 8.1 可完全并行开发

### References

- [Source: internal/tui/components/composer.go#L32-L33] — SetFocused/Focused
- [Source: internal/tui/components/components_test.go] — 已有粘贴测试
- [Reference: bubbletea/examples/] — Bubble Tea 粘贴和鼠标示例
- [Reference: charm.land/bubbletea/v2] — PasteMsg、WithBracketedPaste 文档

## Dev Agent Record

### Agent Model Used

（待实现时填写）

### Completion Notes List

### File List
