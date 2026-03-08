// Package repo handles .hoist.json configuration and repo registration.
package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ConfigFile = ".hoist.json"

// Config is the .hoist.json structure stored in the repo root.
type Config struct {
	WorktreeDir  string `json:"worktree_dir"`
	BranchPrefix string `json:"branch_prefix"`
	BaseBranch   string `json:"base_branch"`
}

// DefaultConfig returns the default hoist configuration.
func DefaultConfig() Config {
	return Config{
		WorktreeDir:  ".worktrees",
		BranchPrefix: "agent/",
		BaseBranch:   "main",
	}
}

// Load reads .hoist.json from the given repo root.
func Load(repoRoot string) (Config, error) {
	path := filepath.Join(repoRoot, ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w (run 'hoist init' first)", ConfigFile, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", ConfigFile, err)
	}
	return cfg, nil
}

// Save writes .hoist.json to the given repo root.
func Save(repoRoot string, cfg Config) error {
	path := filepath.Join(repoRoot, ConfigFile)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// IsGitRepo checks if the given path contains a .git directory or file.
func IsGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}

// EnsureWorktreeDir creates the worktree directory if it doesn't exist.
func EnsureWorktreeDir(repoRoot string, cfg Config) error {
	dir := filepath.Join(repoRoot, cfg.WorktreeDir)
	return os.MkdirAll(dir, 0755)
}

// EnsureGitignore adds the worktree directory to .gitignore if not already present.
func EnsureGitignore(repoRoot string, cfg Config) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	entry := cfg.WorktreeDir + "/"

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	// Check if entry already exists
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == entry || trimmed == cfg.WorktreeDir {
			return nil // already present
		}
	}

	// Append the entry
	var content string
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		content = string(data) + "\n" + entry + "\n"
	} else {
		content = string(data) + entry + "\n"
	}
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// SafeBranchName converts a branch name to a safe directory name.
// Replaces / with - to create a flat directory structure.
func SafeBranchName(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// PrefixBranch adds the configured branch prefix if not already present.
func PrefixBranch(cfg Config, name string) string {
	if strings.HasPrefix(name, cfg.BranchPrefix) {
		return name
	}
	return cfg.BranchPrefix + name
}

// WorktreePath returns the full path to a worktree given the branch name.
func WorktreePath(repoRoot string, cfg Config, branch string) string {
	safe := SafeBranchName(branch)
	return filepath.Join(repoRoot, cfg.WorktreeDir, safe)
}

// FindRepoRoot walks up from the given path to find the repo root
// (directory containing .hoist.json or .git).
func FindRepoRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	dir := abs
	for {
		// Check for .hoist.json first
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			return dir, nil
		}
		// Check for .git
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not in a git repository (searched up from %s)", start)
}
