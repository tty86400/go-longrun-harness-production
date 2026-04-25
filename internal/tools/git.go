package tools

import (
    "context"
    "fmt"
    "log/slog"
    "os/exec"
    "strings"

    "longrunharness/internal/util"
)

type gitStatusTool struct {
    workspace string
    logger    *slog.Logger
}

type gitCommitTool struct {
    workspace string
    logger    *slog.Logger
}

func NewGitStatusTool(workspace string, logger *slog.Logger) Tool {
    return &gitStatusTool{workspace: workspace, logger: logger}
}
func NewGitCommitTool(workspace string, logger *slog.Logger) Tool {
    return &gitCommitTool{workspace: workspace, logger: logger}
}

func (t *gitStatusTool) Name() string { return "git.status" }
func (t *gitStatusTool) Description() string { return "Return git status --short --branch for the workspace repository." }
func (t *gitStatusTool) JSONSchema() map[string]any { return map[string]any{"type": "object", "properties": map[string]any{}} }
func (t *gitStatusTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    _ = args
    cmd := exec.CommandContext(ctx, "git", "status", "--short", "--branch")
    cmd.Dir = t.workspace
    out, err := cmd.CombinedOutput()
    if err != nil {
        return Result{OK: false, Summary: "git status failed", Output: string(out), ExitCode: 1}, nil
    }
    return Result{OK: true, Summary: "git status", Output: string(out)}, nil
}

func (t *gitCommitTool) Name() string { return "git.commit" }
func (t *gitCommitTool) Description() string { return "Create a git commit in the workspace repository." }
func (t *gitCommitTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"message"}, "properties": map[string]any{"message": map[string]any{"type": "string"}, "add_all": map[string]any{"type": "boolean"}}}
}
func (t *gitCommitTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    message := getStringArg(args, "message")
    if strings.TrimSpace(message) == "" {
        return Result{}, fmt.Errorf("message is required")
    }
    if getBoolArg(args, "add_all", true) {
        addCmd := exec.CommandContext(ctx, "git", "add", "-A")
        addCmd.Dir = t.workspace
        if out, err := addCmd.CombinedOutput(); err != nil {
            return Result{OK: false, Summary: "git add failed", Output: string(out), ExitCode: 1}, nil
        }
    }
    commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
    commitCmd.Dir = t.workspace
    out, err := commitCmd.CombinedOutput()
    if err != nil {
        return Result{OK: false, Summary: "git commit failed", Output: string(out), ExitCode: 1}, nil
    }
    return Result{OK: true, Summary: "git commit created", Output: string(out), Metadata: map[string]any{"workspace": t.workspace}}, nil
}

var _ = util.EnsureDir
