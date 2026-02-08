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
git clone https://github.com/jakob-fiegerl/snap.git
cd snap && ./install.sh

# or build it yourself
go build -o snap && sudo mv snap /usr/local/bin/
```

**Optional** — for AI-generated commit messages:

```bash
ollama pull llama3.2:3b
ollama serve
```

## 🧰 Commands

```
snap init                  Start a new repo
snap save "fixed the bug"  Save your changes
snap save                  Save with an AI-generated message 🤖
snap status                Beautiful status with branch, commit info & changes
snap sync                  Pull + push in one go
snap stack                 Browse your commit history
snap branch                Manage branches interactively
snap replay main           Rebase onto another branch
snap reword                Reword a commit message
snap tag                   List, inspect, diff, or create tags
snap space                 Manage workspaces (isolated dev environments)
```

Run `snap <command> --help` for details on any command.

## 🔄 Coming from Git?

| Git | Snap |
|-----|------|
| `git init` | `snap init` |
| `git status` | `snap status` |
| `git add . && git commit -m "msg"` | `snap save "msg"` |
| `git pull && git push` | `snap sync` |
| `git log` | `snap stack` |
| `git checkout -b feature` | `snap branch new feature` |
| `git rebase main` | `snap replay main` |
| `git commit --amend -m "new msg"` | `snap reword` |
| `git tag -l` | `snap tag` |
| `git show v1.0.0` | `snap tag inspect v1.0.0` |
| `git worktree add ...` | `snap space new <name>` |

## 📋 Requirements

- **Go** 1.24.1+
- **Git**
- **Ollama** + llama3.2:3b *(optional, for AI commit messages)*

## 🌐 Workspaces

Workspaces let you work on multiple features in parallel without git stash. Each workspace is an isolated directory with its own working tree.

### Commands

```bash
snap space new <name>              # Create workspace from current branch
snap space new my-feature --from main  # Create workspace from main
snap space switch my-feature       # Get path to workspace
snap space list                    # List all workspaces
snap space merge my-feature        # Merge workspace into base branch
snap space merge my-feature --into main  # Merge workspace into main
```

### How it works

When you create a workspace:
1. Creates git branch: `workspace/my-feature`
2. Creates worktree: `workspaces/my-feature/`
3. Saves metadata: `.snap/workspaces/my-feature.json`

**Directory structure:**
```
your-repo/
  src/              # main working directory
  workspaces/
    my-feature/     # isolated workspace directory
      src/          # independent copy of code
```

**Metadata:**
```json
{
  "name": "my-feature",
  "branch": "workspace/my-feature",
  "base_branch": "main",
  "base_commit": "abc123"
}
```

### Benefits

- **No git stash needed** - each workspace is a separate directory
- **Switch instantly** - just `cd` to the workspace directory
- **Work in parallel** - edit multiple features simultaneously
- **Clean separation** - uncommitted changes stay in their workspace


## 📄 License

MIT — See [LICENSE](LICENSE) for details.
