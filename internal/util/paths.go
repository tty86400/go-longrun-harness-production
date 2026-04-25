package util

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func ResolveWithinBase(baseDir, relative string) (string, error) {
    if strings.TrimSpace(relative) == "" {
        return "", fmt.Errorf("path is empty")
    }
    if filepath.IsAbs(relative) {
        return "", fmt.Errorf("absolute paths are not allowed: %s", relative)
    }
    cleanBase, err := filepath.Abs(baseDir)
    if err != nil {
        return "", err
    }
    target := filepath.Clean(filepath.Join(cleanBase, relative))
    rel, err := filepath.Rel(cleanBase, target)
    if err != nil {
        return "", err
    }
    if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
        return "", fmt.Errorf("path escapes workspace: %s", relative)
    }
    return target, nil
}

func EnsureDir(path string) error {
    return os.MkdirAll(path, 0o755)
}
