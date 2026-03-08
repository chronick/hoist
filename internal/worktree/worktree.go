// Package worktree wraps git worktree operations.
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chronick/hoist/internal/repo"
)

// Create adds a new git worktree with a new branch.
// If the worktree already exists at the expected path, it returns nil (idempotent).
func Create(repoRoot string, cfg repo.Config, branchName, base string) (string, error) {
	branch := repo.PrefixBranch(cfg, branchName)
	wtPath := repo.WorktreePath(repoRoot, cfg, branch)

	// Check if worktree already exists
	if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
		return wtPath, nil
	}

	if base == "" {
		base = cfg.BaseBranch
	}

	// git worktree add <path> -b <branch> <base>
	cmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch, base)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git worktree add: %w", err)
	}

	return wtPath, nil
}

// Destroy removes a worktree and deletes its branch.
// Idempotent: returns nil if the worktree is already gone.
func Destroy(repoRoot string, cfg repo.Config, branchName string) error {
	branch := repo.PrefixBranch(cfg, branchName)
	wtPath := repo.WorktreePath(repoRoot, cfg, branch)

	// Remove worktree if it exists
	if _, err := os.Stat(wtPath); err == nil {
		cmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Try manual cleanup if git worktree remove fails
			os.RemoveAll(wtPath)
			// Prune stale worktree entries
			pruneCmd := exec.Command("git", "worktree", "prune")
			pruneCmd.Dir = repoRoot
			pruneCmd.Run()
		}
	}

	// Delete the branch (ignore errors if already gone)
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repoRoot
	cmd.Run() // intentionally ignore error

	// Clean up empty parent directories in the worktree dir
	cleanEmptyDirs(filepath.Join(repoRoot, cfg.WorktreeDir))

	return nil
}

// cleanEmptyDirs removes empty subdirectories.
func cleanEmptyDirs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			subEntries, err := os.ReadDir(subdir)
			if err == nil && len(subEntries) == 0 {
				os.Remove(subdir)
			}
		}
	}
}

// WorktreeInfo holds information about a single worktree.
type WorktreeInfo struct {
	Path   string
	Branch string
	Head   string
	Bare   bool
}

// List returns all worktrees managed by hoist (those under the worktree dir).
func List(repoRoot string, cfg repo.Config) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	wtDir := filepath.Join(repoRoot, cfg.WorktreeDir)

	var all []WorktreeInfo
	var current WorktreeInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				all = append(all, current)
			}
			current = WorktreeInfo{}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "bare" {
			current.Bare = true
		}
	}
	// Don't forget the last entry
	if current.Path != "" {
		all = append(all, current)
	}

	// Filter to only hoist-managed worktrees
	var managed []WorktreeInfo
	for _, wt := range all {
		if strings.HasPrefix(wt.Path, wtDir) {
			managed = append(managed, wt)
		}
	}

	return managed, nil
}

// WorktreeStatus holds status info for a worktree.
type WorktreeStatus struct {
	WorktreeInfo
	Dirty       bool
	AheadBehind string
}

// Status returns status information for all hoist-managed worktrees.
func Status(repoRoot string, cfg repo.Config) ([]WorktreeStatus, error) {
	worktrees, err := List(repoRoot, cfg)
	if err != nil {
		return nil, err
	}

	var results []WorktreeStatus
	for _, wt := range worktrees {
		ws := WorktreeStatus{WorktreeInfo: wt}

		// Check dirty state
		cmd := exec.Command("git", "-C", wt.Path, "status", "--porcelain")
		out, err := cmd.Output()
		if err == nil {
			ws.Dirty = len(strings.TrimSpace(string(out))) > 0
		}

		// Check ahead/behind
		cmd = exec.Command("git", "-C", wt.Path, "rev-list", "--left-right", "--count",
			fmt.Sprintf("%s...%s", cfg.BaseBranch, wt.Branch))
		out, err = cmd.Output()
		if err == nil {
			ws.AheadBehind = strings.TrimSpace(string(out))
		}

		results = append(results, ws)
	}

	return results, nil
}

// Reset hard-resets a worktree to the base branch and cleans untracked files.
func Reset(repoRoot string, cfg repo.Config, branchName string) error {
	branch := repo.PrefixBranch(cfg, branchName)
	wtPath := repo.WorktreePath(repoRoot, cfg, branch)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", wtPath)
	}

	// git reset --hard <base>
	cmd := exec.Command("git", "-C", wtPath, "reset", "--hard", cfg.BaseBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git reset: %w", err)
	}

	// git clean -fd
	cmd = exec.Command("git", "-C", wtPath, "clean", "-fd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clean: %w", err)
	}

	return nil
}
