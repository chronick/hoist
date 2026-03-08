package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"agent/feature-foo", "agent-feature-foo"},
		{"agent/fix/nested", "agent-fix-nested"},
		{"simple-branch", "simple-branch"},
		{"a/b/c/d", "a-b-c-d"},
		{"no-slashes", "no-slashes"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SafeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("SafeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrefixBranch(t *testing.T) {
	cfg := Config{BranchPrefix: "agent/"}
	tests := []struct {
		input string
		want  string
	}{
		{"feature-foo", "agent/feature-foo"},
		{"agent/feature-foo", "agent/feature-foo"},
		{"fix-bug", "agent/fix-bug"},
		{"agent/nested/thing", "agent/nested/thing"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := PrefixBranch(cfg, tt.input)
			if got != tt.want {
				t.Errorf("PrefixBranch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrefixBranchCustomPrefix(t *testing.T) {
	cfg := Config{BranchPrefix: "bot/"}
	got := PrefixBranch(cfg, "task-123")
	want := "bot/task-123"
	if got != want {
		t.Errorf("PrefixBranch(%q) = %q, want %q", "task-123", got, want)
	}
	// Already prefixed
	got = PrefixBranch(cfg, "bot/task-123")
	if got != want {
		t.Errorf("PrefixBranch(%q) = %q, want %q", "bot/task-123", got, want)
	}
}

func TestWorktreePath(t *testing.T) {
	cfg := Config{WorktreeDir: ".worktrees"}
	got := WorktreePath("/repo", cfg, "agent/feature-foo")
	want := filepath.Join("/repo", ".worktrees", "agent-feature-foo")
	if got != want {
		t.Errorf("WorktreePath = %q, want %q", got, want)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.WorktreeDir != ".worktrees" {
		t.Errorf("WorktreeDir = %q, want %q", cfg.WorktreeDir, ".worktrees")
	}
	if cfg.BranchPrefix != "agent/" {
		t.Errorf("BranchPrefix = %q, want %q", cfg.BranchPrefix, "agent/")
	}
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", cfg.BaseBranch, "main")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		WorktreeDir:  ".worktrees",
		BranchPrefix: "agent/",
		BaseBranch:   "main",
	}

	// Save
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Verify JSON structure
	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed != cfg {
		t.Errorf("parsed config = %+v, want %+v", parsed, cfg)
	}

	// Load
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != cfg {
		t.Errorf("loaded config = %+v, want %+v", loaded, cfg)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error loading missing config")
	}
}

func TestIsGitRepo(t *testing.T) {
	// Not a git repo
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("expected false for non-git dir")
	}

	// Create .git directory
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if !IsGitRepo(dir) {
		t.Error("expected true for dir with .git")
	}
}

func TestEnsureGitignore(t *testing.T) {
	cfg := Config{WorktreeDir: ".worktrees"}

	t.Run("creates new gitignore", func(t *testing.T) {
		dir := t.TempDir()
		if err := EnsureGitignore(dir, cfg); err != nil {
			t.Fatalf("EnsureGitignore: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if got := string(data); got != ".worktrees/\n" {
			t.Errorf("gitignore = %q, want %q", got, ".worktrees/\n")
		}
	})

	t.Run("appends to existing gitignore", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644)
		if err := EnsureGitignore(dir, cfg); err != nil {
			t.Fatalf("EnsureGitignore: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		want := "*.log\n.worktrees/\n"
		if got := string(data); got != want {
			t.Errorf("gitignore = %q, want %q", got, want)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n"), 0644)
		if err := EnsureGitignore(dir, cfg); err != nil {
			t.Fatalf("EnsureGitignore: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		want := ".worktrees/\n"
		if got := string(data); got != want {
			t.Errorf("gitignore = %q, want %q", got, want)
		}
	})

	t.Run("handles missing trailing newline", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log"), 0644)
		if err := EnsureGitignore(dir, cfg); err != nil {
			t.Fatalf("EnsureGitignore: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		want := "*.log\n.worktrees/\n"
		if got := string(data); got != want {
			t.Errorf("gitignore = %q, want %q", got, want)
		}
	})
}

func TestFindRepoRoot(t *testing.T) {
	// Create a temp dir with .git
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0755)

	// Create a nested directory
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0755)

	root, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	if root != dir {
		t.Errorf("FindRepoRoot = %q, want %q", root, dir)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindRepoRoot(dir)
	if err == nil {
		t.Fatal("expected error for non-repo dir")
	}
}
