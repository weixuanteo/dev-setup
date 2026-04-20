package platform

import (
	"fmt"
	"runtime"
	"strings"

	"devsetup/internal/types"
)

func ParseTargetOS(raw string) (types.TargetOS, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(types.TargetLinux):
		return types.TargetLinux, nil
	case string(types.TargetMacOS):
		return types.TargetMacOS, nil
	default:
		return "", fmt.Errorf("unknown --os value %q (expected linux|macos)", raw)
	}
}

func ResolveTargetOS(override string) (types.TargetOS, error) {
	return ResolveTargetOSFromGOOS(runtime.GOOS, override)
}

func ResolveTargetOSFromGOOS(goos, override string) (types.TargetOS, error) {
	if strings.TrimSpace(override) != "" {
		return ParseTargetOS(override)
	}

	switch goos {
	case "darwin":
		return types.TargetMacOS, nil
	case "linux":
		return types.TargetLinux, nil
	default:
		return "", fmt.Errorf("unsupported host OS %q; use --os linux|macos to override", goos)
	}
}
