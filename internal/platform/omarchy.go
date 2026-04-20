package platform

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type LinuxGateDeps struct {
	LookPath func(string) (string, error)
	ReadFile func(string) ([]byte, error)
}

func CheckLinuxSupport() error {
	return CheckLinuxSupportWithDeps(LinuxGateDeps{
		LookPath: exec.LookPath,
		ReadFile: os.ReadFile,
	})
}

func CheckLinuxSupportWithDeps(deps LinuxGateDeps) error {
	if deps.LookPath == nil || deps.ReadFile == nil {
		return errors.New("internal error: linux gate dependencies are not configured")
	}

	if _, err := deps.LookPath("pacman"); err != nil {
		return errors.New("linux target requires pacman in PATH (Omarchy/Arch-based). Install pacman or run with --os macos")
	}

	data, err := deps.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("linux target requires /etc/os-release to confirm an Arch-based distro: %w", err)
	}

	id, idLike := parseOSRelease(data)
	if strings.EqualFold(id, "arch") || containsArchLike(idLike) {
		return nil
	}

	return fmt.Errorf("linux target is restricted to Arch-based systems (ID=%q, ID_LIKE=%q). Use --os macos on macOS", id, idLike)
}

func parseOSRelease(data []byte) (id string, idLike string) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			idLike = value
		}
	}

	return id, idLike
}

func containsArchLike(raw string) bool {
	for _, token := range strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return r == ' ' || r == '\t' || r == ','
	}) {
		if token == "arch" {
			return true
		}
	}
	return false
}
