## 项目背景

具体内容见 @README.md（必读），@docs/project.md（必读）

## 开发流程

本项目使用 SpecKit 管理开发流程，你要根据具体需求，用相应的 Skills 按流程完成开发任务。

1. Create the spec
Use the /speckit.specify command to describe what you want to build. Focus on the what and why, not the tech stack.

2. Create a technical implementation plan
Use the /speckit.plan command to provide your tech stack and architecture choices.

3. Break down into tasks
Use /speckit.tasks to create an actionable task list from your implementation plan.

4. Execute implementation
Use /speckit.implement to execute all tasks and build your feature according to the plan.

For detailed step-by-step instructions, see our comprehensive guide @docs/spec-driven.md (必读).

**重要说明‼️ ：具体内容 @docs/spec-driven.md 非常重要！请必须仔细阅读并遵循这个指南。**

### 特别说明：你要使用中文

1. 我的母语是中文，你要用中文和我沟通，这样更方便。
2. SpecKit开发流程创建的所有文档，全部都用中文编写。

## Write idiomatic Go code

Write idiomatic Go code with goroutines, channels, and interfaces. Optimizes concurrency, implements Go patterns, and ensures proper error handling. Use PROACTIVELY for Go refactoring, concurrency issues, or performance optimization.

When write code:
1. Analyze requirements and design idiomatic Go solutions
2. Implement concurrency patterns using goroutines, channels, and select
3. Create clear interfaces and struct composition patterns
4. Establish comprehensive error handling with custom error types
5. Set up testing framework with table-driven tests and benchmarks
6. Optimize performance using pprof profiling and measurements

Process:
- Prioritize simplicity first - clear is better than clever
- Apply composition over inheritance through well-designed interfaces
- Implement explicit error handling with no hidden magic
- Design concurrent systems that are safe by default
- Benchmark thoroughly before optimizing performance
- Prefer standard library solutions over external dependencies
- Follow effective Go guidelines and community best practices
- Organize code with proper module management and clear package structure

Provide:
-  Idiomatic Go code following effective Go guidelines and conventions
-  Concurrent code with proper synchronization and race condition prevention
-  Table-driven tests with subtests for comprehensive coverage
-  Benchmark functions for performance-critical code paths
-  Error handling with wrapped errors, context, and custom error types
-  Clear interfaces and struct composition patterns
-  go.mod setup with minimal, well-justified dependencies
-  Performance profiling setup and optimization recommendations

## 提交前检查

```bash
Usage: ./check.sh [OPTIONS]

Options:
  -l, --lint          Run lint checks (gofmt, go vet, shellcheck, markdownlint)
  -t, --test          Run tests with race detector
  -b, --build         Run build validation
  -h, --help          Show this help message

If no options specified, runs all checks (lint, test, build).

Examples:
  ./check.sh                          # Run all checks
  ./check.sh --lint                   # Run lint only
```
