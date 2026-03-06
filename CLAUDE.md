# CLAUDE.md

## Project

**hoist** -- Git worktree manager for parallel agent workflows.

Go 1.22+, zero external dependencies (stdlib + git/gh CLI).

## Commands

```bash
go build -o hoist .          # build
go test ./...                # test all
```

## Layout

```
main.go                      # cobra CLI entry point
internal/
  worktree/worktree.go       # git worktree operations
  repo/repo.go               # repo registration and state
  pr/pr.go                   # gh pr create wrapper
```

## Key Design

- Wraps `git worktree add/remove/list` -- no custom git logic
- PR creation via `gh pr create` -- requires gh CLI
- Repo state stored in `.hoist.json` in repo root
- Worktrees created under `.worktrees/` by default
- Branch naming convention: configurable prefix (default `agent/`)
- No daemon, no config file, no database
- Idempotent: create is safe to re-run, destroy cleans up fully

## Beads Workflow Integration

<!-- br-agent-instructions-v1 -->
See skiff CLAUDE.md for beads workflow details.
<!-- end-br-agent-instructions -->
