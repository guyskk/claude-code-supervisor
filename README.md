# Claude Code Configuration Switcher (ccc)

[阅读中文文档](README-CN.md)

A command-line tool for switching between different Claude Code configurations.

## Overview

`ccc` (Claude Code Config) allows you to easily switch between different Claude Code provider configurations (e.g., Kimi, GLM, MiniMax) without manually editing configuration files.

## Features

- Switch between multiple Claude Code configurations with a single command
- Automatically updates the `current_provider` setting
- Passes through all arguments to Claude Code
- Supports debug mode with custom configuration directory
- Simple and intuitive command-line interface
- Displays available providers and current provider in help output

## Installation

### Build from Source

Build the tool:
```bash
./build.sh
```

### Build Options

The build script supports multiple platforms and options:

```bash
# Build for current platform only (default)
./build.sh

# Build for all supported platforms
./build.sh --all

# Build for specific platforms (comma-separated)
./build.sh -p darwin-arm64,linux-amd64

# Specify output directory
./build.sh -o ./bin

# Specify binary name
./build.sh -n myccc
```

**Supported platforms:**
- `darwin-amd64` - macOS x86_64
- `darwin-arm64` - macOS ARM64 (Apple Silicon)
- `linux-amd64` - Linux x86_64
- `linux-arm64` - Linux ARM64
- `windows-amd64` - Windows x86_64

### Install System-wide

```bash
# For your current platform
sudo cp dist/ccc-darwin-arm64 /usr/local/bin/ccc

# Or for a specific platform
sudo cp dist/ccc-linux-amd64 /usr/local/bin/ccc
```

## Configuration

Create a `~/.claude/ccc.json` configuration file:

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

The configuration structure:
- `settings`: Base settings template shared by all providers
- `current_provider`: The last used provider (auto-updated)
- `providers`: Provider-specific settings that will be merged with the base

When switching to a provider, the tool:
1. Starts with the base `settings`
2. Deep merges the provider's settings on top
3. Provider settings override base settings for the same keys
4. Saves the merged result to `~/.claude/settings-{provider}.json`

Example configuration files are provided in the `./tmp/example/` directory.

## Usage

### Basic Commands

```bash
# Display help information (shows available providers)
ccc --help

# Run with current provider
ccc

# Switch to and run with a specific provider
ccc kimi

# Pass arguments to Claude Code
ccc kimi --help
ccc kimi /path/to/project

# Use first provider if current_provider is not set
ccc
```

### Environment Variables

- `CCC_CONFIG_DIR`: Override the configuration directory (default: `~/.claude/`)

  Useful for debugging:
  ```bash
  CCC_CONFIG_DIR=./tmp ccc kimi
  ```

### How Provider Switching Works

1. `ccc` reads the `~/.claude/ccc.json` configuration
2. Deep merges the selected provider's settings with the base settings template
3. Writes the merged configuration to `~/.claude/settings-{provider}.json`
4. Updates the `current_provider` field in `ccc.json`
5. Executes `claude --settings ~/.claude/settings-{provider}.json [additional-args...]`

The configuration merge is recursive, so nested objects like `env` and `permissions` are properly merged.

Each provider has its own settings file (e.g., `settings-kimi.json`, `settings-glm.json`), allowing you to easily see and manage different configurations.

## Command Line Reference

```
Usage: ccc [provider] [args...]

Claude Code Configuration Switcher

Commands:
  ccc              Use the current provider (or the first provider if none is set)
  ccc <provider>   Switch to the specified provider and run Claude Code
  ccc --help       Show this help message (displays available providers)

Environment Variables:
  CCC_CONFIG_DIR   Override the configuration directory (default: ~/.claude/)

Examples:
  ccc              Run Claude Code with the current provider
  ccc kimi         Switch to 'kimi' provider and run Claude Code
  ccc glm          Switch to 'glm' provider and run Claude Code
  ccc m2           Switch to 'm2' (MiniMax) provider and run Claude Code
  ccc kimi --help  Switch to 'kimi' and pass --help to Claude Code
```
