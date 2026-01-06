# Implementation Tasks

## 1. 启动提示信息

- [x] 1.1 在 `cli.go` 的 `runClaude` 函数中，当 `CCC_SUPERVISOR=1` 时输出 log 文件路径
- [x] 1.2 计算并显示 state 目录的完整路径
- [x] 1.3 显示 hook 调用日志和 supervisor 输出日志的路径

## 2. 增强 Hook 日志输出

- [x] 2.1 在 `hook.go` 的 `RunSupervisorHook` 函数中增强 stderr 输出
- [x] 2.2 添加清晰的分节符（如 `==========`）区分不同阶段
- [x] 2.3 在调用 Supervisor 前输出 "正在审查..."
- [x] 2.4 在收到结果后输出审查结果的摘要

## 3. 日志文件内容改进

- [x] 3.1 改进 `hook-invocation.log` 的格式，使其更易读
- [x] 3.2 在 supervisor 输出日志中添加时间戳和分节符
- [x] 3.3 确保 Supervisor 的完整思考过程被记录

## 4. 测试验证

- [x] 4.1 手动测试启动提示信息是否正确显示
- [x] 4.2 手动测试 hook 日志在 verbose 模式（ctrl+o）下是否可见
- [x] 4.3 验证 log 文件内容是否完整且易读
