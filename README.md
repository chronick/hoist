# hoist

Git worktree manager for parallel agent workflows.

Creates, manages, and cleans up git worktrees so multiple agents (or humans) can work on the same repo simultaneously without conflicts.

## Install

```bash
go install github.com/chronick/hoist@latest
```

## Usage

```bash
hoist init ~/repos/myapp                 # register a repo
hoist create agent/coder-1               # create worktree + branch
hoist create agent/coder-2 --base=main   # explicit base branch
hoist list                               # show active worktrees
hoist status                             # worktrees with dirty/clean state
hoist reset agent/coder-1                # reset to base branch
hoist pr agent/coder-1 --title "Fix X"   # create PR via gh
hoist merge agent/coder-1                # fast-forward merge to main
hoist destroy agent/coder-1              # remove worktree + branch
```

## How It Works

`hoist` wraps `git worktree` commands with conventions for branch naming, directory layout, and cleanup. Worktrees are created under `.worktrees/` in the repo root by default.

```
~/repos/myapp/                    # main checkout
~/repos/myapp/.worktrees/
  agent-coder-1/                  # worktree on branch agent/coder-1
  agent-coder-2/                  # worktree on branch agent/coder-2
```

## Part of the Agentic Coding Stack

hoist is one tool in a composable stack:

| Tool | Role |
|------|------|
| **skiff** | Container orchestration (lifecycle, health, DNS) |
| **hoist** | Git worktree management |
| **bosun** | Agent entrypoint/coordinator |
| **beads** | Task tracking and priority |
| **agent-mail** | Inter-agent messaging and file leases |
