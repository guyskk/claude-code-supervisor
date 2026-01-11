# ccc - Claude Code Supervisor

[阅读中文文档](README-CN.md)

**Auto-review and iterate until quality work is delivered. Switch between multiple Claude Code providers with one command.**

---

## Why ccc?

`ccc` is a CLI tool that enhances Claude Code with two killer features:

1. **Supervisor Mode** ⭐ - Automatic task review that ensures high-quality, deliverable work (most valuable)
2. **Seamless Provider Switching** - Switch between Kimi, GLM, MiniMax, and other providers with one command

**Better than `ralph-claude-code`**: Supervisor Mode uses a stop-hook triggered review with a strict six-step framework that significantly improves task completion and quality. Unlike `ralph`'s signal-based exit detection (counting "done" signals or test loops), ccc's Supervisor forks the full session context and evaluates actual work quality—requiring self-review, integration tests, and deployment-ready code. This prevents fake completions where AI claims "done" but the result has poor quality or unresolved issues.

---

## Quick Start (5 minutes)

### 1. Install

**Option A: One-line install (Linux / macOS)**

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

**Option B: Download from [Releases](https://github.com/guyskk/claude-code-config-switcher/releases)**

Download the binary for your platform (`ccc-darwin-arm64`, `ccc-linux-amd64`, etc.) and install to `/usr/local/bin/`.

### 2. Configure

Create `~/.claude/ccc.json`:

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "bypassPermissions"
    }
  },
  "supervisor": {
    "enabled": true,
    "max_iterations": 20,
    "timeout_seconds": 600
  },
  "current_provider": "kimi",
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

> **Security Warning**: `bypassPermissions` allows Claude Code to execute tools without confirmation. Only use this in trusted environments.
>
> **Note**: `current_provider` is auto-managed by `ccc`. For full configuration options, see [Configuration](#configuration).

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

### 4. Validate (Optional)

Verify your provider configuration:

```bash
# Validate current provider
ccc validate

# Validate all providers
ccc validate --all
```

---

## 💡 Pro Tip: Enable Supervisor Mode

Supervisor Mode is the **most valuable feature** of ccc. Once you've completed the Quick Start, enable it by setting `supervisor.enabled: true` in your `ccc.json` config.

See [Supervisor Mode](#supervisor-mode-recommended) below for details.

---

## Supervisor Mode (Recommended)

Supervisor Mode is the most valuable feature of `ccc`. It automatically reviews the Agent's work after each stop and provides feedback if incomplete.

### Enable Supervisor Mode

**Default (config file)**: Set `supervisor.enabled: true` in your `ccc.json` (see Quick Start above).

**Temporary override**: Use the `CCC_SUPERVISOR` environment variable to temporarily override the config:

```bash
# Force enable (even if config.enabled = false)
export CCC_SUPERVISOR=1
ccc kimi

# Force disable (even if config.enabled = true)
export CCC_SUPERVISOR=0
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

### Breaking Change: Supervisor Config Location

**Important**: If you have an existing ccc configuration with `supervisor` nested inside `settings`, you must move it to the top level.

**Old format (no longer supported):**
```json
{
  "settings": {
    "supervisor": {
      "enabled": true,
      "max_iterations": 20,
      "timeout_seconds": 600
    }
  }
}
```

**New format (required):**
```json
{
  "settings": {},
  "supervisor": {
    "enabled": true,
    "max_iterations": 20,
    "timeout_seconds": 600
  }
}
```

The `supervisor` configuration must now be at the top level of `ccc.json`, as a sibling to `settings`, `providers`, and `current_provider`.

### Config File Location

Default: `~/.claude/ccc.json`
Override with: `CCC_CONFIG_DIR` environment variable

### Complete Config Example

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "bypassPermissions"
    },
    "alwaysThinkingEnabled": true,
    "enabledPlugins": {
      "gopls-lsp@claude-plugins-official": true
    },
    "env": {
      "API_TIMEOUT_MS": "300000",
      "DISABLE_TELEMETRY": "1",
      "DISABLE_ERROR_REPORTING": "1"
    }
  },
  "supervisor": {
    "enabled": true,
    "max_iterations": 20,
    "timeout_seconds": 600
  },
  "claude_args": ["--verbose"],
  "current_provider": "kimi",
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking",
        "ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2-0905-preview"
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

### Config Fields

| Field | Description |
|-------|-------------|
| `settings` | Base template shared by all providers |
| `settings.permissions` | Permission settings (allow list, default mode) |
| `settings.env` | Environment variables shared by all providers |
| `settings.*` | Any other Claude Code settings (plugins, thinking mode, etc.) |
| `supervisor` | Supervisor mode configuration (top-level) |
| `claude_args` | Fixed arguments to pass to Claude Code (optional) |
| `current_provider` | Last used provider (auto-managed by ccc) |
| `providers.{name}` | Provider-specific configuration |

### Provider Configuration

Each provider only needs to specify the fields it wants to override. Common fields:

| Field | Description |
|-------|-------------|
| `env.ANTHROPIC_BASE_URL` | API endpoint URL |
| `env.ANTHROPIC_AUTH_TOKEN` | API key/token |
| `env.ANTHROPIC_MODEL` | Main model to use |
| `env.ANTHROPIC_SMALL_FAST_MODEL` | Fast model for quick tasks |

**How merging works**: Provider settings are deep-merged with the base template. Provider `env` takes precedence over `settings.env`.

### Supervisor Configuration

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable Supervisor mode | `false` |
| `max_iterations` | Maximum iterations before forcing stop | `20` |
| `timeout_seconds` | Timeout per supervisor call | `600` |

Can be overridden with `CCC_SUPERVISOR=1` environment variable.

### Custom Supervisor Prompt

Create `~/.claude/SUPERVISOR.md` to customize the Supervisor prompt. This file overrides the default review behavior with your own instructions.

### Automatic Migration

If you have an existing `~/.claude/settings.json`, `ccc` will prompt to migrate it on first run:

- Your `env` fields are moved to `providers.default.env`
- Other fields become the base `settings` template
- Your original file is not modified

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
