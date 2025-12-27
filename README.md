# Claude Code Config Switcher

[阅读中文文档](README-CN.md)

**Switch between multiple Claude Code providers (Kimi, GLM, MiniMax, etc.) with a single command.**

## Overview

`ccc` is a CLI tool that lets you seamlessly switch between different Claude Code API provider configurations. No more manually editing config files—just run `ccc <provider>` and you're done.

## Features

- One-command switching between providers (Kimi, GLM, MiniMax, and more)
- Automatic provider configuration merging
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

# Pass arguments to Claude Code
ccc kimi --help
ccc kimi /path/to/project
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CCC_CONFIG_DIR` | Override config directory (default: `~/.claude/`) |

```bash
# Debug with custom config
CCC_CONFIG_DIR=./tmp ccc kimi
```

## Provider Management

Use the `ccc provider` subcommand to manage your API providers without manually editing JSON files.

### List Providers

```bash
ccc provider list
```

Shows all configured providers with their BASE_URL and MODEL. The current provider is marked with `*`.

### Add a Provider

**Interactive mode** (recommended for first-time setup):
```bash
ccc provider add openai
```

You'll be prompted to enter:
- ANTHROPIC_BASE_URL (must be HTTPS)
- ANTHROPIC_AUTH_TOKEN
- ANTHROPIC_MODEL
- ANTHROPIC_SMALL_FAST_MODEL (optional)

**Non-interactive mode** (for scripts and automation):
```bash
ccc provider add openai \
  --base-url=https://api.openai.com/v1 \
  --token=sk-your-token-here \
  --model=gpt-4 \
  --small-model=gpt-3.5-turbo
```

### Show Provider Details

```bash
ccc provider show kimi
```

Displays the provider's configuration with sensitive tokens masked.

### Update Provider Configuration

```bash
ccc provider set kimi ANTHROPIC_MODEL kimi-k1.5
```

Updates a specific environment variable for a provider. If it's the current provider, the settings file will be automatically regenerated.

### Remove a Provider

```bash
ccc provider remove old-provider
```

Deletes a provider from your configuration. You cannot remove:
- The current provider (switch to another one first)
- The last remaining provider

### Provider Naming Rules

Provider names must:
- Contain only lowercase letters, numbers, hyphens, and underscores
- Not be empty

Examples: `kimi`, `glm-4`, `mini_max`, `openai`

