# GitDex 配置 Schema（V3）

GitDex V3 提供 `.gitdexrc` 的 JSON Schema：

- `configs/schema.gitdexrc.json`

## IDE 使用方式

在 `.gitdexrc` 顶部加入：

```yaml
# yaml-language-server: $schema=./configs/schema.gitdexrc.json
```

即可启用字段补全与基础校验。

## 配置诊断命令

支持以下命令：

- `gitdex config lint`：校验配置合法性并输出安全告警。
- `gitdex config explain`：输出最终生效配置来源与迁移信息。
- `gitdex config source`：输出 source-trace JSON。
- `gitdex config schema`：输出 schema 文件路径。

## 安全策略

- 推荐使用 `api_key_env`，不推荐明文 `api_key`。
- 若 `api_key_env` 被误填为字面 key（如 `sk-...`），系统会兼容运行并给出迁移告警。

## 适配器可配置二进制

- `adapters.git.binary`：配置 Git 可执行文件路径（默认 `git`）。
- `adapters.github.gh.binary`：配置 GitHub CLI 可执行文件路径（默认 `gh`）。

这两个字段用于避免平台相关硬编码，提升跨系统鲁棒性。
