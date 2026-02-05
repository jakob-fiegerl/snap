# 📸 Snap - Git management in a fraction of seconds

**Git, but make it make sense.**

Snap wraps Git in commands that actually read like English. No staging area, no cryptic flags — just save your work and move on.

## ✨ Features

- 🎯 **No staging area** — edit files, save changes, done
- 🤖 **AI commit messages** — let Ollama write them for you
- 💬 **Conversational commands** — `snap save` instead of `git add && git commit`
- 🔄 **Smart sync** — combined push/pull with conflict detection
- 📊 **Visual history** — interactive commit timeline with filtering
- 🌿 **Branch management** — create, switch, and delete branches effortlessly
- 🔀 **Rebase simplified** — replay commits with clear previews
- 🏷️ **Tag management** — list, diff, and create tags
- 🎨 **Beautiful TUI** — modern, colorful terminal interface

## ⚡ Quick Start

```bash
# install
git clone https://github.com/yourusername/snap.git
cd snap && ./install.sh

# or build it yourself
go build -o snap && sudo mv snap /usr/local/bin/
```

**Optional** — for AI-generated commit messages:

```bash
ollama pull phi4
ollama serve
```

## 🧰 Commands

```
snap init                  Start a new repo
snap save "fixed the bug"  Save your changes
snap save                  Save with an AI-generated message 🤖
snap changes               See what's different
snap sync                  Pull + push in one go
snap stack                 Browse your commit history
snap branch                Manage branches interactively
snap replay main           Rebase onto another branch
snap tags                  List, diff, or create tags
```

Run `snap <command> --help` for details on any command.

## 🔄 Coming from Git?

| Git | Snap |
|-----|------|
| `git init` | `snap init` |
| `git status` | `snap changes` |
| `git add . && git commit -m "msg"` | `snap save "msg"` |
| `git pull && git push` | `snap sync` |
| `git log` | `snap stack` |
| `git checkout -b feature` | `snap branch new feature` |
| `git rebase main` | `snap replay main` |
| `git tag -l` | `snap tags` |

## 📋 Requirements

- **Go** 1.24.1+
- **Git**
- **Ollama** + Phi-4 *(optional, for AI commit messages)*

## 📄 License

MIT — See [LICENSE](LICENSE) for details.
