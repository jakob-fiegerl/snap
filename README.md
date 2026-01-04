# Snap

**A better, simpler, yet powerful Git alternative** (built on Git)

Git is powerful, but let's be honest — it's confusing. Snap gives you the same power with commands that actually make sense.

The name evokes the idea of taking **snapshots**, being **quick/snappy**, and the satisfying **"snap into place"** feeling when things work smoothly.

## Why Snap?

- **No staging area** - the #1 confusion point in Git
- **Conversational commands** - reads like English
- **Smart defaults** - does the right thing 90% of the time
- **Time-based thinking** - "when" instead of just hashes
- **Combined operations** - one command for common workflows

## Implementation Status

Snap is currently under active development. Here's what works today:

| Command | Status | Description |
|---------|--------|-------------|
| `snap init` | ✅ Working | Initialize a new repository |
| `snap changes` | ✅ Working | Show uncommitted changes with colors |
| `snap save` | ✅ Working | Commit changes (with AI or custom message) |
| `snap sync` | ✅ Working | Smart push/pull with conflict detection |
| `snap stack` | ✅ Working | Visual commit history timeline |
| `snap branch` | ✅ Working | Create/switch/delete branches with interactive UI |
| `snap replay` | ✅ Working | Replay commits onto another branch (rebase) |
| `snap undo` | 🚧 Planned | Undo last commit |
| `snap goto` | 🚧 Planned | Time travel through history |
| `snap fork` | 🚧 Planned | Clone repository |
| `snap merge` | 🚧 Planned | Merge branches |
| `snap diff` | 🚧 Planned | Show file changes |
| `snap ignore` | 🚧 Planned | Add to .gitignore |

## Core Commands

### `snap init`
Initialize a new repository in the current directory.

```bash
snap init
```

**What it does:**
- Creates a new Git repository
- Checks if already initialized (prevents accidents)
- Shows helpful next steps

### `snap changes`
See what files have changed. Shows modified, added, and untracked files.

```bash
snap changes
```

### `snap save [message]`
Commit your changes. No staging area confusion, just save what you've changed.

```bash
snap save "Add user authentication"  # Use custom message
snap save                             # AI generates message for you
```

**Interactive options when AI generates:**
- `y` - Accept the suggested message
- `n` - Decline and cancel commit
- `e` - Edit the message before committing

### `snap undo`
Undo your last save. Like `git commit --amend` but clearer.

```bash
snap undo           # Undo last save (keeps changes)
snap undo --hard    # Completely remove last save
snap undo 3         # Go back 3 saves
```

### `snap goto <when>`
Time travel through your project history.

```bash
snap goto yesterday
snap goto "before refactor"
snap goto abc123    # Still supports hashes
```

### `snap branch [subcommand] [name]`
Manage branches with an intuitive, interactive interface. No checkout confusion.

```bash
snap branch                  # Interactive list - navigate and switch branches
snap branch new feature-x    # Create and switch to new branch
snap branch switch main      # Switch to existing branch
snap branch delete old-name  # Delete a branch
```

