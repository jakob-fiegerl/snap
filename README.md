# Snap 🚀

[![Go Version](https://img.shields.io/badge/Go-1.24.1+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/yourusername/snap.svg)](https://github.com/yourusername/snap/stargazers)

**The Git alternative that makes version control feel natural** ✨

Snap takes the power of Git and wraps it in commands that actually make sense. No more staging area confusion, cryptic flags, or remembering obscure commands. Just intuitive, conversational Git operations.

> "Git is powerful, but let's be honest — it's confusing. Snap gives you the same power with commands that read like English."

## ✨ Features

- 🎯 **No staging area** - Edit files, save changes. Simple.
- 💬 **Conversational commands** - `snap save` instead of `git add && git commit`
- 🤖 **AI-powered commit messages** - Let Ollama generate meaningful messages for you
- ⏰ **Time-based navigation** - `snap goto yesterday` instead of hunting for hashes
- 🔄 **Smart sync** - Combined push/pull with conflict detection
- 📊 **Visual history** - Interactive commit timeline with filtering
- 🌿 **Branch management** - Create, switch, and delete branches effortlessly
- 🔀 **Rebase simplified** - Replay commits with clear previews
- 🎨 **Beautiful TUI** - Modern, colorful terminal interface

## 📊 Status

| Command | Status | Description |
|---------|--------|-------------|
| `snap init` | ✅ Working | Initialize a new repository |
| `snap changes` | ✅ Working | Show uncommitted changes with colors |
| `snap save` | ✅ Working | Commit changes (with AI or custom message) |
| `snap sync` | ✅ Working | Smart push/pull with conflict detection |
| `snap stack` | ✅ Working | Visual commit history timeline |
| `snap branch` | ✅ Working | Create/switch/delete branches with interactive UI |
| `snap replay` | ✅ Working | Replay commits onto another branch (rebase) |
| `snap tags` | ✅ Working | List all tags sorted by date (newest first) |
| `snap tags diff` | ✅ Working | Show commits since the most recent tag |
| `snap tags create` | ✅ Working | Create and push a new annotated tag |
| `snap undo` | 🚧 Planned | Undo last commit |
| `snap goto` | 🚧 Planned | Time travel through history |
| `snap fork` | 🚧 Planned | Clone repository |
| `snap merge` | 🚧 Planned | Merge branches |
| `snap diff` | 🚧 Planned | Show file changes |
| `snap ignore` | 🚧 Planned | Add to .gitignore |

## 🚀 Quick Start

### Prerequisites

- **Go** 1.24.1 or higher
- **Git** (Snap is built on top of Git)
- **Ollama** with Phi-4 model (optional - for AI commit messages)

### Installation

```bash
# Clone and install
git clone https://github.com/yourusername/snap.git
cd snap
./install.sh

# Or build manually
go build -o snap
sudo mv snap /usr/local/bin/
```

### Setup AI Commit Messages (Optional)

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Download Phi-4 model
ollama pull phi4

# Start Ollama service
ollama serve
```

## 📖 Core Commands

### `snap init`
Initialize a new repository.

```bash
snap init
```

### `snap changes`
Show uncommitted changes with colors.

```bash
snap changes
```

### `snap save [message]`
Commit your changes. AI generates messages automatically, or use custom messages.

```bash
snap save                    # AI-generated message
snap save "Add user auth"    # Custom message
```

**Interactive options:** `y` (accept), `n` (cancel), `e` (edit)

### `snap sync`
Smart push/pull with conflict detection.

```bash
snap sync        # Pull then push
snap sync --from # Pull only
```

### `snap stack`
Interactive commit history viewer.

```bash
snap stack           # Browse commits
snap stack --all     # All branches
snap stack --mine    # Your commits only
snap stack --plain   # Non-interactive
```

### `snap branch`
Manage branches interactively.

```bash
snap branch                  # Interactive list
snap branch new feature-x    # Create & switch
snap branch switch main      # Switch branch
snap branch delete old-name  # Delete branch
```

### `snap replay <branch>`
Replay commits onto another branch (rebase).

```bash
snap replay main
```

Shows preview and confirms before proceeding.

### `snap tags`
List all tags sorted by date (newest first).

```bash
snap tags              # List all tags with details
```

Shows tag name, commit hash, message, and relative time.

#### `snap tags diff`
Shows commits since the most recent tag.

```bash
snap tags diff
```

Displays commit hash, message, additions/deletions, and relative time.

#### `snap tags create <version>`
Creates and pushes a new annotated tag.

```bash
snap tags create v1.2.0
```

Shows a preview of commits since the last tag, then creates the tag with an auto-generated message and pushes it to the remote.

## Quick Start

### Prerequisites

1. **Go** (1.24.1 or higher)
2. **Git** (Snap is built on top of Git)
3. **Ollama** with Phi-4 (optional - for AI-powered commit messages)

### Installation

```bash
# Clone and install
git clone https://github.com/yourusername/snap.git
cd snap
./install.sh

# Or build manually
go build -o snap
sudo mv snap /usr/local/bin/
```

### Setup Ollama (Optional - for AI commit messages)

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Download Phi-4 model
ollama pull phi4
```

## How It Works

Snap is a wrapper around Git that translates intuitive commands into Git operations. You get all the power of Git with none of the confusion.

Behind the scenes:
- `snap init` → `git init`
- `snap changes` → `git status --short`
- `snap save` → `git add -A && git commit`
- `snap stack` → `git log` with visual formatting
- `snap sync` → `git pull && git push` (with conflict detection)
- `snap undo` → `git reset` (soft/hard depending on flags)
- `snap goto` → `git checkout` with smart date/message parsing
- `snap branch` → `git branch` + `git checkout -b` with interactive TUI
- `snap replay` → `git rebase` with visual preview and confirmation
- `snap tags` → `git for-each-ref refs/tags` with formatted output

## Philosophy

Git's complexity comes from its history and Unix philosophy. Snap reimagines version control for modern developers:

1. **No staging area** - You edit files, you save files. Simple.
2. **Human language** - Commands that read like sentences
3. **Smart automation** - The tool should work for you, not the other way around
4. **Visual clarity** - Clean, readable output always
5. **Safety first** - Hard to accidentally destroy work

## 🔄 Coming from Git?

| Git Command | Snap Equivalent |
|-------------|-----------------|
| `git init` | `snap init` |
| `git status` | `snap changes` |
| `git add . && git commit -m "msg"` | `snap save "msg"` |
| `git checkout <ref>` | `snap goto <ref>` |
| `git checkout -b branch` | `snap branch new branch` |
| `git rebase main` | `snap replay main` |
| `git pull && git push` | `snap sync` |
| `git log` | `snap stack` |
| `git tag -l` | `snap tags` |

## 🛠️ How It Works

Snap wraps Git with intuitive commands while preserving full Git power:

- **No staging area** - Direct commits from working directory
- **AI assistance** - Ollama generates conventional commit messages
- **Visual TUIs** - Interactive interfaces for complex operations
- **Smart defaults** - Sensible behavior without flags
- **Safety first** - Confirmations and conflict detection

## 🤝 Contributing

Snap is open source! Help make Git better for everyone:

- 🐛 [Report bugs](https://github.com/yourusername/snap/issues)
- 💡 [Suggest features](https://github.com/yourusername/snap/issues)
- 🛠️ [Submit PRs](https://github.com/yourusername/snap/pulls)
- 📖 [Improve docs](https://github.com/yourusername/snap/wiki)

## 📄 License

MIT - See [LICENSE](LICENSE) for details
