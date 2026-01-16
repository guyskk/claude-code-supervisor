# ccc - Claude Code Supervisor

[English](README.md) | [中文文档](README-CN.md)

## Why ccc?

`ccc` is a CLI tool that enhances Claude Code with two core features:

1. **Supervisor Mode**: ⭐ Automatic task review that ensures high-quality, deliverable work
2. **Seamless Provider Switching**: Switch between Kimi, GLM, MiniMax, and other providers with one command

**Better than `ralph-claude-code`**:

- Supervisor Mode uses a Stop Hook triggered review with a strict framework that significantly improves task completion and quality.
- Unlike ralph's signal-based exit detection, ccc's Supervisor forks the full session context to evaluate actual work quality.
- This prevents fake completions where AI claims "done" but the result has poor quality or unresolved issues.

## Quick Start

### 1. Install

#### Option A: One-line install (Linux / macOS)

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-supervisor/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

#### Option B: Download from [Releases](https://github.com/guyskk/claude-code-supervisor/releases)

Download the binary for your platform (`ccc-darwin-arm64`, `ccc-linux-amd64`, etc.) and install to `/usr/local/bin/`.

### 2. Configure

If you already have `~/.claude/settings.json`, the first time you run `ccc` it will prompt to migrate and automatically generate the ccc config at `~/.claude/ccc.json`.

You can also create the config file manually:

```json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions"
    }
  },
  "providers": {
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    },
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking"
      }
    }
  }
}
```

> **Security Warning**: `bypassPermissions` allows Claude Code to execute tools without confirmation. Only use this in trusted environments.

### 3. Use

```bash
# Show help
ccc --help

# Switch to a provider and run Claude Code
ccc glm

# Run with current provider
ccc

# Pass any Claude Code arguments
ccc glm -p
```

### 4. Validate (Optional)

Verify your provider configuration:

```bash
# Validate current provider
ccc validate

# Validate all providers
ccc validate --all
```

## Patch Command: Replace `claude` with `ccc`

Make `ccc` your default Claude Code by replacing the system `claude` command.

```bash
# Replace claude command with ccc (requires sudo)
sudo ccc patch

# After patching, `claude` command now uses ccc
claude --help    # Shows ccc help

# Restore original claude command
sudo ccc patch --reset
```

## Supervisor Mode (Recommended)

Supervisor Mode is the most valuable feature of `ccc`. It automatically reviews the Agent's work after each stop and provides feedback if incomplete.

### How to Use

1. Start `ccc`, chat with the Agent to confirm requirements and approach:

   ```bash
   ccc
   ```

2. Enable Supervisor Mode using the slash command:

   ```text
   /supervisor OK, start executing
   ```

3. The Agent will execute the task, and Supervisor will automatically review after each stop
   - If work is incomplete, Supervisor provides feedback and Agent continues
   - This repeats until Supervisor confirms the work is complete

### How It Works

1. Agent completes a task and stops, triggering Claude Code's Stop Hook
2. Supervisor (a Claude instance) performs a strict review
3. If work is incomplete or low quality, Supervisor provides feedback
4. Agent continues with the feedback
5. This repeats until Supervisor confirms the work is complete

### Statusline Display

You can configure the statusline in Claude Code with the following command:

```text
/statusline 帮我配置statusline脚本，里面调用 `ccc supervisor-mode` 命令，这个命令会输出 on 或者 off，我希望显示成类似 ... | supervisor on 这样的效果。
```

## Configuration

Config file location, default: `~/.claude/ccc.json`

### Complete Config Example

```json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions"
    },
    "alwaysThinkingEnabled": true
  },
  "supervisor": {
    "max_iterations": 20,
    "timeout_seconds": 600
  },
  "claude_args": ["--verbose"],
  "current_provider": "glm",
  "providers": {
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    },
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

### Config Fields

| Field               | Description                                  |
| ------------------- | -------------------------------------------- |
| `settings`          | Shared Claude Code config template for all providers |
| `supervisor`        | Supervisor mode configuration (optional)     |
| `claude_args`       | Fixed arguments to pass to Claude Code (optional) |
| `current_provider`  | Currently used provider (auto-managed by ccc) |
| `providers.{name}`  | Provider-specific Claude Code configuration  |

### Provider Configuration

Each provider only needs to specify the fields it wants to override. Common fields:

| Field                             | Description                    |
| --------------------------------- | ------------------------------ |
| `env.ANTHROPIC_BASE_URL`          | API endpoint URL               |
| `env.ANTHROPIC_AUTH_TOKEN`        | API key/token                  |
| `env.ANTHROPIC_MODEL`             | Main model to use              |
| `env.ANTHROPIC_SMALL_FAST_MODEL`  | Fast model for quick tasks     |

**How merging works**: Provider settings are deep-merged with the base template. Provider `env` takes precedence over `settings.env`.

### Supervisor Configuration

| Field              | Description                                    | Default |
| ----------------- | ---------------------------------------------- | ------- |
| `max_iterations`  | Maximum iterations before forcing stop         | `20`    |
| `timeout_seconds` | Timeout per supervisor call in seconds        | `600`   |

### Custom Supervisor Prompt

Create `~/.claude/SUPERVISOR.md` to customize the Supervisor prompt. This file overrides the default review behavior with your own instructions.

### Environment Variables

| Variable           | Description                                        |
| ------------------ | -------------------------------------------------- |
| `CCC_CONFIG_DIR`   | Override config directory (default: `~/.claude/`)   |

```bash
# Debug with custom config directory
CCC_CONFIG_DIR=./tmp ccc glm
```

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

## License

MIT License - see LICENSE file for details.
