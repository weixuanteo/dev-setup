package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"devsetup/internal/types"
)

type DiscoverOptions struct {
	RootDir  string
	TargetOS types.TargetOS
	Only     []string
	Skip     []string
}

func ParseCSVNames(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("invalid CSV value %q: empty script name", raw)
		}
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			return nil, fmt.Errorf("invalid script name %q: expected base filename", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	return out, nil
}

func DiscoverScripts(opts DiscoverOptions) ([]types.Script, error) {
	root := opts.RootDir
	if strings.TrimSpace(root) == "" {
		root = "."
	}

	if opts.TargetOS != types.TargetLinux && opts.TargetOS != types.TargetMacOS {
		return nil, fmt.Errorf("unsupported target OS %q", opts.TargetOS)
	}

	candidates := []struct {
		scope string
		dir   string
	}{
		{scope: "common", dir: filepath.Join(root, "runs", "common")},
		{scope: string(opts.TargetOS), dir: filepath.Join(root, "runs", string(opts.TargetOS))},
	}

	var scripts []types.Script
	seenNames := map[string]string{}

	for _, candidate := range candidates {
		entries, err := os.ReadDir(candidate.dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("required directory missing: %s", candidate.dir)
			}
			return nil, fmt.Errorf("read %s: %w", candidate.dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("stat %s/%s: %w", candidate.dir, name, err)
			}
			if !info.Mode().IsRegular() {
				continue
			}

			if prevScope, ok := seenNames[name]; ok {
				return nil, fmt.Errorf("duplicate script name %q found in scopes %q and %q", name, prevScope, candidate.scope)
			}

			absolutePath, err := filepath.Abs(filepath.Join(candidate.dir, name))
			if err != nil {
				return nil, fmt.Errorf("resolve path for %s/%s: %w", candidate.dir, name, err)
			}

			scripts = append(scripts, types.Script{
				Name:  name,
				Path:  absolutePath,
				Scope: candidate.scope,
			})
			seenNames[name] = candidate.scope
		}
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	filtered, err := applyOnlyFilter(scripts, opts.Only)
	if err != nil {
		return nil, err
	}

	filtered, err = applySkipFilter(filtered, opts.Skip)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}

func applyOnlyFilter(scripts []types.Script, only []string) ([]types.Script, error) {
	if len(only) == 0 {
		return scripts, nil
	}

	available := nameSetFromScripts(scripts)
	unknown := unknownNames(only, available)
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown --only script names: %s", strings.Join(unknown, ", "))
	}

	keep := listToSet(only)
	out := make([]types.Script, 0, len(scripts))
	for _, script := range scripts {
		if _, ok := keep[script.Name]; ok {
			out = append(out, script)
		}
	}
	return out, nil
}

func applySkipFilter(scripts []types.Script, skip []string) ([]types.Script, error) {
	if len(skip) == 0 {
		return scripts, nil
	}

	available := nameSetFromScripts(scripts)
	unknown := unknownNames(skip, available)
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown --skip script names: %s", strings.Join(unknown, ", "))
	}

	remove := listToSet(skip)
	out := make([]types.Script, 0, len(scripts))
	for _, script := range scripts {
		if _, ok := remove[script.Name]; ok {
			continue
		}
		out = append(out, script)
	}
	return out, nil
}

func listToSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}

func nameSetFromScripts(scripts []types.Script) map[string]struct{} {
	set := make(map[string]struct{}, len(scripts))
	for _, script := range scripts {
		set[script.Name] = struct{}{}
	}
	return set
}

func unknownNames(requested []string, available map[string]struct{}) []string {
	unknown := make([]string, 0)
	seen := map[string]struct{}{}
	for _, name := range requested {
		if _, already := seen[name]; already {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := available[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	return unknown
}
