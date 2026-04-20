package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devsetup/internal/runner"
	"devsetup/internal/types"
)

func TestListPrintsDeterministicOrder(t *testing.T) {
	root := t.TempDir()
	mustWriteScript(t, filepath.Join(root, "runs", "common", "20-common"), "#!/bin/sh\n")
	mustWriteScript(t, filepath.Join(root, "runs", "macos", "10-macos"), "#!/bin/sh\n")

	code, out, errOut := runCLIInDir(t, root, []string{"list", "--os", "macos"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, errOut)
	}

	if !strings.Contains(out, "Resolved scripts:") {
		t.Fatalf("missing resolved scripts header: %s", out)
	}

	first := strings.Index(out, "1. 10-macos [macos]")
	second := strings.Index(out, "2. 20-common [common]")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("unexpected order output: %s", out)
	}
}

func TestRunExecutesScriptsInOrder(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "run.log")
	t.Setenv("DEVSETUP_RUN_LOG", logPath)

	mustWriteScript(t, filepath.Join(root, "runs", "common", "20-common"), `#!/bin/sh
echo 20-common >> "$DEVSETUP_RUN_LOG"
`)
	mustWriteScript(t, filepath.Join(root, "runs", "macos", "10-macos"), `#!/bin/sh
echo 10-macos >> "$DEVSETUP_RUN_LOG"
`)
	mustWriteScript(t, filepath.Join(root, "runs", "macos", "30-macos"), `#!/bin/sh
echo 30-macos >> "$DEVSETUP_RUN_LOG"
`)

	code, _, errOut := runCLIInDir(t, root, []string{"run", "--os", "macos"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, errOut)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	got := strings.TrimSpace(string(data))
	want := strings.Join([]string{"10-macos", "20-common", "30-macos"}, "\n")
	if got != want {
		t.Fatalf("unexpected execution order\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRunContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "run.log")
	t.Setenv("DEVSETUP_RUN_LOG", logPath)

	mustWriteScript(t, filepath.Join(root, "runs", "common", "10-ok"), `#!/bin/sh
echo 10-ok >> "$DEVSETUP_RUN_LOG"
`)
	mustWriteScript(t, filepath.Join(root, "runs", "common", "20-fail"), `#!/bin/sh
echo 20-fail >> "$DEVSETUP_RUN_LOG"
exit 7
`)
	mustWriteScript(t, filepath.Join(root, "runs", "macos", "30-ok"), `#!/bin/sh
echo 30-ok >> "$DEVSETUP_RUN_LOG"
`)

	code, _, _ := runCLIInDir(t, root, []string{"run", "--os", "macos"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	got := strings.TrimSpace(string(data))
	want := strings.Join([]string{"10-ok", "20-fail", "30-ok"}, "\n")
	if got != want {
		t.Fatalf("expected all scripts to run\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRunDryRunExecutesNothing(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "run.log")
	t.Setenv("DEVSETUP_RUN_LOG", logPath)

	mustWriteScript(t, filepath.Join(root, "runs", "common", "10-common"), `#!/bin/sh
echo should-not-run >> "$DEVSETUP_RUN_LOG"
`)
	mustWriteScript(t, filepath.Join(root, "runs", "macos", "20-macos"), `#!/bin/sh
echo should-not-run >> "$DEVSETUP_RUN_LOG"
`)

	code, _, _ := runCLIInDir(t, root, []string{"run", "--os", "macos", "--dry-run"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected no log file, got err=%v", err)
	}
}

func TestLinuxGateBlocksBeforeDiscovery(t *testing.T) {
	deps := defaultDeps()
	deps.resolveOS = func(override string) (types.TargetOS, error) {
		return types.TargetLinux, nil
	}
	deps.checkLinux = func() error {
		return errors.New("linux gate failed")
	}

	discoverCalled := false
	deps.discover = func(opts runner.DiscoverOptions) ([]types.Script, error) {
		discoverCalled = true
		return nil, nil
	}

	executeCalled := false
	deps.execute = func(ctx context.Context, scripts []types.Script, stdout io.Writer, stderr io.Writer) types.RunSummary {
		executeCalled = true
		return types.RunSummary{}
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := executeWithDeps([]string{"run"}, &out, &errOut, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if discoverCalled {
		t.Fatalf("discover should not be called when linux gate fails")
	}
	if executeCalled {
		t.Fatalf("execute should not be called when linux gate fails")
	}
	if !strings.Contains(errOut.String(), "linux gate failed") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func runCLIInDir(t *testing.T, root string, args []string) (int, string, string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func mustWriteScript(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
