package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"devsetup/internal/types"
)

func ExecuteScripts(ctx context.Context, scripts []types.Script, stdout io.Writer, stderr io.Writer) types.RunSummary {
	summary := types.RunSummary{
		Total:   len(scripts),
		Results: make([]types.RunResult, 0, len(scripts)),
	}

	for _, script := range scripts {
		fmt.Fprintf(stdout, "\n==> Running %s (%s)\n", script.Name, script.Scope)

		startedAt := time.Now()
		cmd := exec.CommandContext(ctx, script.Path)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		err := cmd.Run()
		duration := time.Since(startedAt)
		exitCode := exitCodeForError(err)

		result := types.RunResult{
			Script:    script,
			StartedAt: startedAt,
			Duration:  duration,
			ExitCode:  exitCode,
			Err:       err,
		}
		summary.Results = append(summary.Results, result)

		if err == nil {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
	}

	return summary
}

func PrintSummary(summary types.RunSummary, out io.Writer) {
	fmt.Fprintln(out, "\nRun Summary")
	fmt.Fprintf(out, "%-20s %-8s %-10s %-8s\n", "SCRIPT", "STATUS", "DURATION", "EXIT")
	for _, result := range summary.Results {
		status := "ok"
		if result.Err != nil {
			status = "failed"
		}
		fmt.Fprintf(out, "%-20s %-8s %-10s %-8d\n", result.Script.Name, status, result.Duration.Round(time.Millisecond), result.ExitCode)
	}
	fmt.Fprintf(out, "Totals: total=%d succeeded=%d failed=%d\n", summary.Total, summary.Succeeded, summary.Failed)
}

func ExitCode(summary types.RunSummary) int {
	if summary.Failed > 0 {
		return 1
	}
	return 0
}

func exitCodeForError(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}
