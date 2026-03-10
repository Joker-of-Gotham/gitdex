# GitHub 发布检查清单

这份文档就是把 `gitdex` 发布到公开 GitHub 仓库前，最后该做的事情全部按顺序列出来。

目标仓库：

```text
git@github.com:Joker-of-Gotham/gitdex.git
```

目标版本：

```text
v1.0.0
```

## 1. 对齐模块路径

当前仓库应对齐到：

```text
github.com/Joker-of-Gotham/gitdex
```

如果以后 fork 到别的账号，用下面脚本整体改路径：

PowerShell：

```powershell
.\scripts\set-module-path.ps1 -ModulePath github.com/<你的用户或组织>/gitdex
```

Bash：

```bash
./scripts/set-module-path.sh github.com/<你的用户或组织>/gitdex
```

脚本会一起更新：

- `go.mod`
- `cmd/`、`internal/`、`test/` 下的 Go import
- docs、GitHub workflow、辅助脚本

## 2. 清理本地生成物

发布前不要把这些本地或生成目录带进历史：

- `bin/`
- `dist/`
- `.gitdex/`
- `.cursor/`
- `.opencode/`

这些内容已经被忽略，不应该被提交。

## 3. 跑发布前质量门

最低要求：

```powershell
go vet ./...
go test ./...
go build ./...
```

建议再跑：

```powershell
go test -race ./...
```

Windows 也可以直接用：

```powershell
.\build.ps1 -Target test
.\build.ps1 -Target build
```

## 4. 构建发布资产

本地一次性生成完整发布资产：

```bash
./scripts/build.sh v1.0.0 dist
```

如果你在 Windows 上不想依赖 Bash，也可以直接：

```powershell
.\build.ps1 -Target assets
```

预期文件：

- `dist/gitdex-windows-amd64.exe`
- `dist/gitdex-windows-arm64.exe`
- `dist/gitdex-linux-amd64`
- `dist/gitdex-linux-arm64`
- `dist/gitdex-macos-amd64`
- `dist/gitdex-macos-arm64`
- `dist/gitdex-source.zip`
- `dist/checksums.txt`

GitHub Actions 在 tag 推送后也会自动生成同样的一组文件。

## 5. 检查 README 展示面

发布前确认：

- `README.md` 顶部 hero 没有裁边或文字溢出
- 顶部 badge 行、快捷入口行能正常渲染
- 仓库导航链接可用
- 三张展示图都存在：
  - `docs/assets/readme-hero.svg`
  - `docs/assets/readme-observability.svg`
  - `docs/assets/readme-advisory.svg`

## 6. 设置 GitHub 仓库信息

建议仓库描述：

```text
AI-native Git workflow for local repositories with visible context, memory, raw output, and execution flow.
```

建议 website：

```text
https://github.com/Joker-of-Gotham/gitdex#readme
```

建议 topics：

- `git`
- `tui`
- `ollama`
- `local-first`
- `observability`
- `developer-tools`
- `terminal-ui`
- `ai-workflow`

这些需要在 GitHub 网页端仓库设置里补上。

## 7. 分批推送仓库

建议按下面顺序提交：

1. 仓库骨架和忽略规则
2. 核心源码
3. 文档、展示资源、GitHub 社区文件

然后：

```bash
git push -u origin main
```

## 8. 创建并推送发布 tag

```bash
git tag v1.0.0
git push origin v1.0.0
```

之后 `Release` workflow 会自动：

- 执行 `go vet`
- 执行 `go test -race ./...`
- 构建六个平台二进制
- 生成 `gitdex-source.zip`
- 生成 `checksums.txt`
- 创建 GitHub Release 并上传附件

## 9. Release 标题和正文

建议标题：

```text
gitdex v1.0.0
```

参考内容：

- [../.github/RELEASE_TEMPLATE.md](../.github/RELEASE_TEMPLATE.md)
- `./scripts/build.sh` 生成的 `dist/release-notes.md`

## 10. 最后一遍公开检查

- `README.md` 已经足够完整且渲染正常
- `docs/README_zh.md`、`docs/GETTING_STARTED_zh.md`、`docs/OPERATION_DEMO_zh.md` 不再乱码
- `CODE_OF_CONDUCT.md`、`CONTRIBUTING.md`、`SECURITY.md` 已存在
- `.github/workflows/ci.yml`、`.github/workflows/release.yml`、`.github/workflows/codeql.yml` 已接通
- 模块路径已经是 `github.com/Joker-of-Gotham/gitdex`
- 对旧用户的兼容说明仍保留了 `.gitmanualrc` 和 `GITMANUAL_*`
