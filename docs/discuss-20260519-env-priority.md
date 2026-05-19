# 讨论：env 优先级问题排查（PR #79）

日期：2026-05-19
话题：核实 ccc 切换 provider 时 env 的生效优先级，评估 PR #79 的方案是否安全。
触发：https://github.com/guyskk/claude-code-supervisor/pull/79

> 本文件 Append-only，只追加不覆盖。

---

## 第 1 轮：背景探索与关键发现

### 1.1 PR #79 是什么

- PR #79 标题：`Preserve user env settings during provider switch`
- 分支：`fix/preserve-user-env-in-settings`，head = `a251cbf`
- 状态：**OPEN，CONFLICTING / DIRTY**（与 main 冲突）
- 目标：解决"切换 provider 时把用户在 settings.json 里手动配置的 env 抹掉"的问题（Issue #74 类问题）。

PR #79 的代码改动（针对它的 base）：
1. `CleanEnvInSettings`：从「移除 `ANTHROPIC_*`/`CLAUDE_*` 前缀 + provider 同名 key」收窄为「只移除 provider 同名 key」，即**保留用户的 `ANTHROPIC_*`/`CLAUDE_*` 自定义 env**。
2. `provider.go`：删除 `delete(settingsWithHook, "env")`，即**不再清空 settings.json 的 env，让合并后的 env 留在 settings.json**。

### 1.2 关键发现：PR #79 已与 main 分叉，且 main 已用另一套方案修复

- `git merge-base main pr79` = `8f0bf0c`（PR #73 合并点）。
- PR #79 从 `8f0bf0c` 拉出，但 main 之后已经走了 **PR #76（commit `c71511a` "fix: preserve user-defined env in settings.json"）**，针对**同一个问题**做了**另一套修复**。
- 因此 PR #79 与 main 是**针对同一问题的两个独立方案**，PR #79 是陈旧的、与已合并的 #76 冲突的备选方案。
- PR #79 diff 里还在改 `CleanEnvInSettings`，但 main 上该函数**已被 #76 删除**——这就是冲突来源。

### 1.3 当前 main（已含 #76）的 env 处理逻辑（实际代码确认）

`internal/provider/provider.go` `SwitchWithHook()`（行 37-121）：
- 分别提取三个来源的 env：`userEnvMap`（settings.json）、`baseEnvMap`（ccc.json settings）、`providerEnvMap`（ccc.json provider）。
- `managedEnvKeys` = base env keys + provider env keys。
- `MergeWithPriority(base, provider, user)` 深合并（user > provider > base）。
- `EnsureStopHook` 注入 Supervisor Stop hook。
- **`delete(settingsWithHook, "env")`** 然后写回 `FilterUserEnvForSettings(userEnvMap, managedEnvKeys)`。
- 子进程 env = `MergeEnvMaps(baseEnvMap, providerEnvMap)`（**只有 base + provider，不含 user env**）。

`internal/config/config.go` `FilterUserEnvForSettings()`（行 256-276）：
- 写入 settings.json 的 env：跳过 `managedEnvKeys`，并且**跳过所有 `ANTHROPIC_*` / `CLAUDE_*` 前缀 key**。
- 结论：当前 main 下，**settings.json 的 env 永远不含 `ANTHROPIC_*`/`CLAUDE_*`**，因此 settings.json 与 provider env 不可能在这些 key 上冲突。

`internal/cli/exec.go` `runClaude()`（行 129-151）：
- `env := os.Environ()` 复制 ccc 进程环境。
- 过滤掉所有 `CLAUDE_*` / `ANTHROPIC_*` 前缀的进程 env（保证 provider 优先）。
- 追加 `switchResult.EnvVars`（= base + provider env）。
- `syscall.Exec(claudePath, args, env)` 用该环境替换为 claude 进程。

### 1.4 待验证的核心未知

官方文档 `docs/claude-code-settings.md` 说明了 settings 作用域优先级（managed > user > project），但**没有说明 settings.json 的 `env` 字段与「claude 进程实际继承的 OS 环境变量」谁优先**。

