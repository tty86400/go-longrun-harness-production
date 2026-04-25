package tools

import (
    "context"
    "fmt"
    "log/slog"
    "os/exec"
    "time"

    "longrunharness/internal/config"
)

type benchmarkTool struct {
    cfg       config.BenchmarkConfig
    workspace string
    logger    *slog.Logger
}

func NewBenchmarkTool(cfg config.BenchmarkConfig, workspace string, logger *slog.Logger) Tool {
    return &benchmarkTool{cfg: cfg, workspace: workspace, logger: logger}
}

func (t *benchmarkTool) Name() string { return "benchmark.run" }
func (t *benchmarkTool) Description() string { return "Run a preconfigured benchmark script by name. The script itself is configured by the operator, not the model." }
func (t *benchmarkTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"name"}, "properties": map[string]any{"name": map[string]any{"type": "string"}, "timeout_seconds": map[string]any{"type": "integer"}}}
}
func (t *benchmarkTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    name := getStringArg(args, "name")
    script, ok := t.cfg.Scripts[name]
    if !ok {
        return Result{}, fmt.Errorf("unknown benchmark script %q", name)
    }
    timeout := getIntArg(args, "timeout_seconds", t.cfg.TimeoutSeconds)
    runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
    defer cancel()

    stdout := newLimitedBuffer(t.cfg.MaxOutputBytes)
    stderr := newLimitedBuffer(t.cfg.MaxOutputBytes)
    cmd := exec.CommandContext(runCtx, "sh", "-lc", script)
    cmd.Dir = t.workspace
    cmd.Stdout = stdout
    cmd.Stderr = stderr
    t.logger.Info("tool benchmark.run", "name", name)
    err := cmd.Run()
    output, truncated := combineOutput(stdout, stderr)
    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else {
            exitCode = 1
        }
    }
    return Result{OK: err == nil, Summary: "benchmark executed", Output: output, ExitCode: exitCode, Truncated: truncated, Metadata: map[string]any{"name": name}}, nil
}
