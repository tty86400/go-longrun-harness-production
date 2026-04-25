package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "sort"

    "log/slog"

    "longrunharness/internal/config"
)

type Registry struct {
    tools map[string]Tool
}

func NewRegistry(cfg config.ToolsConfig, workspace string, logger *slog.Logger) (*Registry, error) {
    r := &Registry{tools: map[string]Tool{}}
    if cfg.Files.Enabled {
        r.Register(NewFilesReadTool(cfg.Files, workspace))
        r.Register(NewFilesWriteTool(cfg.Files, workspace))
        r.Register(NewFilesAppendTool(cfg.Files, workspace))
        r.Register(NewFilesListTool(cfg.Files, workspace))
    }
    if cfg.Shell.Enabled {
        r.Register(NewShellExecTool(cfg.Shell, workspace, logger))
    }
    if cfg.Git.Enabled {
        r.Register(NewGitStatusTool(workspace, logger))
        r.Register(NewGitCommitTool(workspace, logger))
    }
    if cfg.HTTP.Enabled {
        r.Register(NewHTTPFetchTool(cfg.HTTP))
    }
    if cfg.Benchmark.Enabled {
        r.Register(NewBenchmarkTool(cfg.Benchmark, workspace, logger))
    }
    return r, nil
}

func (r *Registry) Register(tool Tool) {
    if r.tools == nil {
        r.tools = map[string]Tool{}
    }
    r.tools[tool.Name()] = tool
}

func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (Result, error) {
    tool, ok := r.tools[name]
    if !ok {
        return Result{}, fmt.Errorf("unknown tool: %s", name)
    }
    return tool.Execute(ctx, args)
}

func (r *Registry) Exists(name string) bool {
    _, ok := r.tools[name]
    return ok
}

func (r *Registry) Names() []string {
    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}

func (r *Registry) CatalogJSON() string {
    items := make([]map[string]any, 0, len(r.tools))
    for _, name := range r.Names() {
        t := r.tools[name]
        items = append(items, map[string]any{
            "name":        t.Name(),
            "description": t.Description(),
            "json_schema": t.JSONSchema(),
        })
    }
    data, _ := json.MarshalIndent(items, "", "  ")
    return string(data)
}
