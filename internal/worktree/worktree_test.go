package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chronick/hoist/internal/repo"
)

func TestCreateIdempotent(t *testing.T) {
	// Verify that Create returns the expected path when the directory
	// already exists (idempotent case), without calling git.
	dir := t.TempDir()
	cfg := repo.Config{
		WorktreeDir:  ".worktrees",
		BranchPrefix: "agent/",
		BaseBranch:   "main",
	}

	// Pre-create the worktree directory to simulate existing worktree
	wtPath := repo.WorktreePath(dir, cfg, "agent/feature-x")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create should return the path without error (idempotent)
	got, err := Create(dir, cfg, "feature-x", "main")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got != wtPath {
		t.Errorf("Create path = %q, want %q", got, wtPath)
	}
}

func TestDestroyIdempotent(t *testing.T) {
	// Destroy should not error on a non-existent worktree.
	dir := t.TempDir()
	cfg := repo.Config{
		WorktreeDir:  ".worktrees",
		BranchPrefix: "agent/",
		BaseBranch:   "main",
	}

	err := Destroy(dir, cfg, "nonexistent")
	if err != nil {
		t.Fatalf("Destroy non-existent: %v", err)
	}
}

func TestCleanEmptyDirs(t *testing.T) {
	dir := t.TempDir()

	// Create an empty subdirectory
	emptyDir := filepath.Join(dir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a non-empty subdirectory
	notEmptyDir := filepath.Join(dir, "notempty")
	if err := os.Mkdir(notEmptyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notEmptyDir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	cleanEmptyDirs(dir)

	// Empty dir should be gone
	if _, err := os.Stat(emptyDir); err == nil {
		t.Error("expected empty dir to be removed")
	}
	// Non-empty dir should remain
	if _, err := os.Stat(notEmptyDir); err != nil {
		t.Error("expected non-empty dir to remain")
	}
}
