package types

import "time"

type TargetOS string

const (
	TargetLinux TargetOS = "linux"
	TargetMacOS TargetOS = "macos"
)

type Script struct {
	Name  string
	Path  string
	Scope string
}

type RunResult struct {
	Script    Script
	StartedAt time.Time
	Duration  time.Duration
	ExitCode  int
	Err       error
}

type RunSummary struct {
	Total     int
	Succeeded int
	Failed    int
	Results   []RunResult
}
