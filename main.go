package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/chronick/hoist/internal/pr"
	"github.com/chronick/hoist/internal/repo"
	"github.com/chronick/hoist/internal/worktree"
)

var version = "0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "version", "--version", "-v":
		fmt.Printf("hoist %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	case "init":
		return cmdInit(args)
	case "create":
		return cmdCreate(args)
	case "destroy":
		return cmdDestroy(args)
	case "list":
		return cmdList(args)
	case "status":
		return cmdStatus(args)
	case "reset":
		return cmdReset(args)
	case "pr":
		return cmdPR(args)
	case "merge":
		return cmdMerge(args)
	default:
		return fmt.Errorf("unknown command: %s (run 'hoist help' for usage)", cmd)
	}
}

func printUsage() {
	fmt.Println("hoist: git worktree manager for parallel agent workflows")
	fmt.Println()
	fmt.Println("usage: hoist <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  init     [path]                 register repo for worktree management")
	fmt.Println("  create   <branch> [--base=main]  create worktree + branch")
	fmt.Println("  destroy  <branch>                remove worktree and branch")
	fmt.Println("  list                             show active worktrees")
	fmt.Println("  status                           worktrees with dirty/clean state")
	fmt.Println("  reset    <branch>                reset worktree to base branch")
	fmt.Println("  pr       <branch> [flags]        create PR via gh CLI")
	fmt.Println("  merge    <branch> [--target=main] fast-forward merge to target")
	fmt.Println("  version                          print version")
	fmt.Println()
	fmt.Println("global config (set via 'hoist init'):")
	fmt.Println("  --worktree-dir    worktree directory (default: .worktrees)")
	fmt.Println("  --branch-prefix   branch name prefix (default: agent/)")
}

// cmdInit registers a repo for worktree management.
func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	worktreeDir := fs.String("worktree-dir", "", "worktree directory (default: .worktrees)")
	branchPrefix := fs.String("branch-prefix", "", "branch prefix (default: agent/)")
	baseBranch := fs.String("base-branch", "", "base branch (default: main)")
	fs.Parse(args)

	// Determine repo path
	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	// Resolve to absolute path
	abs, err := resolveRepoPath(repoPath)
	if err != nil {
		return err
	}

	// Verify it's a git repo
	if !repo.IsGitRepo(abs) {
		return fmt.Errorf("%s is not a git repository", abs)
	}

	// Load existing config or use defaults
	cfg, err := repo.Load(abs)
	if err != nil {
		cfg = repo.DefaultConfig()
	}

	// Apply flag overrides
	if *worktreeDir != "" {
		cfg.WorktreeDir = *worktreeDir
	}
	if *branchPrefix != "" {
		cfg.BranchPrefix = *branchPrefix
	}
	if *baseBranch != "" {
		cfg.BaseBranch = *baseBranch
	}

	// Save config
	if err := repo.Save(abs, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Create worktree directory
	if err := repo.EnsureWorktreeDir(abs, cfg); err != nil {
		return fmt.Errorf("create worktree dir: %w", err)
	}

	// Update .gitignore
	if err := repo.EnsureGitignore(abs, cfg); err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}

	fmt.Printf("initialized hoist in %s\n", abs)
	fmt.Printf("  worktree dir:   %s\n", cfg.WorktreeDir)
	fmt.Printf("  branch prefix:  %s\n", cfg.BranchPrefix)
	fmt.Printf("  base branch:    %s\n", cfg.BaseBranch)
	return nil
}

// cmdCreate creates a new worktree and branch.
func cmdCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	base := fs.String("base", "", "base branch (default: from config)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: hoist create <branch-name> [--base=main]")
	}
	branchName := fs.Arg(0)

	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wtPath, err := worktree.Create(repoRoot, cfg, branchName, *base)
	if err != nil {
		return err
	}

	branch := repo.PrefixBranch(cfg, branchName)
	fmt.Printf("created worktree: %s\n", wtPath)
	fmt.Printf("  branch: %s\n", branch)
	return nil
}

