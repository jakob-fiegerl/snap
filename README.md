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
snap tag                  List, inspect, diff, or create tags
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

## 📋 Requirements

- **Go** 1.24.1+
- **Git**
- **Ollama** + llama3.2:3b *(optional, for AI commit messages)*

## 🌐 Future commands

### Workspaces
`snap new <name>` Create workspace.
`snap new my-feature`
`snap new experiment --from develop`
`snap switch <name>` Switch workspace. Uncommitted changes stay in previous workspace.
`snap switch my-feature`
`snap list` List all workspaces.
`snap merge <workspace> [--into main]` Merge workspace.

#### Technical Documentation

```bash
snap new my-feature --from main
```

1. Create git branch: `workspace/my-feature`
2. Create worktree: `git worktree add workspaces/my-feature workspace/my-feature`
3. Save metadata: `.snap/workspaces/my-feature.json`

**Directory structure:**
```
workspaces/
  my-feature/          # isolated directory
    .git              # points to .git/worktrees/my-feature
    src/
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

##### Workspace Switching

```bash
snap switch my-feature
```

1. Validate workspace exists
2. Change directory to `workspaces/my-feature/`
3. Update `.snap/state.json` current workspace

**No git stash needed** - each workspace is a separate directory.

##### Workspace Merging

```bash
snap merge auth-ai --into main
```

1. If `--into main`: checkout main in original repo
2. Merge `workspace/auth-ai` branch
3. Delete workspace branch and directory

**Implementation:**
```bash
# Into main
git checkout main
git merge workspace/auth-ai
git worktree remove workspaces/auth-ai
git branch -D workspace/auth-ai

# Into current workspace
cd workspaces/my-feature
git merge workspace/auth-ai
```


## 📄 License

MIT — See [LICENSE](LICENSE) for details.