这是决定 PR #79 方案是否安全的关键问题：
- 若 **settings.json `env` 覆盖进程 env**：PR #79 保留用户旧的 `ANTHROPIC_BASE_URL` 在 settings.json 里，会**压过命令行传入的 provider env**，导致**切换 provider 失效**（用错 base url / token）。这是严重 bug。
- 若 **进程 env 覆盖 settings.json `env`**：PR #79 相对安全，provider env 仍生效，settings.json 里的旧 env 只在 provider 未定义该 key 时兜底。

**下一步：用 claude 实测这个优先级。**

---

## 第 2 轮：实测验证（claude 2.1.144，隔离环境 CLAUDE_CONFIG_DIR=./tmp/claude-home）

三个实验，settings.json 端口/值 vs 进程 env 端口/值，互为对照：

### 实验 A — 中性变量，经 Bash 工具观测
- settings.json: `CCC_PRECEDENCE_TEST=VALUE_FROM_SETTINGS_JSON`
- 进程 env: `CCC_PRECEDENCE_TEST=VALUE_FROM_PROCESS_ENV`
- `claude -p "printenv CCC_PRECEDENCE_TEST"` 输出：**`VALUE_FROM_SETTINGS_JSON`**
- → settings.json env 覆盖进程 env。

### 实验 B（决定性）— ANTHROPIC_BASE_URL，观测 claude 真实 API 连接目标
- settings.json: `ANTHROPIC_BASE_URL=http://127.0.0.1:19991`
- 进程 env: `ANTHROPIC_BASE_URL=http://127.0.0.1:19992`
- 双端口监听器结果：**`HIT port=19991 (SETTINGS_JSON)`**
- → 连 claude 自己发 API 用的 `ANTHROPIC_BASE_URL`，settings.json 也覆盖进程 env。

### 实验 C — settings.json 无 env（模拟当前 main 经 FilterUserEnvForSettings 剥离后的状态）
- settings.json: 无 env
- 进程 env: `ANTHROPIC_BASE_URL=http://127.0.0.1:19992`
- 结果：**`HIT port=19992 (PROCESS_ENV)`**
- → settings.json 不含该 key 时，进程 env（= ccc 传入的 provider env）才生效。

### 实测结论（确定）

**Claude Code 中：settings.json 的 `env` 严格覆盖 claude 进程继承的 OS 环境变量。**
ccc 用 `syscall.Exec` 传入的 provider env 属于"进程环境变量"层级，**优先级低于 settings.json 的 env**。

### 对三个问题的回答

1. **env 优先级**：`settings.json.env` > 进程 env（ccc 命令行传入的 provider env）。仅当 settings.json 不含该 key 时，进程 env 兜底生效。
2. **哪里的 env 生效**：settings.json 里有 `ANTHROPIC_BASE_URL` 就用它；没有才用 ccc 传入的 provider 值。
3. **启动时不清除 settings.json 的 env 是否有问题**：**是，严重问题**。因 settings.json env 压过进程 env，ccc 靠进程 env 传 provider 配置会被 settings.json 里残留的旧 `ANTHROPIC_*` 覆盖 → **切换 provider 静默失效**（用错 base_url / token / model）。

### 对 #76（当前 main）与 #79 的判断

- **当前 main（#76）方案正确**：`FilterUserEnvForSettings` 把所有 `ANTHROPIC_*`/`CLAUDE_*` 及 base/provider managed key 一律从 settings.json 剔除 → settings.json 永不含这些 key → provider env 经进程 env 必然生效（实验 C 证明）。同时保留用户非冲突、非 `ANTHROPIC_*`/`CLAUDE_*` 的自定义 env（实验思路同 A，那类 key 会被保留并生效，正是用户想要的）。
- **PR #79 方案不安全，不应合并**：
  - 它只移除"与当前 provider 同名的 key"，**保留用户的 `ANTHROPIC_*`/`CLAUDE_*`**；并删除 `delete(settingsWithHook,"env")`。
  - 后果：provider 未覆盖的 `ANTHROPIC_*`（如 `ANTHROPIC_MODEL`、`ANTHROPIC_SMALL_FAST_MODEL`）或切换 provider 后 settings.json 残留的旧值，会留在 settings.json 并**覆盖**命令行传入的 provider env → 切 provider 失效 / 模型错乱。
  - 附带：provider token 被持久化写入 settings.json 文件（安全 & 陈旧问题）。
  - PR #79 想解决的真实诉求（用户手动配的非 provider env 不该被抹掉）**已被 #76 满足**。
