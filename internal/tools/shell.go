package tools

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "longrunharness/internal/config"
    "longrunharness/internal/util"
)

type shellExecTool struct {
    cfg       config.ShellConfig
    workspace string
    logger    *slog.Logger
}

func NewShellExecTool(cfg config.ShellConfig, workspace string, logger *slog.Logger) Tool {
    return &shellExecTool{cfg: cfg, workspace: workspace, logger: logger}
}

func (t *shellExecTool) Name() string { return "shell.exec" }
func (t *shellExecTool) Description() string {
    return "Execute an argv-style command in the sandboxed workspace. There is no intermediate shell parser."
}
func (t *shellExecTool) JSONSchema() map[string]any {
    return map[string]any{
        "type":     "object",
        "required": []string{"argv"},
        "properties": map[string]any{
            "argv":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
            "workdir":         map[string]any{"type": "string"},
            "timeout_seconds": map[string]any{"type": "integer"},
        },
    }
}
func (t *shellExecTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    argv := getStringSliceArg(args, "argv")
    if len(argv) == 0 {
        return Result{}, fmt.Errorf("argv is required")
    }
    if !t.commandAllowed(argv[0]) {
        return Result{}, fmt.Errorf("command %q is not allowed", argv[0])
    }

    workdir := getStringArg(args, "workdir")
    if workdir == "" {
        workdir = "."
    }
    resolvedDir, err := util.ResolveWithinBase(t.workspace, workdir)
    if err != nil {
        return Result{}, err
    }
    if stat, err := os.Stat(resolvedDir); err != nil || !stat.IsDir() {
        return Result{}, fmt.Errorf("workdir is not a directory: %s", workdir)
    }

    timeoutSeconds := getIntArg(args, "timeout_seconds", t.cfg.DefaultTimeoutSeconds)
    if timeoutSeconds <= 0 {
        timeoutSeconds = t.cfg.DefaultTimeoutSeconds
    }
    runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
    defer cancel()

    cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
    cmd.Dir = resolvedDir
    cmd.Env = t.allowedEnv()

    stdout := newLimitedBuffer(t.cfg.MaxOutputBytes)
    stderr := newLimitedBuffer(t.cfg.MaxOutputBytes)
    cmd.Stdout = stdout
    cmd.Stderr = stderr

    t.logger.Info("tool shell.exec", "argv", argv, "workdir", filepath.ToSlash(workdir))
    err = cmd.Run()
    output, truncated := combineOutput(stdout, stderr)
    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else if runCtx.Err() != nil {
            exitCode = 124
        } else {
            return Result{}, err
        }
    }
    return Result{
        OK:        err == nil,
        Summary:   "command executed",
        Output:    output,
        ExitCode:  exitCode,
        Truncated: truncated,
        Metadata:  map[string]any{"argv": argv, "workdir": filepath.ToSlash(workdir)},
    }, nil
}

func (t *shellExecTool) commandAllowed(command string) bool {
    lower := strings.ToLower(strings.TrimSpace(command))
    for _, denied := range t.cfg.DeniedCommands {
        if lower == strings.ToLower(strings.TrimSpace(denied)) {
            return false
        }
    }
    if len(t.cfg.AllowedCommands) == 0 {
        return true
    }
    for _, allowed := range t.cfg.AllowedCommands {
        if lower == strings.ToLower(strings.TrimSpace(allowed)) {
            return true
        }
    }
    return false
}

func (t *shellExecTool) allowedEnv() []string {
    if len(t.cfg.EnvAllowlist) == 0 {
        return []string{}
    }
    allowed := map[string]struct{}{}
    for _, key := range t.cfg.EnvAllowlist {
        allowed[strings.TrimSpace(key)] = struct{}{}
    }
    out := []string{}
    for _, entry := range os.Environ() {
        parts := strings.SplitN(entry, "=", 2)
        if len(parts) != 2 {
            continue
        }
        if _, ok := allowed[parts[0]]; ok {
            out = append(out, entry)
        }
    }
    return out
}