**Interactive mode features:**
- Navigate with arrow keys or `j`/`k`
- Press `Enter` to switch to selected branch
- Press `n` to create a new branch
- Press `d` to delete selected branch (can't delete current branch)
- Press `?` to toggle help
- Current branch highlighted in green with `*` marker
- Shows upstream tracking and last commit message

**Example interactive view:**
```
Branches

  * main [origin/main] Fixed the login bug
→   feature-login Add login form
    hotfix-123 Quick bug fix

Press ? for help
```

### `snap sync`
Smart push/pull combined. Figures out what you need automatically.

```bash
snap sync        # Pull then push - sync everything
snap sync --from # Only pull changes from remote
```

**What it does:**
- Checks for uncommitted changes (prompts you to save first)
- Pulls latest changes from remote
- Detects and reports merge conflicts
- Pushes your commits to remote
- Automatically sets upstream branch on first push

### `snap stack`
Interactive commit history viewer. Navigate, filter, and checkout commits with ease.

```bash
snap stack           # Interactive commit browser
snap stack --all     # Include all branches
snap stack --mine    # Only your commits
snap stack --plain   # Non-interactive output (for scripts/piping)
snap stack README.md # History for specific file
```

**Interactive features:**
- Navigate with arrow keys or `j`/`k`
- Press `/` to filter commits by message, hash, or author
- Press `Enter` to checkout a specific commit
- Press `g`/`G` to jump to top/bottom
- Press `c` to clear active filter
- Press `?` to toggle help

**Example view:**
```
Commit History

→ ● 2 minutes ago Fixed the login bug
    abc123f by John Doe
  │
  ● 2 hours ago Added user authentication  
    def456a by Jane Smith
  │
  ● yesterday Initial project setup
    789beef by John Doe

↑/k: up  ↓/j: down  g: top  G: bottom  /: filter  c: clear filter
Enter: checkout  ?: toggle help  q: quit
```

### `snap fork <url>`
Clone a repository (because that's what it actually is).

```bash
snap fork https://github.com/user/repo
```

### `snap replay <branch>`
Replay your commits onto another branch. This is what Git calls "rebasing", but with a name that makes sense.

```bash
snap replay main           # Replay current branch commits onto main
snap replay main -i        # Interactive replay (coming soon)
```

**What it does:**
- Shows you exactly which commits will be replayed
- Confirms before making changes
- Handles conflicts gracefully with clear instructions
- Prevents common mistakes (already up-to-date, same branch, etc.)

**Example workflow:**
```bash
# You're on feature-branch with 3 new commits
snap replay main

# Output shows:
Replay commits from 'feature-branch' onto 'main'

The following 3 commit(s) will be replayed:

● Add user profile page (2 hours ago)
  abc123f
│
● Update navigation menu (3 hours ago)
  def456a
│
● Fix login form validation (4 hours ago)
  789beef

Proceed with replay? (y/n):
```

### `snap merge <branch>`
Merge branches. Auto-detects conflicts and opens a clean conflict resolver.

```bash
snap merge feature-x
```

### `snap diff [file]`
See what changed. Kept simple.

```bash
snap diff           # All changes
snap diff README.md  # Specific file
```

### `snap ignore <pattern>`
Add patterns to your ignore file interactively.

```bash
snap ignore "*.log"
snap ignore node_modules
```

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

## Philosophy

Git's complexity comes from its history and Unix philosophy. Snap reimagines version control for modern developers:

1. **No staging area** - You edit files, you save files. Simple.
2. **Human language** - Commands that read like sentences
3. **Smart automation** - The tool should work for you, not the other way around
4. **Visual clarity** - Clean, readable output always
5. **Safety first** - Hard to accidentally destroy work

## Coming from Git?

| Git Command | Snap Equivalent |
|-------------|-----------------|
| `git init` | `snap init` |
| `git status` | `snap changes` |
| `git add . && git commit -m "msg"` | `snap save "msg"` |
| `git commit --amend` | `snap undo` |
| `git checkout <ref>` | `snap goto <ref>` |
| `git checkout -b branch` | `snap branch new branch` |
| `git branch -d branch` | `snap branch delete branch` |
| `git checkout branch` | `snap branch switch branch` |
| `git rebase main` | `snap replay main` |
| `git pull && git push` | `snap sync` |
| `git clone` | `snap fork` |
| `git log` | `snap stack` |

## Examples

```bash
# Start a new project
snap init

# Make some changes, check what changed
snap changes

# Save the changes
snap save "Initial setup"

# Oops, forgot something
snap undo
# ... make changes ...
snap save "Initial setup (complete)"

# Create a new feature
snap branch new user-login

# Save progress
snap save "Add login form"
snap save "Add validation"

# Check what you've done
snap stack

# Sync your work with remote
snap sync

# Keep feature branch up to date with main
snap replay main

# Go back to main and merge
snap goto main
snap merge user-login

# Push everything
snap sync

# See full project history
snap stack --all
```

## License

MIT
