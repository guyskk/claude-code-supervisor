# Design: use-syscall-exec

## Technical Approach

### 条件编译策略

使用 Go 的 build tags 实现平台特定代码：

```
internal/cli/
├── cli.go              # 主逻辑，调用 executeProcess()
├── exec_unix.go        # //go:build unix
└── exec_windows.go     # //go:build windows
```

### Unix 实现 (exec_unix.go)

```go
//go:build unix

package cli

import (
    "os"
    "os/exec"
    "syscall"
)

func executeProcess(claudePath string, args []string, env []string) error {
    return syscall.Exec(claudePath, args, env)
}
```

关键点：
- `syscall.Exec` 不返回（除非出错）
- 当前进程直接被 claude 替换
- 需要先用 `exec.LookPath` 获取完整路径

### Windows 实现 (exec_windows.go)

```go
//go:build windows

package cli

import (
    "os"
    "os/exec"
)

func executeProcess(claudePath string, args []string, env []string) error {
    cmd := exec.Command(claudePath, args[1:]...)
    cmd.Env = env
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

Windows 保持现有行为，因为 Windows 没有 fork/exec 语义。

### runClaude 函数重构

```go
func runClaude(cfg *config.Config, providerName string, settings map[string]interface{}, cmdArgs []string) error {
    // 查找 claude 可执行文件路径
    claudePath, err := exec.LookPath("claude")
    if err != nil {
        return fmt.Errorf("claude not found in PATH: %w", err)
    }

    // 构建参数 (argv[0] 必须是程序名)
    settingsPath := config.GetSettingsPath(providerName)
    args := []string{"claude", "--settings", settingsPath}
    if len(cfg.ClaudeArgs) > 0 {
        args = append(args, cfg.ClaudeArgs...)
    }
    args = append(args, cmdArgs...)

    // 构建环境变量
    authToken := provider.GetAuthToken(settings)
    env := append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

    // 执行（Unix 上不返回，Windows 上等待子进程）
    return executeProcess(claudePath, args, env)
}
```

## Trade-offs

### 优点
1. 进程树更扁平，资源占用更少
2. 信号处理更直接（Ctrl+C 直接发给 claude）
3. 退出码直接继承，无需解析错误

### 缺点
1. 代码复杂度略增（需要维护两个平台文件）
2. Unix 上 `Launching with provider` 消息后无法执行更多代码

### 决策
优点明显大于缺点，且符合 ccc 作为"配置切换器"的设计理念。

## Testing Considerations

1. **单元测试**：无法直接测试 `syscall.Exec`（进程会被替换）
2. **集成测试**：现有集成测试可继续验证整体行为
3. **Mock 策略**：可以通过 `var execProcessFunc = executeProcess` 模式支持测试 mock
