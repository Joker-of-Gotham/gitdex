# PERFORMANCE BASELINES

本文件定义 GitDex V3 的首批性能基准与预算门槛，用于回归检测。

## 基准命令

```powershell
go test ./internal/executor -run=^$ -bench="." -benchmem -count=1
go test ./internal/llm/budget -run=^$ -bench="." -benchmem -count=1
```

## 当前基线（Windows amd64）

- `BenchmarkParseCommand`: `951.2 ns/op`, `824 B/op`, `15 allocs/op`
- `BenchmarkStripTrailingWhitespace`: `294.4 ns/op`, `144 B/op`, `4 allocs/op`
- `BenchmarkEstimateTokens`: `58.95 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkCompressGitContent`: `165.3 ns/op`, `0 B/op`, `0 allocs/op`

## 预算门槛（P0 回归告警）

- `BenchmarkParseCommand` > `2.0 us/op` 或 > `30 allocs/op`
- `BenchmarkStripTrailingWhitespace` > `0.8 us/op`
- `BenchmarkEstimateTokens` > `120 ns/op`
- `BenchmarkCompressGitContent` > `300 ns/op`

## 维护规则

- 每次性能相关变更后更新本文件中的“当前基线”。
- 若超过预算门槛，必须在 PR 中附带原因分析和优化/豁免说明。
- CI 后续可将上述门槛接入自动化校验脚本。

