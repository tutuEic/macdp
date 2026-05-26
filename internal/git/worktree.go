package git

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tutuEic/macdp/internal/store"
)

// Manager handles git worktree operations for task isolation.
// Each task gets its own worktree so agents can work in parallel without conflicts.
type Manager struct {
	baseRepo     string // path to the main git repo
	worktreeDir  string // directory for worktrees (e.g., ".macdp/worktrees")
	branchPrefix string // prefix for task branches (e.g., "macdp/")
}

// NewManager creates a git worktree manager.
func NewManager(baseRepo, worktreeDir, branchPrefix string) *Manager {
	return &Manager{
		baseRepo:     baseRepo,
		worktreeDir:  worktreeDir,
		branchPrefix: branchPrefix,
	}
}

// Prepare creates a worktree for a task.
// Returns the worktree path the agent should work in.
func (m *Manager) Prepare(ctx context.Context, task *store.Task) (string, error) {
	if m.baseRepo == "" {
		return "", fmt.Errorf("no base repo configured")
	}

	taskBranch := m.branchPrefix + task.ID
	worktreePath := filepath.Join(m.worktreeDir, "task-"+task.ID)

	// Ensure worktree directory exists
	if err := os.MkdirAll(m.worktreeDir, 0755); err != nil {
		return "", fmt.Errorf("create worktree dir: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		log.Printf("[git] Worktree already exists for %s, reusing", task.ID)
		task.Worktree = worktreePath
		task.Branch = taskBranch
		return worktreePath, nil
	}

	// Create branch from main
	cmd := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "checkout", "-b", taskBranch, "main")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Try from HEAD if main doesn't exist
		cmd2 := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "checkout", "-b", taskBranch)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("create branch %s: %s / %s", taskBranch, string(out), string(out2))
		}
		log.Printf("[git] Created branch %s from HEAD (main not found)", taskBranch)
	} else {
		log.Printf("[git] Created branch %s from main", taskBranch)
	}

	// Create worktree
	cmd = exec.CommandContext(ctx, "git", "-C", m.baseRepo, "worktree", "add", worktreePath, taskBranch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("worktree add: %s: %w", string(out), err)
	}

	log.Printf("[git] Worktree created: %s (branch: %s)", worktreePath, taskBranch)

	task.Worktree = worktreePath
	task.Branch = taskBranch
	return worktreePath, nil
}

// Cleanup removes the worktree and optionally merges changes.
func (m *Manager) Cleanup(ctx context.Context, task *store.Task, mergeChanges bool) error {
	if task.Worktree == "" {
		return nil
	}

	worktreePath := task.Worktree
	branch := task.Branch

	// Commit any changes made by the agent
	if err := m.commitChanges(ctx, worktreePath, fmt.Sprintf("macdp: %s (%s)", task.Title, task.ID)); err != nil {
		log.Printf("[git] Commit warning for %s: %v", task.ID, err)
	}

	// Merge back if requested and task succeeded
	if mergeChanges && task.Status == store.TaskDone {
		if err := m.mergeBranch(ctx, branch); err != nil {
			log.Printf("[git] Merge failed for %s: %v", task.ID, err)
			// Don't fail the cleanup — merge failure is not fatal
		}
	}

	// Remove worktree
	cmd := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "worktree", "remove", worktreePath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[git] Worktree remove warning for %s: %s", task.ID, string(out))
	}

	// Delete the branch
	cmd = exec.CommandContext(ctx, "git", "-C", m.baseRepo, "branch", "-D", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[git] Branch delete warning for %s: %s", task.ID, string(out))
	}

	// Clean up the directory if still exists
	os.RemoveAll(worktreePath)

	log.Printf("[git] Cleaned up worktree for %s", task.ID)
	return nil
}

// commitChanges stages and commits all changes in the worktree.
func (m *Manager) commitChanges(ctx context.Context, worktreePath, message string) error {
	// Check if there are any changes
	statusCmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "status", "--porcelain")
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(statusOut))) == 0 {
		return nil // no changes to commit
	}

	// Stage all changes
	addCmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "add", "-A")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", string(out), err)
	}

	// Commit
	commitCmd := exec.CommandContext(ctx, "git", "-C", worktreePath,
		"commit", "-m", message,
		"--author=macdp <macdp@local>",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", string(out), err)
	}

	log.Printf("[git] Committed changes in %s: %s", worktreePath, message)
	return nil
}

// mergeBranch merges a task branch into the base branch.
func (m *Manager) mergeBranch(ctx context.Context, branch string) error {
	// First, checkout the base branch
	checkoutCmd := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "checkout", "main")
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		// Try whatever the current branch is
		log.Printf("[git] Checkout main failed, trying current: %s", string(out))
	}

	// Merge the task branch
	mergeCmd := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "merge", branch, "--no-ff", "-m",
		fmt.Sprintf("macdp: Merge %s", branch))
	out, err := mergeCmd.CombinedOutput()
	if err != nil {
		// Try to abort merge
		abortCmd := exec.CommandContext(ctx, "git", "-C", m.baseRepo, "merge", "--abort")
		abortCmd.Run()
		return fmt.Errorf("merge %s: %s: %w", branch, string(out), err)
	}

	log.Printf("[git] Merged branch %s into main", branch)
	return nil
}

// GetChanges returns the list of files changed in a worktree.
func (m *Manager) GetChanges(worktreePath string) ([]string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--name-only", "HEAD~1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Try diff against the initial commit
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--name-only", "--cached")
		out, err = cmd.CombinedOutput()
		if err != nil {
			return nil, nil
		}
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// IsRepo checks if a directory is a git repository.
func IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// ensureTimeout sets a default timeout if none is set on the context.
func ensureTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, 30*time.Second)
}
