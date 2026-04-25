package util

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
    if err != nil {
        return err
    }
    tempName := temp.Name()
    defer os.Remove(tempName)

    if _, err := temp.Write(data); err != nil {
        temp.Close()
        return err
    }
    if err := temp.Chmod(perm); err != nil {
        temp.Close()
        return err
    }
    if err := temp.Sync(); err != nil {
        temp.Close()
        return err
    }
    if err := temp.Close(); err != nil {
        return err
    }
    return os.Rename(tempName, path)
}

func WriteJSONAtomic(path string, v any, perm os.FileMode) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal JSON: %w", err)
    }
    data = append(data, '\n')
    return WriteFileAtomic(path, data, perm)
}
