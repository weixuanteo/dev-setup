package platform

import (
	"testing"

	"devsetup/internal/types"
)

func TestResolveTargetOSFromGOOS(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		override  string
		want      types.TargetOS
		wantError bool
	}{
		{name: "darwin maps to macos", goos: "darwin", want: types.TargetMacOS},
		{name: "linux maps to linux", goos: "linux", want: types.TargetLinux},
		{name: "unsupported host errors", goos: "windows", wantError: true},
		{name: "override bypasses host", goos: "windows", override: "linux", want: types.TargetLinux},
		{name: "invalid override errors", goos: "darwin", override: "bsd", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveTargetOSFromGOOS(tc.goos, tc.override)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
