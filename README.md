# packc

> Pack Context — A lightweight CLI tool to prepare code contexts for LLMs.

## Overview

`packc` scans your project, strips unnecessary content (comments, blank lines, trailing spaces), and packs the remaining code into a single text file with a clear project structure. Perfect for feeding code into Large Language Models while minimizing token usage.

## Features

- **Smart scanning** — Automatically skips dependency directories (`node_modules`, `vendor`, `.git`, `dist`, `build`, `__pycache__`, etc.)
- **Token optimization** — Removes comments and compresses whitespace to reduce token consumption
- **Project structure** — Includes a directory tree and file paths in the output, helping LLMs understand your codebase layout
- **File filtering** — Filter by extension and max file size
- **Auto backup** — Backs up previous outputs with timestamp; auto-cleans old backups
- **Token estimation** — Estimates token usage (heuristic-based, no dependencies)
- **Zero dependencies** — Single binary, no runtime required

## Quick Start

### Installation

**Download the binary** for your platform from the releases page, or build from source:

```bash
git clone https://github.com/yourusername/packc
cd packc
go build -o packc
```

### Basic Usage

#### Pack current directory
```bash
packc .
```

#### Pack specific files or directories
```bash
packc ./src ./main.go ./README.md
```

#### Pack only
```bash
packc -ext .go,.js .
```

#### Set max file size
```bash
packc -max-size 500KB .
```

#### Specify output file
```bash
packc -o context.txt .
```

#### Keep comments
```bash
packc -keep-comments .
```

### Configuration File

On first run, `packc` creates a `packc.ini` file next to the binary with default settings:

```ini
[Default]
Extensions = .go,.js,.py,.java,.c,.cpp,.rs,.md,.txt
MaxSize = 1MB
Output = packc_output.txt
KeepComments = false
SkipDeps = true
ShowStats = true
BackupEnabled = true
BackupRetentionDays = 7
```

Command-line flags override the config file values.

## Command Line Options

| Flag             | Description                                                  | Default                            |
| ---------------- | ------------------------------------------------------------ | ---------------------------------- |
| `-ext`           | File extensions to include (comma-separated, e.g., `.go,.js`) | Auto-detect common code extensions |
| `-max-size`      | Maximum file size (e.g., `500KB`, `2MB`)                     | `1MB`                              |
| `-o`             | Output file name                                             | `packc_output.txt`                 |
| `-keep-comments` | Keep comments in output (default: remove)                    | `false`                            |
| `-skip-deps`     | Skip dependency directories                                  | `true`                             |
| `-stats`         | Show statistics after packing                                | `true`                             |

If no path is provided, `packc` scans the current directory (`"."`).

## Output Example

`packc` generates a single text file with this structure:


```bash
# DIRECTORY STRUCTURE
.
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   └── walker/
│       └── scanner.go
└── README.md

# ===== FILE CONTENTS =====

# FILE: cmd/main.go
package main
import "fmt"
func main() {
    fmt.Println("Hello")
}

# FILE: internal/config/config.go
package config
...
```

## Approximation Token Estimation

After packing, `packc` estimates the token count using a heuristic algorithm:

```bash
📊 Token Estimate:
  Characters: 52,341
  Tokens: ~12,847
  Pages: ~25.7 pages
  Cost: $0.3854 (GPT-4)
```

## Backup System

- **Auto-backup**: When `BackupEnabled = true` (default), existing output files are moved to `./backup/` with a timestamp before new output is written.
- **Auto-cleanup**: Backups older than `BackupRetentionDays` are automatically deleted.
- Backup files are stored in `./backup/` (next to the binary).

## Requirements

- **Binary**: No dependencies, just download and run.
- **Source**: Go 1.21+ (if building from source).

## Why packc?

| Feature               | packc | Others                        |
| --------------------- | ----- | ----------------------------- |
| Removes comments      | ✅     | Varies                        |
| Skips dependency dirs | ✅     | Some                          |
| Shows project tree    | ✅     | Rare                          |
| Auto-backup           | ✅     | No                            |
| Token estimation      | ✅     | Rare                          |
| Zero dependencies     | ✅     | Often requires Python/runtime |
| Single binary         | ✅     | Varies                        |

## License

MIT
