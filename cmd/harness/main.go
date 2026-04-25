package main

import (
    "context"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "longrunharness/internal/config"
    "longrunharness/internal/harness"
    "longrunharness/internal/observability"
    "longrunharness/internal/provider"
    "longrunharness/internal/tools"
)

func main() {
    var (
        configPath string
        taskPath   string
        runID      string
        resumeID   string
        printCatalog bool
    )
    flag.StringVar(&configPath, "config", "", "path to JSON config")
    flag.StringVar(&taskPath, "task", "", "path to task markdown/text file")
    flag.StringVar(&runID, "run-id", "", "optional run ID for a new run")
    flag.StringVar(&resumeID, "resume-run", "", "resume an existing run ID")
    flag.BoolVar(&printCatalog, "print-catalog", false, "print the tool catalog and exit")
    flag.Parse()

    if configPath == "" {
        fmt.Fprintln(os.Stderr, "-config is required")
        os.Exit(2)
    }

    cfg, err := config.Load(configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "load config: %v\n", err)
        os.Exit(1)
    }
    if resumeID == "" {
        resumeID = cfg.Run.ResumeRunID
    }
    if resumeID == "" && taskPath == "" {
        fmt.Fprintln(os.Stderr, "-task is required unless -resume-run is provided")
        os.Exit(2)
    }

    objective := ""
    if taskPath != "" {
        data, err := os.ReadFile(taskPath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "read task: %v\n", err)
            os.Exit(1)
        }
        objective = string(data)
    }

    var session *harness.Session
    var state *harness.State
    if resumeID != "" {
        session, state, err = harness.ResumeSession(cfg.Run, resumeID)
    } else {
        session, state, err = harness.NewSession(cfg.Run, runID, objective)
    }
    if err != nil {
        fmt.Fprintf(os.Stderr, "open session: %v\n", err)
        os.Exit(1)
    }
    defer session.Close()

    logHandle, err := observability.NewLogger(cfg.Observability.JSONLogs, session.RunDir+"/run.log")
    if err != nil {
        fmt.Fprintf(os.Stderr, "create logger: %v\n", err)
        os.Exit(1)
    }
    defer logHandle.Close()
    logger := logHandle.Logger.With("run_id", session.RunID)

    registry, err := tools.NewRegistry(cfg.Tools, cfg.Run.Workspace, logger)
    if err != nil {
        logger.Error("create tool registry failed", "error", err)
        os.Exit(1)
    }
    if printCatalog {
        fmt.Println(registry.CatalogJSON())
        return
    }

    actorProvider, err := provider.New(cfg.Actor)
    if err != nil {
        logger.Error("create actor provider failed", "error", err)
        os.Exit(1)
    }
    var reviewerProvider provider.Provider
    if cfg.Reviewer != nil {
        reviewerProvider, err = provider.New(*cfg.Reviewer)
        if err != nil {
            logger.Error("create reviewer provider failed", "error", err)
            os.Exit(1)
        }
    }
    var summarizerProvider provider.Provider
    if cfg.Summarizer != nil {
        summarizerProvider, err = provider.New(*cfg.Summarizer)
        if err != nil {
            logger.Error("create summarizer provider failed", "error", err)
            os.Exit(1)
        }
    }

    metrics := observability.NewMetrics(observability.RunNamespace(session.RunID))
    stopServer, err := observability.StartServer(cfg.Observability.MetricsAddr, cfg.Observability.EnablePprof, logger)
    if err != nil {
        logger.Error("start observability server failed", "error", err)
        os.Exit(1)
    }
    defer func() {
        _ = stopServer(context.Background())
    }()

    eng := harness.NewEngine(cfg, harness.Providers{Actor: actorProvider, Reviewer: reviewerProvider, Summarizer: summarizerProvider}, registry, session, logger, metrics)

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    result, err := eng.Run(ctx, state)
    if err != nil {
        logger.Error("run failed", "error", err)
        os.Exit(1)
    }
    logger.Info("run complete", "status", result.State.Status, "step", result.State.Step)
    fmt.Printf("run_dir=%s\nstatus=%s\n", session.RunDir, result.State.Status)
}

var _ = slog.LevelInfo
