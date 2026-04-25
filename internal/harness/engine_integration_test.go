package harness

import (
    "context"
    "io"
    "log/slog"
    "os"
    "path/filepath"
    "testing"

    "longrunharness/internal/config"
    "longrunharness/internal/observability"
    "longrunharness/internal/provider"
    "longrunharness/internal/tools"
)

func TestEngineRunWithMockProvider(t *testing.T) {
    root := t.TempDir()
    cfg := &config.Config{
        Run: config.RunConfig{
            IDPrefix:          "itest",
            Workspace:         filepath.Join(root, "workspace"),
            RunsDir:           filepath.Join(root, "runs"),
            PersistPrompts:    true,
            PersistProviderIO: true,
        },
        Prompt: config.PromptConfig{Language: "en"},
        Loop: config.LoopConfig{
            MaxSteps:               5,
            MaxActionsPerStep:      2,
            StepTimeoutSeconds:     30,
            MaxWallClockMinutes:    10,
            MaxConsecutiveFailures: 3,
        },
        Review: config.ReviewConfig{EveryNSteps: 2},
        Memory: config.MemoryConfig{RecentEvents: 12, EstimatedPromptBudget: 12000, SummaryTriggerTokens: 8000, SummarizeEveryNSteps: 4},
        Actor:  config.ProviderConfig{Kind: "mock", Name: "actor"},
        Reviewer: &config.ProviderConfig{Kind: "mock", Name: "reviewer"},
        Summarizer: &config.ProviderConfig{Kind: "mock", Name: "summarizer"},
        Tools: config.ToolsConfig{Files: config.FilesConfig{Enabled: true, MaxReadBytes: 1 << 20, MaxWriteBytes: 1 << 20, MaxListEntries: 100}},
    }

    session, state, err := NewSession(cfg.Run, "", "Create and verify a demo artifact.")
    if err != nil {
        t.Fatalf("new session: %v", err)
    }
    defer session.Close()

    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    registry, err := tools.NewRegistry(cfg.Tools, cfg.Run.Workspace, logger)
    if err != nil {
        t.Fatalf("new registry: %v", err)
    }
    actor, _ := provider.New(cfg.Actor)
    reviewer, _ := provider.New(*cfg.Reviewer)
    summarizer, _ := provider.New(*cfg.Summarizer)
    metrics := observability.NewMetrics("itest")

    eng := NewEngine(cfg, Providers{Actor: actor, Reviewer: reviewer, Summarizer: summarizer}, registry, session, logger, metrics)
    result, err := eng.Run(context.Background(), state)
    if err != nil {
        t.Fatalf("run: %v", err)
    }
    if result.State.Status != "completed" {
        t.Fatalf("unexpected status: %s", result.State.Status)
    }
    data, err := os.ReadFile(filepath.Join(cfg.Run.Workspace, "demo.txt"))
    if err != nil {
        t.Fatalf("read demo file: %v", err)
    }
    if string(data) == "" {
        t.Fatalf("demo file should not be empty")
    }
    if _, err := os.Stat(filepath.Join(session.RunDir, "RUN_REPORT.md")); err != nil {
        t.Fatalf("expected run report: %v", err)
    }
}
