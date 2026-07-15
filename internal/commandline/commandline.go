package commandline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func Parse(command string) (string, []string, error) {
	parts, err := windows.DecomposeCommandLine(strings.TrimSpace(command))
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", nil, fmt.Errorf("empty command")
	}
	return parts[0], parts[1:], nil
}

func ParseArgs(args string) ([]string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, nil
	}
	parts, err := windows.DecomposeCommandLine("cmd " + args)
	if err != nil {
		return nil, err
	}
	if len(parts) <= 1 {
		return nil, nil
	}
	return parts[1:], nil
}

func ResolveExecutable(exe, workDir string) string {
	if exe == "" || filepath.IsAbs(exe) || workDir == "" {
		return exe
	}
	candidate := filepath.Join(workDir, exe)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return exe
}
