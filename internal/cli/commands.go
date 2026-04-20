package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"devsetup/internal/platform"
	"devsetup/internal/runner"
	"devsetup/internal/types"
)

type commandDeps struct {
	resolveOS  func(string) (types.TargetOS, error)
	checkLinux func() error
	discover   func(runner.DiscoverOptions) ([]types.Script, error)
	execute    func(context.Context, []types.Script, io.Writer, io.Writer) types.RunSummary
}

func defaultDeps() commandDeps {
	return commandDeps{
		resolveOS:  platform.ResolveTargetOS,
		checkLinux: platform.CheckLinuxSupport,
		discover:   runner.DiscoverScripts,
		execute:    runner.ExecuteScripts,
	}
}

func Execute(args []string, stdout io.Writer, stderr io.Writer) int {
	return executeWithDeps(args, stdout, stderr, defaultDeps())
}

func executeWithDeps(args []string, stdout io.Writer, stderr io.Writer, deps commandDeps) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:], stdout, stderr, deps)
	case "list":
		return listCommand(args[1:], stdout, stderr, deps)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 1
	}
}

type sharedFlags struct {
	TargetOS types.TargetOS
	Only     []string
	Skip     []string
}

func parseSharedFlags(osOverride string, onlyRaw string, skipRaw string, deps commandDeps) (sharedFlags, error) {
	only, err := runner.ParseCSVNames(onlyRaw)
	if err != nil {
		return sharedFlags{}, err
	}

	skip, err := runner.ParseCSVNames(skipRaw)
	if err != nil {
		return sharedFlags{}, err
	}

	targetOS, err := deps.resolveOS(osOverride)
	if err != nil {
		return sharedFlags{}, err
	}

	if targetOS == types.TargetLinux {
		if err := deps.checkLinux(); err != nil {
			return sharedFlags{}, err
		}
	}

	return sharedFlags{
		TargetOS: targetOS,
		Only:     only,
		Skip:     skip,
	}, nil
}

func runCommand(args []string, stdout io.Writer, stderr io.Writer, deps commandDeps) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var osOverride string
	var onlyRaw string
	var skipRaw string
	var dryRun bool

	fs.StringVar(&osOverride, "os", "", "target OS override: linux|macos")
	fs.StringVar(&onlyRaw, "only", "", "comma-separated script names to include")
	fs.StringVar(&skipRaw, "skip", "", "comma-separated script names to skip")
	fs.BoolVar(&dryRun, "dry-run", false, "print the execution plan without executing")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 1
	}

	shared, err := parseSharedFlags(osOverride, onlyRaw, skipRaw, deps)
	if err != nil {
		fmt.Fprintf(stderr, "input error: %v\n", err)
		return 1
	}

	scripts, err := deps.discover(runner.DiscoverOptions{
		RootDir:  ".",
		TargetOS: shared.TargetOS,
		Only:     shared.Only,
		Skip:     shared.Skip,
	})
	if err != nil {
		fmt.Fprintf(stderr, "discovery error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Target OS: %s\n", shared.TargetOS)
	fmt.Fprintln(stdout, "Execution plan:")
	if len(scripts) == 0 {
		fmt.Fprintln(stdout, "(no scripts selected)")
		return 0
	}
	for idx, script := range scripts {
		fmt.Fprintf(stdout, "%d. %s [%s]\n", idx+1, script.Name, script.Scope)
	}

	if dryRun {
		fmt.Fprintln(stdout, "Dry run enabled. No scripts were executed.")
		return 0
	}

	summary := deps.execute(context.Background(), scripts, stdout, stderr)
	runner.PrintSummary(summary, stdout)
	return runner.ExitCode(summary)
}

func listCommand(args []string, stdout io.Writer, stderr io.Writer, deps commandDeps) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var osOverride string
	var onlyRaw string
	var skipRaw string

	fs.StringVar(&osOverride, "os", "", "target OS override: linux|macos")
	fs.StringVar(&onlyRaw, "only", "", "comma-separated script names to include")
	fs.StringVar(&skipRaw, "skip", "", "comma-separated script names to skip")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 1
	}

	shared, err := parseSharedFlags(osOverride, onlyRaw, skipRaw, deps)
	if err != nil {
		fmt.Fprintf(stderr, "input error: %v\n", err)
		return 1
	}

	scripts, err := deps.discover(runner.DiscoverOptions{
		RootDir:  ".",
		TargetOS: shared.TargetOS,
		Only:     shared.Only,
		Skip:     shared.Skip,
	})
	if err != nil {
		fmt.Fprintf(stderr, "discovery error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Target OS: %s\n", shared.TargetOS)
	if len(scripts) == 0 {
		fmt.Fprintln(stdout, "No scripts selected.")
		return 0
	}

	fmt.Fprintln(stdout, "Resolved scripts:")
	for idx, script := range scripts {
		fmt.Fprintf(stdout, "%d. %s [%s]\n", idx+1, script.Name, script.Scope)
	}

	return 0
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  devsetup run [--os linux|macos] [--only a,b] [--skip c,d] [--dry-run]")
	fmt.Fprintln(out, "  devsetup list [--os linux|macos] [--only a,b] [--skip c,d]")
}
