# GitDex V3 回归门禁

高频事故回归项（必须长期保绿）：

1. whitespace 管线
- 文件写入必须移除行尾空白并保证单个结尾换行。
- `git add` / `git commit` 前置清洗必须生效。

2. GitHub 404/422 诊断
- `github_op` 失败必须输出 `[GITDEX DIAGNOSIS]` 指引。
- 对不可重试错误必须建议 skip/manual，而不是盲目重试。

3. provider unavailable
- 主 provider 不可用时必须给出明确诊断码。
- 可用 secondary 存在时允许受控降级，不得黑盒退化。

CI 中通过 `regression` 任务在三平台执行上述回归测试。