// cmdDestroy removes a worktree and its branch.
func cmdDestroy(args []string) error {
	fs := flag.NewFlagSet("destroy", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: hoist destroy <branch-name>")
	}
	branchName := fs.Arg(0)

	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := worktree.Destroy(repoRoot, cfg, branchName); err != nil {
		return err
	}

	branch := repo.PrefixBranch(cfg, branchName)
	fmt.Printf("destroyed worktree for branch: %s\n", branch)
	return nil
}

// cmdList shows active hoist-managed worktrees.
func cmdList(args []string) error {
	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	worktrees, err := worktree.List(repoRoot, cfg)
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("no hoist-managed worktrees")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BRANCH\tPATH\tHEAD")
	for _, wt := range worktrees {
		head := wt.Head
		if len(head) > 8 {
			head = head[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", wt.Branch, wt.Path, head)
	}
	w.Flush()
	return nil
}

// cmdStatus shows dirty/clean state for each worktree.
func cmdStatus(args []string) error {
	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	statuses, err := worktree.Status(repoRoot, cfg)
	if err != nil {
		return err
	}

	if len(statuses) == 0 {
		fmt.Println("no hoist-managed worktrees")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BRANCH\tPATH\tSTATE\tAHEAD/BEHIND")
	for _, s := range statuses {
		state := "clean"
		if s.Dirty {
			state = "dirty"
		}
		ab := s.AheadBehind
		if ab == "" {
			ab = "-"
		} else {
			// Format "behind\tahead" as "ahead/behind"
			parts := strings.Fields(ab)
			if len(parts) == 2 {
				ab = parts[1] + "/" + parts[0]
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Branch, s.Path, state, ab)
	}
	w.Flush()
	return nil
}

// cmdReset resets a worktree to the base branch.
func cmdReset(args []string) error {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: hoist reset <branch-name>")
	}
	branchName := fs.Arg(0)

	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := worktree.Reset(repoRoot, cfg, branchName); err != nil {
		return err
	}

	branch := repo.PrefixBranch(cfg, branchName)
	fmt.Printf("reset %s to %s\n", branch, cfg.BaseBranch)
	return nil
}

// cmdPR creates a GitHub PR from a worktree branch.
func cmdPR(args []string) error {
	fs := flag.NewFlagSet("pr", flag.ExitOnError)
	title := fs.String("title", "", "PR title")
	body := fs.String("body", "", "PR body")
	base := fs.String("base", "", "base branch for PR (default: from config)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: hoist pr <branch-name> [--title '...'] [--body '...'] [--base main]")
	}
	branchName := fs.Arg(0)

	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	return pr.Create(repoRoot, cfg, branchName, *title, *body, *base)
}

// cmdMerge fast-forward merges a branch into the target.
func cmdMerge(args []string) error {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	target := fs.String("target", "", "target branch (default: from config)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: hoist merge <branch-name> [--target main]")
	}
	branchName := fs.Arg(0)

	repoRoot, cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := pr.Merge(repoRoot, cfg, branchName, *target); err != nil {
		return err
	}

	branch := repo.PrefixBranch(cfg, branchName)
	t := *target
	if t == "" {
		t = cfg.BaseBranch
	}
	fmt.Printf("merged %s into %s (fast-forward)\n", branch, t)
	return nil
}

// loadConfig finds the repo root and loads the hoist config.
func loadConfig() (string, repo.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", repo.Config{}, fmt.Errorf("getwd: %w", err)
	}

	repoRoot, err := repo.FindRepoRoot(cwd)
	if err != nil {
		return "", repo.Config{}, err
	}

	cfg, err := repo.Load(repoRoot)
	if err != nil {
		return "", repo.Config{}, err
	}

	return repoRoot, cfg, nil
}

// resolveRepoPath resolves a path argument to an absolute path.
func resolveRepoPath(path string) (string, error) {
	if path == "." || path == "" {
		return os.Getwd()
	}
	abs, err := abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return abs, nil
}

// abs returns the absolute path, handling ~ expansion is left to the shell.
func abs(path string) (string, error) {
	if strings.HasPrefix(path, "/") {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd + "/" + path, nil
}