- **建议**：PR #79 关闭（其诉求已由 #76 覆盖），不需要再改代码；若发现 #76 仍有未覆盖边界，应基于 main 现状重新评估，而非合并 #79。

---

## 第 3 轮：用户决策与"迁移 + 提示"方案设计

### 用户决策

1. **PR #79 已关闭**（已执行 `gh pr close 79`，附实测说明评论）。
2. 用户认可"#76 会丢失用户手动配的 `ANTHROPIC_*`/`CLAUDE_*`"是真实问题，要求：把这些 key **迁移到 ccc.json**（不丢失），并且**打断用户给出提示**（不静默），更稳妥。

### #76 现状复核（确认丢失问题真实存在）

`FilterUserEnvForSettings`（config.go:256）对 `managedEnvKeys` 命中 或 `ANTHROPIC_*`/`CLAUDE_*` 前缀的 key 一律 `continue` → 既不写回 settings.json，也不进子进程 env（子进程 env = `MergeEnvMaps(base, provider)`，不含 user）。结论：用户手动配的全局 `CLAUDE_CODE_*` 等，跑一次 ccc 后**永久丢失且失效**。迁移方案方向正确。

### 现有提示/迁移机制（用于风格对齐）

- `internal/migration/migration.go`：`GetUserInputFunc`（stdin `bufio.Reader`，测试可覆盖）、`PromptUser()`（`[y/N]`）、`MigrateFromSettings()`。
- 触发点 `internal/cli/cli.go:280-297`：**仅在首次运行（ccc.json 不存在、`config.Load()` 报错）** 时提示迁移，把整个 settings.json 转成 ccc.json 的 `default` provider。
- 这与本需求不同：本需求是**每次运行**都要检测 settings.json 里是否残留 `ANTHROPIC_*`/`CLAUDE_*`（ccc.json 已存在的常规场景），需新增独立的检测+提示+迁移逻辑，挂在启动路径（`runClaude` / `SwitchWithHook` 之前）。

### 提示方案设计（待对齐）

**触发条件**：启动 claude 前，`LoadSettings()` 得到的 user settings.json `env` 中存在任何 `ANTHROPIC_*`/`CLAUDE_*` 或命中 `managedEnvKeys` 的 key。

**交互内容**：列出检测到的这些 key，说明 WHY（settings.json 的 env 会覆盖 ccc 经命令行传入的 provider env，导致切换 provider 失效，已实测证明），给出迁移计划，`[Y/n]` 确认。

**迁移目标分类**（仍建议区分，避免凭据串台）：
- 凭据/端点/模型类（黑名单：`ANTHROPIC_BASE_URL`/`ANTHROPIC_AUTH_TOKEN`/`ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL`/`ANTHROPIC_SMALL_FAST_MODEL`）：**不进 base**（进 base 会被所有 provider 继承，旧端点/模型串台/陈旧）。提示建议用户改用 provider 配置；ccc 这里仅从 settings.json 移除。
- 其余 `ANTHROPIC_*`/`CLAUDE_*` 行为开关类：迁入 `ccc.json.settings.env`（base 模板），既不丢失又继续生效，且 settings.json 仍被清空（实验 C 保证切换正确）。

**幂等**：迁移后 settings.json 不再含这些 key → 后续运行不再触发提示（除非用户又手动加回，此时再次提示，因稀少不扰民）。

### 必须与用户对齐的关键设计点（阻塞项）

