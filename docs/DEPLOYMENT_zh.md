# 部署与发布设计

这份文档解释 `gitdex` 是如何构建、校验并发布到 GitHub 的。

## 发布模型

`gitdex` 是 CLI/TUI 产品，主要部署目标是 GitHub Releases，而不是服务器运行时。

每次推送 `v*` tag，都会自动发布：

- `gitdex-windows-amd64.exe`
- `gitdex-windows-arm64.exe`
- `gitdex-linux-amd64`
- `gitdex-linux-arm64`
- `gitdex-macos-amd64`
- `gitdex-macos-arm64`
- `gitdex-source.zip`
- `checksums.txt`

## GitHub Actions 布局

### CI

文件：`.github/workflows/ci.yml`

检查项：

- `go vet ./...`
- `go test ./...`
- `go build ./...`
- `golangci-lint`
- `go test -race ./...`

### Release

文件：`.github/workflows/release.yml`

流程：

1. 在 `v*` tag 推送时触发
2. 执行 `go vet`
3. 执行 `go test -race ./...`
4. 执行 `./scripts/build.sh <tag> dist`
5. 把发布资产上传到 GitHub Releases

### CodeQL

文件：`.github/workflows/codeql.yml`

作用：

- 在 push、pull request 和每周定时任务中做基础安全扫描

## 本地构建脚本

`scripts/build.sh` 是统一的发布资产构建脚本。它会生成和 GitHub Release 完全一致的文件名，并额外生成：

- `dist/release-notes.md`
- `dist/checksums.txt`

本地示例：

```bash
./scripts/build.sh v1.0.0 dist
```

## Release Notes 生成方式

`scripts/render-release-notes.sh` 负责生成 GitHub Actions 用的 release 正文，避免手写文案和实际附件列表不一致。

## 仓库元信息建议

建议在 GitHub 仓库设置中补齐：

- description：
  `AI-native Git workflow for local repositories with visible context, memory, raw output, and execution flow.`
- website：
  `https://github.com/Joker-of-Gotham/gitdex#readme`
- topics：
  `git`、`tui`、`ollama`、`local-first`、`observability`、`developer-tools`、`terminal-ui`、`ai-workflow`

这些内容需要在 GitHub 网页端设置。

## 社区健康文件

仓库现在应包含：

- `LICENSE`
- `CODE_OF_CONDUCT.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- issue templates
- pull request template

这样 GitHub 侧栏和仓库信息面会更完整，更符合公开产品仓库的呈现方式。

## 发布后检查

推送 tag 之后，按顺序确认：

1. `Release` workflow 成功
2. 打开 GitHub release 页面
3. 六个平台二进制都存在
4. `gitdex-source.zip` 存在
5. `checksums.txt` 存在
6. release 正文和当前产品能力一致
