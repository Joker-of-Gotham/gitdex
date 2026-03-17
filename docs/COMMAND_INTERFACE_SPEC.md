# GitDex 命令接口规范（V3）

## 统一语义层

V3 将用户输入统一映射为 `object-action` 语义：

- `/goal <text>` -> `goal.create`
- `/run accept|all` -> `suggestion.execute`
- `/mode manual|auto|cruise` -> `mode.set`
- `/config ...` -> `config.set`
- `/creative` -> `creative.run`
- `/analyze` -> `flow.analyze`
- `/interval <sec>` -> `cruise.set_interval`
- `/help` -> `ui.help`
- `/test` -> `llm.probe`
- `/clear` -> `ui.clear_log`

## 设计收益

- slash 命令、内部分发、后续自动化接口共享同一语义基线。
- 便于扩展到 API / MCP 风格调用，不再绑定单一输入形式。
- 可在 object-action 层统一做审计、权限与观测。

## 扩展规则

- 新命令必须先定义 object-action，再加输入别名。
- object-action 名称使用 `object.action` 形式，使用小写 snake/kebab-free。
- 语义冲突时，以 object-action 为准，UI 文本只是映射层。