1. **非交互场景兜底**：supervisor 模式会重入 ccc（`CCC_SUPERVISOR_ID` 已设）、`ccc -p`、CI、stdin 非 TTY —— **绝不能在这些路径上阻塞等待输入**。兜底策略二选一：
   - (a) 回退到 #76 行为（剥离丢弃，切换仍正确，但丢全局偏好）；
   - (b) 静默执行同样的"选择性迁移"+ stderr 一行提示（不丢失，推荐）。
2. **用户拒绝（选 n）时的行为**：(a) 中止 ccc 不启动 claude，让用户自行处理；(b) 仍按 #76 安全剥离后继续启动（保证切换正确，但本次仍丢偏好）。
3. **凭据黑名单**是否认可？`ANTHROPIC_MODEL`/`ANTHROPIC_SMALL_FAST_MODEL` 是否也算 provider 专属（不进 base）？
4. **base 已存在同名 key**时，迁移是否以 settings.json 的值覆盖 base？
5. `ccc validate` 路径是否也要触发提示？（建议不触发，仅启动路径触发）

---

## 第 4 轮：最终决策（方案大幅简化为"硬守卫，无交互"）

用户多轮明确指示，最终方案锁定：

1. **不迁移、不帮用户改任何配置**。配置冲突是 ccc 解决不了的，交给用户自己处理。
2. **不剥离、不静默继续**（这是对 #76 当前"静默 strip 后继续"行为的明确改变）。
3. **无交互、无 y/N**：检测到冲突 → **直接报错提示并中止，不启动 claude**。交互与非交互行为一致。
4. **报错信息必须包含**：冲突的具体 key 列表；原因（settings.json 的 `env` 覆盖 ccc 经命令行传入的 provider env，已实测证明，会导致切换 provider 失效）；用户自己的修复方法（从 `~/.claude/settings.json` 的 `env` 删除这些 key；provider 相关配置改在 `~/.claude/ccc.json` 里配）。
5. **作用范围**：正常启动路径 + `ccc validate` 都要做此检测。
6. **检测判据**：settings.json 的 `env` 中存在 `ANTHROPIC_*` / `CLAUDE_*` 前缀 key，或与 base/provider env 同名的 managed key。非冲突的用户自定义 env（如 `MY_CUSTOM_VAR`）不受影响、允许保留。

### 行为变化说明（实施时需注意）

- #76 的 `FilterUserEnvForSettings` 当前承担"静默剥离 `ANTHROPIC_*`/`CLAUDE_*`/managed key"的职责并继续运行；新方案下，命中即**中止**，不会走到剥离继续的逻辑。剥离逻辑是否保留/简化属计划阶段细节（因为中止后 settings.json 仍由用户保有原样，不再被 ccc 改写这些 key）。
- 这是一个**破坏性行为变化**：原本能跑（静默丢配置）的场景，现在会被拦下要求用户先清理 settings.json。需在 README / docs/settings-merge-strategy.md 同步说明。

### 状态：需求与方案已完全对齐，无遗留疑问，待用户给出进入计划模式的指令。

---

## 第 5 轮：用户批准，进入计划模式

用户回复"好"，确认方案。补充探索确认的集成点：
- 启动路径：`internal/cli/exec.go` `runClaude()`，应在 `provider.SwitchWithHook()` 之前做守卫检测。
- 校验路径：`internal/cli/cli.go` `runValidate()` 持有完整 `*config.Config`，应在调用 `validate.Run()` 之前做同样检测（`validate.Config` 接口仅暴露 `Providers()`/`CurrentProvider()`，故守卫放 `runValidate` 内最简单）。
- 检测逻辑应作为 `internal/config` 的导出函数（如 `DetectEnvConflicts`），供两处复用，附错误信息格式化。
- managedEnvKeys = base env keys ∪ provider env keys；判据 = `ANTHROPIC_*`/`CLAUDE_*` 前缀 ∨ 命中 managedEnvKeys。
- TDD：检测函数有逻辑，需单测；两处集成点行为需测试。
- 文档：README.md / README-CN.md / docs/settings-merge-strategy.md 同步破坏性变化。

进入计划模式编写详细 Plan。
