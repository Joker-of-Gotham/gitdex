# GitDex V3 参考项目借鉴规则集

本文件将四个参考项目的经验固化为“可迁移 / 不可迁移 / 替代方案”。

## 1) gh-dash

### 可迁移
- Section 化 UI 结构（Shell + Sections + Footer + Input）。
- 信息密度与视觉层级设计（状态色、卡片边界、主次文本）。
- 快速键位发现与帮助提示模式。

### 不可直接迁移
- 与 GitDex 目标不一致的 GitHub-only 业务入口。
- 绑定 gh-dash 内部领域模型的数据结构。

### 替代方案
- 保留视觉交互范式，替换为 GitDex 的 Suggestions/Git/Goals/Log/Config 五区。

## 2) lazygit

### 可迁移
- context-controller 分层模式。
- 配置迁移与校验路径（版本迁移 + 默认兜底 + 用户可解释）。
- 命令执行的预检与失败恢复思路。

### 不可直接迁移
- 直接复用 lazygit 的命令实现细节与内部业务对象。

### 替代方案
- 采用“runtime + adapter + diagnostics”的 GitDex 执行内核。

## 3) diffnav

### 可迁移
- 焦点分区与滚动区域控制（键盘 + 鼠标统一语义）。
- 双区域信息对照的布局方法。

### 不可直接迁移
- 面向 diff-only 的固定数据模型。

### 替代方案
- 将区域滚动能力抽象为通用 ScrollEngine，服务 Git/Goals/Log 等区域。

## 4) octo.nvim

### 可迁移
- object-action 命令语义。
- polling 与状态同步策略。
- GitHub 工作流任务抽象方式。

### 不可直接迁移
- nvim 运行时依赖与 Lua 生态绑定部分。

### 替代方案
- 在 GitDex 中定义 object-action 语法层，再映射到 slash 命令和 adapter 能力。

## 统一借鉴禁忌

- 禁止直接复制参考项目内部对象模型到 GitDex 核心域。
- 禁止把平台差异掩盖在 UI 层；平台能力边界必须在 adapter 层显式声明。
- 禁止以“增加上下文量”替代“上下文质量控制”。

## 统一落地要求

- 每项借鉴必须映射到一个明确模块与测试条目。
- 每项“不可迁移”都必须给出替代设计，不允许留空。
- 每项设计必须满足三平台行为一致性与可回滚性。
