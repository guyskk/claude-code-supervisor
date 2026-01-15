// Package cli 提供 patch 命令的实现
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findClaudePath 查找 claude 可执行文件路径
// 使用 exec.LookPath 在 PATH 中查找
// 返回完整路径和 error（未找到时）
func findClaudePath() (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH")
	}
	return claudePath, nil
}

// checkAlreadyPatched 检查是否已经 patch 过
// 使用 exec.LookPath("ccc-claude") 检测
// 返回：是否已 patch、ccc-claude 路径、error
func checkAlreadyPatched() (bool, string, error) {
	cccClaudePath, err := exec.LookPath("ccc-claude")
	if err != nil {
		// ccc-claude 不存在，说明未 patch
		return false, "", nil
	}
	return true, cccClaudePath, nil
}

// createWrapperScript 创建包装脚本
// 脚本内容：
//
//	#!/bin/sh
//	export CCC_CLAUDE=<cccClaudePath>
//	exec ccc "$@"
//
// 返回 error 表示创建失败
func createWrapperScript(claudePath, cccClaudePath string) error {
	// 生成脚本内容
	scriptContent := fmt.Sprintf(`#!/bin/sh
export CCC_CLAUDE=%s
exec ccc "$@"
`, cccClaudePath)

	// 写入脚本文件
	if err := os.WriteFile(claudePath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}

	return nil
}

// rollbackPatch 回滚 patch 操作
// 在 patch 失败时调用，将 ccc-claude 改回 claude
// 返回 error 表示回滚失败
func rollbackPatch(claudePath, cccClaudePath string) error {
	if err := os.Rename(cccClaudePath, claudePath); err != nil {
		return fmt.Errorf("failed to rollback patch: %w", err)
	}
	return nil
}

// applyPatch 执行 patch 操作
// 步骤：
// 1. 重命名 claude → ccc-claude
// 2. 创建包装脚本
// 3. 如果步骤 2 失败，回滚步骤 1
// 返回 error 表示操作失败
func applyPatch(claudePath string) error {
	// 生成 ccc-claude 路径
	// 注意：必须与 checkAlreadyPatched 和 runReset 中的文件名一致
	// 使用 ccc-claude（不是 .real 后缀）
	dir := filepath.Dir(claudePath)
	cccClaudePath := filepath.Join(dir, "ccc-claude")

	// 步骤 1: 重命名 claude → ccc-claude
	if err := os.Rename(claudePath, cccClaudePath); err != nil {
		return fmt.Errorf("failed to rename claude: %w", err)
	}

	// 步骤 2: 创建包装脚本
	if err := createWrapperScript(claudePath, cccClaudePath); err != nil {
		// 步骤 3: 回滚操作
		_ = rollbackPatch(claudePath, cccClaudePath)
		return fmt.Errorf("failed to create wrapper script (rolled back): %w", err)
	}

	return nil
}

// resetPatch 执行 reset 操作
// 步骤：
// 1. 找到 ccc-claude 路径
// 2. 重命名 ccc-claude → claude（覆盖包装脚本）
// 返回 error 表示操作失败
func resetPatch(cccClaudePath string) error {
	// 从 ccc-claude 路径推导出 claude 路径
	// ccc-claude 的路径是 /path/to/ccc-claude
	// claude 的路径是 /path/to/claude
	dir := filepath.Dir(cccClaudePath)
	claudePath := filepath.Join(dir, "claude")

	// 重命名 ccc-claude → claude
	if err := os.Rename(cccClaudePath, claudePath); err != nil {
		return fmt.Errorf("failed to restore claude: %w", err)
	}

	return nil
}

// RunPatch 执行 patch 命令
// opts.Reset 为 false 时执行 patch，为 true 时执行 reset
// 返回 error 表示操作失败，nil 表示成功
func RunPatch(opts *PatchCommandOptions) error {
	if opts.Reset {
		return runReset()
	}
	return runPatch()
}

// runPatch 执行 patch 操作
func runPatch() error {
	// 检查是否已经 patch 过
	patched, _, err := checkAlreadyPatched()
	if err != nil {
		return err
	}
	if patched {
		fmt.Println("Already patched")
		return nil
	}

	// 查找 claude 路径
	claudePath, err := findClaudePath()
	if err != nil {
		return err
	}

	// 执行 patch
	if err := applyPatch(claudePath); err != nil {
		return err
	}

	fmt.Println("Patched successfully")
	fmt.Println("Claude command now uses ccc")
	fmt.Println("Run 'claude' to start ccc, or 'sudo ccc patch --reset' to undo")
	return nil
}

// runReset 执行 reset 操作
func runReset() error {
	// 查找 ccc-claude 路径
	cccClaudePath, err := exec.LookPath("ccc-claude")
	if err != nil {
		fmt.Println("Not patched")
		return nil
	}

	// 执行 reset
	if err := resetPatch(cccClaudePath); err != nil {
		return err
	}

	fmt.Println("Reset successfully")
	fmt.Println("Claude command restored to original")
	return nil
}
