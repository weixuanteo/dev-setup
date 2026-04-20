package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"devsetup/internal/types"
)

func TestDiscoverScriptsMergeAndOrder(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "runs", "common"))
	mustMkdirAll(t, filepath.Join(root, "runs", "macos"))

	mustWriteFile(t, filepath.Join(root, "runs", "common", "20-common"), "#!/bin/sh\n")
	mustWriteFile(t, filepath.Join(root, "runs", "common", ".hidden"), "#!/bin/sh\n")
	mustMkdirAll(t, filepath.Join(root, "runs", "common", "subdir"))
	mustWriteFile(t, filepath.Join(root, "runs", "macos", "10-macos"), "#!/bin/sh\n")

	scripts, err := DiscoverScripts(DiscoverOptions{RootDir: root, TargetOS: types.TargetMacOS})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := []string{scripts[0].Name, scripts[1].Name}
	want := []string{"10-macos", "20-common"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if scripts[0].Scope != "macos" || scripts[1].Scope != "common" {
		t.Fatalf("unexpected scopes: %+v", scripts)
	}
}

func TestDiscoverScriptsDuplicateName(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "runs", "common"))
	mustMkdirAll(t, filepath.Join(root, "runs", "macos"))

	mustWriteFile(t, filepath.Join(root, "runs", "common", "bun"), "#!/bin/sh\n")
	mustWriteFile(t, filepath.Join(root, "runs", "macos", "bun"), "#!/bin/sh\n")

	_, err := DiscoverScripts(DiscoverOptions{RootDir: root, TargetOS: types.TargetMacOS})
	if err == nil {
		t.Fatalf("expected duplicate name error")
	}
}

func TestDiscoverScriptsFilters(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "runs", "common"))
	mustMkdirAll(t, filepath.Join(root, "runs", "macos"))

	mustWriteFile(t, filepath.Join(root, "runs", "common", "bun"), "#!/bin/sh\n")
	mustWriteFile(t, filepath.Join(root, "runs", "common", "tmux"), "#!/bin/sh\n")
	mustWriteFile(t, filepath.Join(root, "runs", "macos", "zed"), "#!/bin/sh\n")

	t.Run("only filter", func(t *testing.T) {
		scripts, err := DiscoverScripts(DiscoverOptions{
			RootDir:  root,
			TargetOS: types.TargetMacOS,
			Only:     []string{"bun", "zed"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := namesOf(scripts)
		want := []string{"bun", "zed"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("skip filter", func(t *testing.T) {
		scripts, err := DiscoverScripts(DiscoverOptions{
			RootDir:  root,
			TargetOS: types.TargetMacOS,
			Skip:     []string{"tmux"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := namesOf(scripts)
		want := []string{"bun", "zed"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("only then skip", func(t *testing.T) {
		scripts, err := DiscoverScripts(DiscoverOptions{
			RootDir:  root,
			TargetOS: types.TargetMacOS,
			Only:     []string{"bun", "tmux"},
			Skip:     []string{"tmux"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := namesOf(scripts)
		want := []string{"bun"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("unknown only", func(t *testing.T) {
		_, err := DiscoverScripts(DiscoverOptions{
			RootDir:  root,
			TargetOS: types.TargetMacOS,
			Only:     []string{"missing"},
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("unknown skip after only", func(t *testing.T) {
		_, err := DiscoverScripts(DiscoverOptions{
			RootDir:  root,
			TargetOS: types.TargetMacOS,
			Only:     []string{"bun"},
			Skip:     []string{"zed"},
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestParseCSVNames(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		want      []string
		wantError bool
	}{
		{name: "empty", in: "", want: nil},
		{name: "normal", in: "bun, tmux", want: []string{"bun", "tmux"}},
		{name: "dedup", in: "bun,bun", want: []string{"bun"}},
		{name: "invalid empty entry", in: "bun,", wantError: true},
		{name: "invalid path", in: "../bun", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCSVNames(tc.in)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func namesOf(scripts []types.Script) []string {
	out := make([]string, 0, len(scripts))
	for _, script := range scripts {
		out = append(out, script.Name)
	}
	return out
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
