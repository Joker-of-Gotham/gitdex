# 从零开始部署与使用

这份文档假设你的机器上还没有 `gitdex` 所需环境。

## 1. 先安装基础环境

优先使用官方安装文档：

- Git: https://git-scm.com/book/en/v2/Getting-Started-Installing-Git
- Go: https://go.dev/doc/install
- Ollama: https://docs.ollama.com/quickstart
- Ollama for Windows: https://docs.ollama.com/windows
- Ollama for Linux: https://docs.ollama.com/linux
- GitHub SSH 配置: https://docs.github.com/en/authentication/connecting-to-github-with-ssh

安装完成后重新打开终端，确认命令可用：

```powershell
git --version
go version
ollama --version
```

只要有任何一个命令跑不通，就先不要继续。

## 2. 启动 Ollama 并拉模型

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

## 3. 获取源码

```powershell
git clone git@github.com:Joker-of-Gotham/gitdex.git
cd gitdex
```

如果你准备 fork 到自己的账号下，后面再用 `scripts/` 里的模块路径替换脚本统一改名即可。

## 4. 先跑测试

Windows：

```powershell
.\build.ps1 -Target test
```

macOS / Linux：

```bash
make test
```

这一步能最快判断本地环境是否可用。

## 5. 构建 gitdex

Windows：

```powershell
.\build.ps1 -Target build
```

macOS / Linux：

```bash
make build
```

## 6. 启动 gitdex

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

## 7. 首次启动时你会看到什么

1. 在真实 Git 仓库里启动 `gitdex`
2. 首次进入先看到语言选择界面
3. 选择主模型，再按需选择 verifier 模型
4. 等待第一轮 AI 分析完成
5. 按 `o` 或 `O` 查看 `Workflow`、`Timeline`、`Context`、`Memory`、`Raw`、`Result`、`Thinking`
6. 按 `g` 设目标，或按 `f` 选工作流

## 8. 最小验证清单

至少确认这些点：

- 程序启动后没有乱码或损坏图标
- 在全新配置目录下，首启语言选择会出现
- 选完语言后界面会立即切换
- 本地有 Ollama 模型时，模型选择界面会正常出现
- 对一条仅查看 advisory 按 `y` 后，会被标记为已查看，不会陷入重复刷新
- 在主界面按 `L` 能重新打开语言设置
- 执行或查看一条建议后，`Timeline` 和 `Result` 会出现反馈

## 9. 常见问题

如果 `gitdex` 打开了，但 AI 没有启用：

- 确认 Ollama 正在运行
- 确认本地确实有模型：

```powershell
ollama list
```

如果检测不到 Git：

- 确认 `git` 在 `PATH` 里
- 安装完成后重新打开终端

如果你之前已经运行过一次，首启语言选择没有出现：

- 这是正常现象，因为配置目录已经存在
- 直接在主界面按 `L`，就可以重新打开语言设置

如果你准备发布到 GitHub：

- 看 [DEPLOYMENT_zh.md](DEPLOYMENT_zh.md)
- 看 [PUBLISHING_TO_GITHUB_zh.md](PUBLISHING_TO_GITHUB_zh.md)
