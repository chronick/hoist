// Package pr wraps gh CLI for pull request and merge operations.
package pr

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chronick/hoist/internal/repo"
)

// Create pushes a branch and creates a GitHub PR via gh CLI.
func Create(repoRoot string, cfg repo.Config, branchName, title, body, base string) error {
	branch := repo.PrefixBranch(cfg, branchName)
	wtPath := repo.WorktreePath(repoRoot, cfg, branch)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", wtPath)
	}

	if base == "" {
		base = cfg.BaseBranch
	}

	// Push branch to origin
	pushCmd := exec.Command("git", "-C", wtPath, "push", "-u", "origin", branch)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	// Build gh pr create command
	args := []string{"pr", "create", "--head", branch, "--base", base}
	if title != "" {
		args = append(args, "--title", title)
	}
	if body != "" {
		args = append(args, "--body", body)
	}

	prCmd := exec.Command("gh", args...)
	prCmd.Dir = repoRoot
	prCmd.Stdout = os.Stdout
	prCmd.Stderr = os.Stderr
	if err := prCmd.Run(); err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}

	return nil
}

// Merge performs a fast-forward merge of a branch into the target.
func Merge(repoRoot string, cfg repo.Config, branchName, target string) error {
	branch := repo.PrefixBranch(cfg, branchName)
	wtPath := repo.WorktreePath(repoRoot, cfg, branch)

	if target == "" {
		target = cfg.BaseBranch
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", wtPath)
	}

	// Verify worktree is clean
	statusCmd := exec.Command("git", "-C", wtPath, "status", "--porcelain")
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		return fmt.Errorf("worktree has uncommitted changes, commit or stash first")
	}

	// Checkout target branch in main repo
	checkoutCmd := exec.Command("git", "checkout", target)
	checkoutCmd.Dir = repoRoot
	checkoutCmd.Stdout = os.Stdout
	checkoutCmd.Stderr = os.Stderr
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s: %w", target, err)
	}

	// Fast-forward merge
	mergeCmd := exec.Command("git", "merge", "--ff-only", branch)
	mergeCmd.Dir = repoRoot
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr
	if err := mergeCmd.Run(); err != nil {
		return fmt.Errorf("git merge --ff-only %s: %w (branch may not be fast-forwardable)", branch, err)
	}

	return nil
}
