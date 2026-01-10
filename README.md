# Claude Code Config Switcher

[阅读中文文档](README-CN.md)

**Switch between multiple Claude Code providers (Kimi, GLM, MiniMax, etc.) with a single command.**

---

## Why ccc?

`ccc` is a CLI tool that enhances Claude Code with two killer features:

1. **Seamless Provider Switching** - Switch between Kimi, GLM, MiniMax, and other providers with one command
2. **Supervisor Mode** - Automatic task review and iteration that ensures high-quality, deliverable work

Unlike `ralph-claude-code`, Supervisor Mode uses a strict six-step review framework that catches common issues like "asking without doing", "planning without executing", and "missing integration tests".

---

## Quick Start (5 minutes)

### 1. Install

```bash
# Linux / macOS (auto-detect platform)
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

### 2. Configure

Create `~/.claude/ccc.json`:

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "acceptEdits"
    }
  },
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking"
      }
    },
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    }
  }
}
```

> **Note**: This is a minimal configuration to get you started quickly. For complete configuration options including advanced settings, see the [Configuration](#configuration) section below.

### 3. Use

```bash
# Switch to a provider and run Claude Code
ccc kimi

# Run with current provider
ccc

# Pass any Claude Code arguments
ccc glm --help
ccc kimi /path/to/project
```

---

## Supervisor Mode (Recommended)

Supervisor Mode is the most valuable feature of `ccc`. It automatically reviews the Agent's work after each stop and provides feedback if incomplete.

### Enable Supervisor Mode

```bash
export CCC_SUPERVISOR=1
ccc kimi
```

### How It Works

1. Agent completes a task and attempts to stop
2. Supervisor (a Claude instance) reviews the work using a strict six-step framework
3. If work is incomplete or low quality, Supervisor provides feedback
4. Agent continues with the feedback
5. This repeats until Supervisor confirms the work is complete

### What the Supervisor Checks

The Supervisor uses a comprehensive review framework:

| Step | Check |
|------|-------|
| 1 | Understands user's original requirements |
| 2 | Verifies actual work was done (not just questions/plans) |
| 3 | Checks for common traps (asking-only, test loops, fake completion) |
| 4 | Evaluates code quality (no TODOs, has self-review, has tests) |
| 5 | Ensures deliverability (integration tests, deployment-ready) |
| 6 | Provides constructive feedback when rejecting work |

### Key Benefits

- **Catches "asking without doing"** - Agents that only ask questions instead of executing
- **Requires self-review** - Code must be reviewed by the Agent itself
- **Demands integration tests** - No "it should work" - must be verified
- **Prevents early stopping** - Agent must iterate until quality is acceptable
- **Max 20 iterations** - Prevents infinite loops

### Example Output

```
[supervisor] starting supervisor review
[supervisor] iteration count: 1/20
[supervisor] supervisor review completed
[supervisor] work not satisfactory, agent will continue
[supervisor] feedback: The code has TODO comments. Please complete all pending items and add integration tests before stopping.
```

### Logs

Supervisor logs are saved to `~/.claude/ccc/supervisor-{id}.log` for debugging.

---

## Core Features

### Provider Switching

```bash
# Switch to a specific provider
ccc kimi    # Switch to Kimi (Moonshot)
ccc glm     # Switch to GLM (Zhipu AI)
ccc m2      # Switch to MiniMax

# Run with current provider (or first available)
ccc

# See available providers
ccc --help
```

### Configuration Validation

```bash
# Validate current provider
ccc validate

# Validate a specific provider
ccc validate kimi

# Validate all providers (parallel check)
ccc validate --all
```

Output example:
```
Validating 3 provider(s)...

  Valid: kimi
    Base URL: https://api.moonshot.cn/anthropic
    Model: kimi-k2-thinking
    API connection: OK

  Valid: glm
    Base URL: https://open.bigmodel.cn/api/anthropic
    Model: glm-4.7
    API connection: OK

  Warning: m2
    Base URL: https://api.minimaxi.com/anthropic
    Model: MiniMax-M2.1
    API connection: HTTP 503: Service unavailable

All providers valid (1 with API warnings)
```

### Replace Claude Command

`ccc` can completely replace `claude` in your workflow:

```bash
# Instead of: claude --help
ccc --help

# Instead of: claude /path/to/project
ccc /path/to/project

# Instead of: claude --debug --verbose
ccc --debug --verbose
```

All arguments are passed through to Claude Code unchanged.

---

## Configuration

### Config File Location

Default: `~/.claude/ccc.json`
Override with: `CCC_CONFIG_DIR` environment variable

### Config Structure

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "acceptEdits"
    },
    "alwaysThinkingEnabled": true,
    "env": {
      "API_TIMEOUT_MS": "300000",
      "DISABLE_TELEMETRY": "1",
      "DISABLE_ERROR_REPORTING": "1",
      "DISABLE_NON_ESSENTIAL_MODEL_CALLS": "1",
      "DISABLE_BUG_COMMAND": "1",
      "DISABLE_COST_WARNINGS": "1"
    }
  },
  "claude_args": ["--verbose", "--debug"],
  "current_provider": "kimi",
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking",
        "ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2-0905-preview"
      }
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `settings` | Base template shared by all providers |
| `claude_args` | Fixed arguments to pass to Claude Code (optional) |
| `current_provider` | Last used provider (auto-updated) |
| `providers` | Provider-specific overrides |

**How merging works**: Provider settings are deep-merged with the base template. Provider `env` takes precedence over `settings.env`.

### Automatic Migration

If you have an existing `~/.claude/settings.json`, `ccc` can automatically migrate it on first run:

```bash
ccc

# Prompt: "Would you like to create ccc config from existing settings? [y/N]"
# Press 'y' to migrate
```

Migration behavior:
- `env` fields from `settings.json` → `providers.default.env`
- Other fields → `settings` (base template)

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CCC_CONFIG_DIR` | Override config directory (default: `~/.claude/`) |
| `CCC_SUPERVISOR` | Enable Supervisor mode (`"1"` = enable, `"0"` = disable) |

```bash
# Debug with custom config directory
CCC_CONFIG_DIR=./tmp ccc kimi

# Enable Supervisor mode
export CCC_SUPERVISOR=1
ccc kimi
```

---

## Advanced Usage

### Supervisor Configuration

You can configure Supervisor behavior in `~/.claude/ccc-supervisor.json` (optional):

```json
{
  "enabled": true,
  "max_iterations": 20,
  "timeout_seconds": 600
}
```

### Custom Supervisor Prompt

Create `~/.claude/SUPERVISOR.md` to customize the Supervisor prompt. See `internal/cli/supervisor_prompt_default.md` for the default prompt.

---

## Building from Source

```bash
# Build for all platforms
./build.sh --all

# Build for specific platforms
./build.sh -p darwin-arm64,linux-amd64

# Custom output directory
./build.sh -o ./bin
```

**Supported platforms:** `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`

---

## License

MIT License - see LICENSE file for details.
