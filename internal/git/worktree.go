package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager handles git worktree operations for isolated agent workspaces.
type WorktreeManager struct {
	RepoPath     string
	WorktreeDir  string
	BranchPrefix string
}

// NewWorktreeManager creates a new worktree manager.
func NewWorktreeManager(repoPath string) *WorktreeManager {
	return &WorktreeManager{
		RepoPath:     repoPath,
		WorktreeDir:  filepath.Join(repoPath, ".macdp", "worktrees"),
		BranchPrefix: "macdp/",
	}
}

// Create creates a new git worktree for a task.
func (wm *WorktreeManager) Create(taskID, baseBranch string) (string, error) {
	if baseBranch == "" {
		baseBranch = "main"
	}

	branchName := wm.BranchPrefix + taskID
	worktreePath := filepath.Join(wm.WorktreeDir, taskID)

	// Ensure worktree dir exists
	if err := os.MkdirAll(wm.WorktreeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree dir: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return worktreePath, nil // already exists
	}

	// git worktree add -b branchName path baseBranch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = wm.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add failed: %s: %w", string(output), err)
	}

	return worktreePath, nil
}

// Remove removes a git worktree.
func (wm *WorktreeManager) Remove(taskID string) error {
	worktreePath := filepath.Join(wm.WorktreeDir, taskID)

	// git worktree remove path
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = wm.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", string(output), err)
	}

	return nil
}

// CommitAll stages and commits all changes in a worktree.
func (wm *WorktreeManager) CommitAll(taskID, message string) error {
	worktreePath := filepath.Join(wm.WorktreeDir, taskID)

	// git add -A
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s: %w", string(output), err)
	}

	// git commit -m "message"
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if there's nothing to commit
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %s: %w", string(output), err)
	}

	return nil
}

// MergeInto merges a task branch into the target branch.
func (wm *WorktreeManager) MergeInto(taskID, targetBranch string) error {
	branchName := wm.BranchPrefix + taskID

	// Checkout target branch in main repo
	cmd := exec.Command("git", "checkout", targetBranch)
	cmd.Dir = wm.RepoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s failed: %s: %w", targetBranch, string(output), err)
	}

	// git merge --squash branchName
	cmd = exec.Command("git", "merge", "--squash", branchName)
	cmd.Dir = wm.RepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge failed: %s: %w", string(output), err)
	}

	// git commit
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("macdp: merge task %s", taskID))
	cmd.Dir = wm.RepoPath
	output, err = cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "nothing to commit") {
		return fmt.Errorf("git commit after merge failed: %s: %w", string(output), err)
	}

	return nil
}

// ListWorktrees lists all active MACDP worktrees.
func (wm *WorktreeManager) ListWorktrees() ([]string, error) {
	entries, err := os.ReadDir(wm.WorktreeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var worktrees []string
	for _, e := range entries {
		if e.IsDir() {
			worktrees = append(worktrees, e.Name())
		}
	}
	return worktrees, nil
}

// CleanupAll removes all MACDP worktrees.
func (wm *WorktreeManager) CleanupAll() error {
	worktrees, err := wm.ListWorktrees()
	if err != nil {
		return err
	}
	for _, id := range worktrees {
		wm.Remove(id)
	}
	return nil
}
