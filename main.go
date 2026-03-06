package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("hoist: git worktree manager for parallel agent workflows")
	fmt.Println("usage: hoist <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  init <repo>          register a repo for worktree management")
	fmt.Println("  create <branch>      create worktree + branch")
	fmt.Println("  destroy <branch>     remove worktree and branch")
	fmt.Println("  list                 show active worktrees")
	fmt.Println("  status               worktrees with dirty/clean state")
	fmt.Println("  reset <branch>       reset worktree to base branch")
	fmt.Println("  pr <branch>          create PR via gh CLI")
	fmt.Println("  merge <branch>       fast-forward merge to target")
	return nil
}
