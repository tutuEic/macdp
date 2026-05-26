package orchestrator

import (
	"fmt"
	"strings"

	"github.com/tutuEic/macdp/internal/store"
)

type ContextBridge struct {
	store *store.DB
}

func NewContextBridge(db *store.DB) *ContextBridge {
	return &ContextBridge{store: db}
}

func (cb *ContextBridge) BuildContext(task *store.Task) string {
	var ctx strings.Builder

	for _, depID := range task.DependsOn {
		dep, err := cb.store.GetTask(depID)
		if err != nil {
			continue
		}
		ctx.WriteString(fmt.Sprintf("## Completed: %s\n", dep.Title))
		if dep.Output != "" {
			output := dep.Output
			if len(output) > 3000 {
				output = output[:3000] + "\n...(truncated)"
			}
			ctx.WriteString(output)
		}
		if len(dep.FilesChanged) > 0 {
			ctx.WriteString("\nChanged files:\n")
			for _, f := range dep.FilesChanged {
				ctx.WriteString(fmt.Sprintf("- %s\n", f))
			}
		}
		ctx.WriteString("\n\n")
	}

	if task.Module != "" {
		ctx.WriteString(fmt.Sprintf("## Your module: %s\n", task.Module))
		ctx.WriteString(fmt.Sprintf("Branch: %s\n\n", task.Branch))
	}

	return ctx.String()
}
