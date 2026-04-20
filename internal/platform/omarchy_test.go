package platform

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckLinuxSupportWithDeps(t *testing.T) {
	t.Run("fails when pacman is missing", func(t *testing.T) {
		err := CheckLinuxSupportWithDeps(LinuxGateDeps{
			LookPath: func(name string) (string, error) {
				return "", errors.New("not found")
			},
			ReadFile: func(path string) ([]byte, error) {
				return []byte("ID=arch\n"), nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "pacman") {
			t.Fatalf("expected pacman error, got: %v", err)
		}
	})

	t.Run("fails when distro is non arch", func(t *testing.T) {
		err := CheckLinuxSupportWithDeps(LinuxGateDeps{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/pacman", nil
			},
			ReadFile: func(path string) ([]byte, error) {
				return []byte("ID=ubuntu\nID_LIKE=debian\n"), nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "Arch-based") {
			t.Fatalf("expected Arch-based error, got: %v", err)
		}
	})

	t.Run("passes on arch-like distro", func(t *testing.T) {
		err := CheckLinuxSupportWithDeps(LinuxGateDeps{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/pacman", nil
			},
			ReadFile: func(path string) ([]byte, error) {
				return []byte("ID=manjaro\nID_LIKE=arch\n"), nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
