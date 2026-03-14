# gitdex

<div align="center">
  <img src="assets/readme-hero.svg" alt="gitdex 预览图" width="960" />
  <p><strong>面向本地仓库的 AI 原生 Git 工作台。</strong></p>
  <p>把 Git 状态、上下文预算、记忆、原始模型输出、思考文本和执行结果集中展示在一个可审查的 TUI 里。</p>
</div>

## 这是什么

`gitdex` 不是把聊天框包一层 Git 外壳，而是把完整的 Git 决策链路直接摊开给你看：

- 仓库状态始终可见
- LLM 上下文分区和预算可见
- 记忆是可持久化、可查看、可追踪的
- Raw 输出、清洗后输出、Thinking 和执行结果都可查看
- 纯查看 advisory 和真实会执行的命令被明确区分
- 首次启动就提供语言选择，运行中也能按 `L` 切换

## 快速入口

| 你想做什么 | 去哪里 |
| --- | --- |
| 从零部署并运行 | [GETTING_STARTED_zh.md](GETTING_STARTED_zh.md) |
| 看一次完整操作演示 | [OPERATION_DEMO_zh.md](OPERATION_DEMO_zh.md) |
| 查看部署设计 | [DEPLOYMENT_zh.md](DEPLOYMENT_zh.md) |
| 按步骤发布到 GitHub | [PUBLISHING_TO_GITHUB_zh.md](PUBLISHING_TO_GITHUB_zh.md) |
| 阅读英文总览 | [../README.md](../README.md) |

## 当前产品面

| 模块 | 现在能看到什么 |
| --- | --- |
| Git 状态 | 工作区、暂存区、分支、上游、远程、stash、部分平台状态 |
| AI 链路 | Context budget、Prompt 分区、知识命中、Raw 输出、清洗结果、Verifier 反馈 |
| Workflow | 当前阶段、当前目标、待处理建议、最近事件 |
| Memory | 记忆文件路径、更新时间、偏好、仓库模式、已解决目标、会话历史 |
| Interaction | advisory、命令、文件写入、补参数四类建议 |
| Runtime | 首启语言选择、模型选择、运行时按 `L` 重选语言 |

## 快速开始

依赖：

- Git
- Go
- 一个 AI 提供方：
- Ollama，本地至少一个模型，例如 `qwen2.5:3b`
- 或 OpenAI / DeepSeek API Key 与模型配置

直接运行源码：

```powershell
go run .\cmd\gitdex
```

Windows 构建并运行：

```powershell
.\build.ps1 -Target test
.\build.ps1 -Target build
.\bin\gitdex.exe
```

macOS / Linux：

```bash
make test
make build
./bin/gitdex
```

## 建议的首次使用顺序

1. 在真实 Git 仓库里启动 `gitdex`
2. 首次进入先选择界面语言
3. 如果你使用本地 provider，并且当前配置模型不可用，就进入模型选择界面
4. 按 `o` 或 `O` 浏览 `Workflow`、`Timeline`、`Context`、`Memory`、`Raw`、`Result`、`Thinking`
5. 按 `[` 和 `]` 切换滚动焦点
6. 用滚轮或 `up/down/pgup/pgdn` 查看当前区域完整内容
7. 按 `g` 设目标，或按 `f` 选工作流
8. 按 `y` 接受一条建议，确认 `Timeline` 和 `Result` 都有反馈

## 常用快捷键

- `y`：接受当前建议
- `n`：跳过当前建议
- `w`：查看或收起解释
- `z`：切换 focus/full AI 模式
- `r`：刷新并重新分析
- `l`：展开或收起操作日志
- `g`：设置当前目标
- `f`：选择工作流目标
- `L`：重新打开语言设置
- `o` / `O`：切换 observability 面板
- `[` / `]`：切换滚动焦点
- `up` / `down` / `PgUp` / `PgDn`：滚动当前聚焦区域
- `t`：切换 thinking 面板
- `Tab` / `Shift+Tab`：切换建议
- `q`：退出

## 配置位置

当前主配置入口：

- 项目级：`.gitdexrc`
- 全局：Linux/macOS 为 `~/.config/gitdex/config.yaml`
- 全局：Windows 为 `%AppData%\gitdex\config.yaml`
- 环境变量：`GITDEX_*`

为了平滑迁移，当前仍兼容读取：

- `.gitmanualrc`
- 旧的 `gitmanual` 全局配置目录
- `GITMANUAL_*`
- 家目录下旧的 `.gitmanual/` 记忆文件

项目级配置示例见 [../configs/example.gitdexrc](../configs/example.gitdexrc)。
