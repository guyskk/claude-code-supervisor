# Claude Code Config Switcher

[阅读中文文档](README-CN.md)

**Switch between multiple Claude Code providers (Kimi, GLM, MiniMax, etc.) with a single command.**

## Overview

`ccc` is a CLI tool that lets you seamlessly switch between different Claude Code API provider configurations. No more manually editing config files—just run `ccc <provider>` and you're done.

## Features

- One-command switching between providers (Kimi, GLM, MiniMax, and more)
- Automatic provider configuration merging
- Configuration validation with API connectivity testing
- Pass-through of all Claude Code arguments
- Debug mode with custom config directories
- Clean, intuitive CLI interface

## Installation

### Download from Releases

Pre-built binaries are available on the [Releases page](https://github.com/guyskk/claude-code-config-switcher/releases).

```bash
# Download for your platform
curl -LO https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-$(uname -s)-$(uname -m)

# Install system-wide
sudo chmod +x ccc-$(uname -s)-$(uname -m)
sudo mv ccc-$(uname -s)-$(uname -m) /usr/local/bin/ccc

# Verify installation
ccc --version
```

**Supported platforms:** `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`, `windows-amd64.exe`

### Build from Source

```bash
# Build for all platforms
./build.sh --all

# Build for specific platforms
./build.sh -p darwin-arm64,linux-amd64

# Custom output directory
./build.sh -o ./bin
```

**Supported platforms:** `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`, `windows-amd64`

## Configuration

Create `~/.claude/ccc.json`:

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
        "ANTHROPIC_MODEL": "glm-4.7",
        "ANTHROPIC_SMALL_FAST_MODEL": "glm-4.7"
      }
    },
    "m2": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "MiniMax-M2.1",
        "ANTHROPIC_SMALL_FAST_MODEL": "MiniMax-M2.1"
      }
    }
  }
}
```

**Config structure:**
- `settings` — Base template shared by all providers
- `current_provider` — Last used provider (auto-updated)
- `providers` — Provider-specific overrides

**How it works:** When switching providers, `ccc` deep-merges the provider's config with the base template, then saves it to `~/.claude/settings-{provider}.json`.

See `./tmp/example/` for more examples.

### Automatic Migration

If you have an existing Claude Code `settings.json` file, `ccc` can automatically migrate it to the new `ccc.json` format on first run.

When you run `ccc` for the first time and `~/.claude/ccc.json` doesn't exist:
1. `ccc` detects if `~/.claude/settings.json` exists
2. Prompts you to confirm migration: `Would you like to create ccc config from existing settings? [y/N]`
3. If you confirm, creates `ccc.json` with:
   - Your existing settings (permissions, thinking mode, etc.) as the base template
   - Your API configuration (`env` fields) moved to a `default` provider
4. Your original `settings.json` is left unchanged

**Migration behavior:**
- `env` fields from `settings.json` → `providers.default.env`
- All other fields → `settings` (shared base template)
- `current_provider` set to `default`

## Usage

```bash
# Show available providers
ccc --help

# Run with current provider
ccc

# Switch to a provider
ccc kimi

# Validate current provider configuration
ccc validate

# Validate a specific provider
ccc validate kimi

# Validate all providers
ccc validate --all

# Pass arguments to Claude Code
ccc kimi --help
ccc kimi /path/to/project
```

### Validation Command

The `ccc validate` command helps you verify your provider configurations:

- **Checks configuration format**: Validates JSON syntax and required fields
- **Validates environment variables**: Ensures `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` are present
- **Tests API connectivity**: Attempts a connection to the provider's API endpoint
- **Shows detailed results**: Color-coded output (green=valid, red=invalid, yellow=warning)

Example output:
```
$ ccc validate --all
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
    API connection: HTTP 503: <error message>

2/3 providers valid, 1 with API warnings
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CCC_CONFIG_DIR` | Override config directory (default: `~/.claude/`) |

```bash
# Debug with custom config
CCC_CONFIG_DIR=./tmp ccc kimi
```
