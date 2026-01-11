<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

## 项目背景

本项目使用 OpenSpec 管理开发流程，你要根据具体任务情况，用相应的 Skills，例如 openspec:proposal 规划需求和任务，openspec:apply 执行任务，完成后使用 openspec:archive 归档。

具体内容见 @README.md （必读），@/openspec/AGENTS.md (必读) 和 @openspec/project.md （必读）。

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
