# 从零开始部署与使用

这份文档假设你的机器上还没有运行 `gitdex` 所需的环境。

## 1. 先安装基础环境

优先使用官方安装文档：

- Git: https://git-scm.com/book/en/v2/Getting-Started-Installing-Git
- Go: https://go.dev/doc/install
- Ollama: https://docs.ollama.com/quickstart
- OpenAI API: https://platform.openai.com/docs/quickstart
- DeepSeek API: https://api-docs.deepseek.com/
- GitHub SSH 配置: https://docs.github.com/en/authentication/connecting-to-github-with-ssh

安装完成后重新打开终端，确认命令可用：

```powershell
git --version
go version
```

如果你计划使用 Ollama，再额外确认：

```powershell
ollama --version
```

只要有任何一个必需命令跑不通，就先不要继续。

## 2. 选择 AI 提供方式

`gitdex` 目前支持：

- 纯 Ollama
- 纯 OpenAI
- 纯 DeepSeek
- 混用，例如 OpenAI 主模型 + Ollama verifier

### 方案 A：Ollama

至少准备一个本地模型：

```powershell
ollama pull qwen2.5:3b
```

如果你希望启用 verifier，再准备一个更大的模型：

```powershell
ollama pull qwen2.5:7b
```

确认模型存在：

```powershell
ollama list
```

### 方案 B：OpenAI

设置 API Key：

```powershell
$env:OPENAI_API_KEY="your_key_here"
```

### 方案 C：DeepSeek

设置 API Key：

```powershell
$env:DEEPSEEK_API_KEY="your_key_here"
```

## 3. 获取源码

```powershell
git clone git@github.com:Joker-of-Gotham/gitdex.git
cd gitdex
```

如果你后续要 fork 到自己的仓库，再用 `scripts/` 里的模块路径替换脚本统一改名即可。

## 4. 配置 `.gitdexrc`

最小 OpenAI 配置：

```yaml
llm:
  provider: "openai"
  endpoint: "https://api.openai.com/v1"
  api_key_env: "OPENAI_API_KEY"
  primary:
    provider: "openai"
    model: "gpt-4.1-mini"
    enabled: true
```

最小 DeepSeek 配置：

```yaml
llm:
  provider: "deepseek"
  endpoint: "https://api.deepseek.com"
  api_key_env: "DEEPSEEK_API_KEY"
  primary:
    provider: "deepseek"
    model: "deepseek-chat"
    enabled: true
```

混用示例：OpenAI 主模型 + Ollama verifier

```yaml
llm:
  primary:
    provider: "openai"
    model: "gpt-4.1-mini"
    endpoint: "https://api.openai.com/v1"
    api_key_env: "OPENAI_API_KEY"
    enabled: true
  secondary:
    provider: "ollama"
    model: "qwen2.5:7b"
    endpoint: "http://localhost:11434"
    enabled: true
```

完整示例看 [configs/example.gitdexrc](../configs/example.gitdexrc)。

## 5. 先跑测试

Windows：

```powershell
.\build.ps1 -Target test
```

macOS / Linux：

```bash
make test
```

这一步能最快判断本地环境是否可用。

## 6. 构建 gitdex

Windows：

```powershell
.\build.ps1 -Target build
```

macOS / Linux：

```bash
make build
```

## 7. 启动 gitdex

直接运行源码：

```powershell
go run .\cmd\gitdex
```

运行构建产物：

```powershell
.\bin\gitdex.exe
```

macOS / Linux：

```bash
go run ./cmd/gitdex
./bin/gitdex
```

## 8. 首次启动时你会看到什么

1. 在真实 Git 仓库里启动 `gitdex`
2. 首次进入先看到语言选择界面
3. 如果使用本地 provider，并且当前配置模型不可用，就会出现模型选择界面
4. 等待第一轮 AI 分析完成
5. 按 `o` 或 `O` 查看 `Workflow`、`Timeline`、`Context`、`Memory`、`Raw`、`Result`、`Thinking`
6. 按 `[` 和 `]` 切换滚动焦点
7. 用鼠标滚轮或 `up/down/pgup/pgdn` 查看当前区域全部内容
8. 按 `g` 设目标，或按 `f` 选工作流

## 9. 最小验证清单

至少确认这些点：

- 程序启动后没有乱码或损坏图标
- 在全新配置目录下，首启语言选择会出现
- 选完语言后界面会立刻切换
- 仅查看 advisory 按 `y` 后，会被标记为已查看，不会陷入重复刷新
- `Thinking` 面板在 provider 暴露 reasoning 时能看到内容
- 左侧主列、右侧 Git Areas、右侧 Observability 都能用滚轮和键盘滚动
- 执行或查看一条建议后，`Timeline` 和 `Result` 会出现反馈

## 10. 常见问题

如果 `gitdex` 打开了，但 AI 没有启用：

- 确认 provider 凭据或本地运行时真的可用
- 对 Ollama，确认服务已启动并且模型存在：

```powershell
ollama list
```

- 对 OpenAI / DeepSeek，确认启动 `gitdex` 的那个 shell 里已经设置好 API Key

如果检测不到 Git：

- 确认 `git` 在 `PATH` 里
- 安装完成后重新打开终端

如果你之前已经运行过一次，首启语言选择没有出现：

- 这是正常现象，因为配置目录已经存在
- 直接在主界面按 `L` 就可以重新打开语言设置

如果你准备发布到 GitHub：

- 看 [DEPLOYMENT_zh.md](DEPLOYMENT_zh.md)
- 看 [PUBLISHING_TO_GITHUB_zh.md](PUBLISHING_TO_GITHUB_zh.md)
