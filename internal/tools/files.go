package tools

import (
    "context"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "longrunharness/internal/config"
    "longrunharness/internal/util"
)

type filesReadTool struct {
    cfg       config.FilesConfig
    workspace string
}

type filesWriteTool struct {
    cfg       config.FilesConfig
    workspace string
}

type filesAppendTool struct {
    cfg       config.FilesConfig
    workspace string
}

type filesListTool struct {
    cfg       config.FilesConfig
    workspace string
}

func NewFilesReadTool(cfg config.FilesConfig, workspace string) Tool {
    return &filesReadTool{cfg: cfg, workspace: workspace}
}
func NewFilesWriteTool(cfg config.FilesConfig, workspace string) Tool {
    return &filesWriteTool{cfg: cfg, workspace: workspace}
}
func NewFilesAppendTool(cfg config.FilesConfig, workspace string) Tool {
    return &filesAppendTool{cfg: cfg, workspace: workspace}
}
func NewFilesListTool(cfg config.FilesConfig, workspace string) Tool {
    return &filesListTool{cfg: cfg, workspace: workspace}
}

func (t *filesReadTool) Name() string { return "files.read" }
func (t *filesReadTool) Description() string { return "Read a UTF-8 text file from the sandboxed workspace." }
func (t *filesReadTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"path"}, "properties": map[string]any{"path": map[string]any{"type": "string"}, "max_bytes": map[string]any{"type": "integer"}}}
}
func (t *filesReadTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    _ = ctx
    rel := getStringArg(args, "path")
    maxBytes := getIntArg(args, "max_bytes", t.cfg.MaxReadBytes)
    if maxBytes <= 0 || maxBytes > t.cfg.MaxReadBytes {
        maxBytes = t.cfg.MaxReadBytes
    }
    path, err := util.ResolveWithinBase(t.workspace, rel)
    if err != nil {
        return Result{}, err
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return Result{}, err
    }
    truncated := false
    if len(data) > maxBytes {
        data = data[:maxBytes]
        truncated = true
    }
    return Result{OK: true, Summary: "file read", Output: string(data), Files: []string{rel}, Truncated: truncated}, nil
}

func (t *filesWriteTool) Name() string { return "files.write" }
func (t *filesWriteTool) Description() string { return "Write a UTF-8 text file into the sandboxed workspace." }
func (t *filesWriteTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"path", "content"}, "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "create_dirs": map[string]any{"type": "boolean"}}}
}
func (t *filesWriteTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    _ = ctx
    rel := getStringArg(args, "path")
    content := getStringArg(args, "content")
    if len(content) > t.cfg.MaxWriteBytes {
        return Result{}, fmt.Errorf("content exceeds max_write_bytes")
    }
    path, err := util.ResolveWithinBase(t.workspace, rel)
    if err != nil {
        return Result{}, err
    }
    if getBoolArg(args, "create_dirs", true) {
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
            return Result{}, err
        }
    }
    if err := util.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
        return Result{}, err
    }
    return Result{OK: true, Summary: "file written", Files: []string{rel}, Metadata: map[string]any{"bytes": len(content)}}, nil
}

func (t *filesAppendTool) Name() string { return "files.append" }
func (t *filesAppendTool) Description() string { return "Append UTF-8 text to a file in the sandboxed workspace." }
func (t *filesAppendTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"path", "content"}, "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "create_dirs": map[string]any{"type": "boolean"}}}
}
func (t *filesAppendTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    _ = ctx
    rel := getStringArg(args, "path")
    content := getStringArg(args, "content")
    if len(content) > t.cfg.MaxWriteBytes {
        return Result{}, fmt.Errorf("content exceeds max_write_bytes")
    }
    path, err := util.ResolveWithinBase(t.workspace, rel)
    if err != nil {
        return Result{}, err
    }
    if getBoolArg(args, "create_dirs", true) {
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
            return Result{}, err
        }
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return Result{}, err
    }
    defer f.Close()
    if _, err := f.WriteString(content); err != nil {
        return Result{}, err
    }
    return Result{OK: true, Summary: "file appended", Files: []string{rel}, Metadata: map[string]any{"bytes": len(content)}}, nil
}

func (t *filesListTool) Name() string { return "files.list" }
func (t *filesListTool) Description() string { return "List files in a workspace directory, optionally recursively." }
func (t *filesListTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "recursive": map[string]any{"type": "boolean"}, "max_entries": map[string]any{"type": "integer"}}}
}
func (t *filesListTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    _ = ctx
    rel := getStringArg(args, "path")
    if rel == "" {
        rel = "."
    }
    recursive := getBoolArg(args, "recursive", false)
    maxEntries := getIntArg(args, "max_entries", t.cfg.MaxListEntries)
    if maxEntries <= 0 || maxEntries > t.cfg.MaxListEntries {
        maxEntries = t.cfg.MaxListEntries
    }
    path, err := util.ResolveWithinBase(t.workspace, rel)
    if err != nil {
        return Result{}, err
    }
    var entries []string
    if recursive {
        err = filepath.WalkDir(path, func(current string, d fs.DirEntry, walkErr error) error {
            if walkErr != nil {
                return walkErr
            }
            if current == path {
                return nil
            }
            relative, _ := filepath.Rel(t.workspace, current)
            entries = append(entries, filepath.ToSlash(relative))
            if len(entries) >= maxEntries {
                return fs.SkipAll
            }
            return nil
        })
    } else {
        list, readErr := os.ReadDir(path)
        if readErr != nil {
            return Result{}, readErr
        }
        for _, entry := range list {
            relative, _ := filepath.Rel(t.workspace, filepath.Join(path, entry.Name()))
            entries = append(entries, filepath.ToSlash(relative))
            if len(entries) >= maxEntries {
                break
            }
        }
    }
    if err != nil && err != fs.SkipAll {
        return Result{}, err
    }
    sort.Strings(entries)
    return Result{OK: true, Summary: "files listed", Output: strings.Join(entries, "\n"), Files: entries}, nil
}
