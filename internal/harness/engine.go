package harness

import (
    "context"
    "fmt"
    "log/slog"
    "path/filepath"
    "strings"
    "time"

    "longrunharness/internal/config"
    "longrunharness/internal/observability"
    "longrunharness/internal/provider"
    "longrunharness/internal/tools"
)

type Providers struct {
    Actor      provider.Provider
    Reviewer   provider.Provider
    Summarizer provider.Provider
}

type Engine struct {
    cfg      *config.Config
    providers Providers
    tools    *tools.Registry
    session  *Session
    logger   *slog.Logger
    metrics  *observability.Metrics
}

func NewEngine(cfg *config.Config, providers Providers, registry *tools.Registry, session *Session, logger *slog.Logger, metrics *observability.Metrics) *Engine {
    return &Engine{
        cfg:      cfg,
        providers: providers,
        tools:    registry,
        session:  session,
        logger:   logger,
        metrics:  metrics,
    }
}

func (e *Engine) Run(ctx context.Context, state *State) (*RunResult, error) {
    if state == nil {
        return nil, fmt.Errorf("state is nil")
    }
    deadline := state.StartedAt.Add(e.cfg.Loop.MaxWallClockDuration())
    if state.Step == 0 && len(state.RecentEvents) == 0 {
        _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: 0, Kind: "objective", Actor: "system", Content: state.Objective})
    }

    startStep := state.Step + 1
    for step := startStep; step <= e.cfg.Loop.MaxSteps; step++ {
        if ctx.Err() != nil {
            state.Status = "cancelled"
            state.LastError = ctx.Err().Error()
            break
        }
        if time.Now().After(deadline) {
            state.Status = "stopped"
            state.LastError = "max wall clock duration reached"
            break
        }

        state.Step = step
        state.UpdatedAt = time.Now().UTC()
        state.Stats.Steps = step
        e.metrics.Steps.Add(1)
        e.logger.Info("step started", "step", step)

        stepCtx, cancel := context.WithTimeout(ctx, time.Duration(e.cfg.Loop.StepTimeoutSeconds)*time.Second)
        stepFailed := false

        decision, rawDecision, err := e.askActor(stepCtx, state)
        if err != nil {
            stepFailed = true
            state.ConsecutiveFailures++
            state.LastError = err.Error()
            _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "actor_error", Actor: "actor", Content: err.Error()})
            cancel()
            if err := e.persistStep(state); err != nil {
                return nil, err
            }
            if state.ConsecutiveFailures >= e.cfg.Loop.MaxConsecutiveFailures {
                state.Status = "failed"
                break
            }
            continue
        }

        if len(decision.UpdatedPlan) > 0 {
            state.Plan = mergeUnique(decision.UpdatedPlan)
        }
        if len(decision.UpdatedTodo) > 0 {
            state.Todo = mergeUnique(decision.UpdatedTodo)
        }
        state.RollingSummary = mergeText(state.RollingSummary, decision.SituationAssessment)
        _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "actor_decision", Actor: "actor", Content: rawDecision})

        if decision.Done {
            state.Completed = true
            state.FinalAnswer = strings.TrimSpace(decision.FinalAnswer)
            state.Status = "completed"
            cancel()
            break
        }

        actions := decision.Actions
        if len(actions) > e.cfg.Loop.MaxActionsPerStep {
            actions = actions[:e.cfg.Loop.MaxActionsPerStep]
        }
        if len(actions) == 0 {
            stepFailed = true
            state.LastError = "actor returned zero actions without done=true"
            _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "noop", Actor: "actor", Content: state.LastError})
        }

        for i, action := range actions {
            result, artifact, execErr := e.executeAction(stepCtx, state, i+1, action)
            if execErr != nil {
                stepFailed = true
                state.LastError = execErr.Error()
                _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "tool_error", Actor: "tool", Tool: action.Tool, Title: action.Reason, Content: execErr.Error()})
                continue
            }
            if !result.OK {
                stepFailed = true
                state.Stats.ToolFailures++
                e.metrics.ToolFailures.Add(1)
            }
            _ = e.addEvent(state, Event{
                Timestamp: time.Now().UTC(),
                Step:      step,
                Kind:      "tool_result",
                Actor:     "tool",
                Tool:      action.Tool,
                OK:        result.OK,
                Title:     action.Reason,
                Content:   renderToolObservation(result),
                Artifact:  artifact,
            })
        }

        if e.cfg.Review.EveryNSteps > 0 && step%e.cfg.Review.EveryNSteps == 0 {
            review, rawReview, reviewErr := e.askReviewer(stepCtx, state)
            if reviewErr != nil {
                stepFailed = true
                state.LastError = reviewErr.Error()
                _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "review_error", Actor: "reviewer", Content: reviewErr.Error()})
            } else {
                state.Stats.Reviews++
                e.metrics.Reviews.Add(1)
                if len(review.RevisedPlan) > 0 {
                    state.Plan = mergeUnique(review.RevisedPlan)
                }
                if len(review.UpdatedTodo) > 0 {
                    state.Todo = mergeUnique(review.UpdatedTodo)
                }
                if strings.TrimSpace(review.NextPriority) != "" {
                    state.Todo = mergeUnique(append([]string{review.NextPriority}, state.Todo...))
                }
                state.RollingSummary = mergeText(state.RollingSummary, review.ProgressAssessment)
                _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "review", Actor: "reviewer", Content: rawReview})
                if review.ShouldSummarize {
                    if err := e.summarize(stepCtx, state); err != nil {
                        _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "summary_error", Actor: "summarizer", Content: err.Error()})
                    }
                }
            }
        }

        if shouldSummarize(state, e.cfg.Memory.SummaryTriggerTokens, e.cfg.Memory.SummarizeEveryNSteps) {
            if err := e.summarize(stepCtx, state); err != nil {
                _ = e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: step, Kind: "summary_error", Actor: "summarizer", Content: err.Error()})
            }
        }

        cancel()

        if stepFailed {
            state.ConsecutiveFailures++
        } else {
            state.ConsecutiveFailures = 0
        }

        if err := e.persistStep(state); err != nil {
            return nil, err
        }
        if state.ConsecutiveFailures >= e.cfg.Loop.MaxConsecutiveFailures {
            state.Status = "failed"
            break
        }
    }

    if state.Completed {
        state.Status = "completed"
    } else if state.Status == "" {
        if ctx.Err() != nil {
            state.Status = "cancelled"
            state.LastError = ctx.Err().Error()
        } else if state.ConsecutiveFailures >= e.cfg.Loop.MaxConsecutiveFailures {
            state.Status = "failed"
        } else {
            state.Status = "stopped"
        }
    }
    if strings.TrimSpace(state.FinalAnswer) == "" {
        switch state.Status {
        case "completed":
            state.FinalAnswer = "Task completed, but the actor did not provide a final answer. Inspect the run report and transcript for details."
        case "cancelled":
            state.FinalAnswer = "Run cancelled before completion. Resume from the saved checkpoint if needed."
        case "failed":
            state.FinalAnswer = "Run stopped after repeated failures. Inspect the transcript, checkpoints, and raw prompts to diagnose the issue."
        default:
            state.FinalAnswer = "Run stopped before an explicit completion signal. Inspect the report and artifacts for the latest known state."
        }
    }
    state.EndedAt = time.Now().UTC()
    state.UpdatedAt = state.EndedAt

    if err := e.persistStep(state); err != nil {
        return nil, err
    }
    report := e.buildRunReport(state)
    if _, err := e.session.WriteRunReport(report); err != nil {
        return nil, err
    }
    e.logger.Info("run finished", "status", state.Status, "step", state.Step)
    return &RunResult{State: state}, nil
}

func (e *Engine) persistStep(state *State) error {
    state.Stats.Checkpoints++
    if err := e.session.SaveState(state); err != nil {
        return err
    }
    if err := e.session.SaveCheckpoint(state); err != nil {
        return err
    }
    return nil
}

func (e *Engine) askActor(ctx context.Context, state *State) (ActorDecision, string, error) {
    messages := []provider.Message{
        {Role: "system", Content: buildActorSystemPrompt(e.tools.CatalogJSON(), e.cfg.Loop.MaxActionsPerStep, e.cfg.Prompt.Language)},
        {Role: "user", Content: buildActorUserPrompt(state, e.cfg.Loop.MaxSteps)},
    }
    decision, raw, err := generateJSON[ActorDecision](ctx, state.Step, "actor", e.providers.Actor, e.cfg.Actor, messages, e.session, e.cfg.Run, e.metrics)
    if err != nil {
        return ActorDecision{}, raw, err
    }
    if err := validateDecision(decision, e.tools, e.cfg.Loop.MaxActionsPerStep); err != nil {
        return ActorDecision{}, raw, err
    }
    state.Stats.ActorCalls++
    return decision, raw, nil
}

func (e *Engine) askReviewer(ctx context.Context, state *State) (ReviewReport, string, error) {
    providerCfg := e.cfg.Actor
    currentProvider := e.providers.Actor
    if e.providers.Reviewer != nil && e.cfg.Reviewer != nil {
        providerCfg = *e.cfg.Reviewer
        currentProvider = e.providers.Reviewer
    }
    messages := []provider.Message{
        {Role: "system", Content: buildReviewerSystemPrompt(e.cfg.Prompt.Language)},
        {Role: "user", Content: buildReviewerUserPrompt(state)},
    }
    report, raw, err := generateJSON[ReviewReport](ctx, state.Step, "reviewer", currentProvider, providerCfg, messages, e.session, e.cfg.Run, e.metrics)
    if err == nil {
        state.Stats.ReviewerCalls++
    }
    return report, raw, err
}

func (e *Engine) summarize(ctx context.Context, state *State) error {
    providerCfg := e.cfg.Actor
    currentProvider := e.providers.Actor
    if e.providers.Summarizer != nil && e.cfg.Summarizer != nil {
        providerCfg = *e.cfg.Summarizer
        currentProvider = e.providers.Summarizer
    } else if e.providers.Reviewer != nil && e.cfg.Reviewer != nil {
        providerCfg = *e.cfg.Reviewer
        currentProvider = e.providers.Reviewer
    }
    messages := []provider.Message{
        {Role: "system", Content: buildSummarizerSystemPrompt(e.cfg.Prompt.Language)},
        {Role: "user", Content: buildSummarizerUserPrompt(state)},
    }
    report, raw, err := generateJSON[SummaryReport](ctx, state.Step, "summarizer", currentProvider, providerCfg, messages, e.session, e.cfg.Run, e.metrics)
    if err != nil {
        return err
    }
    state.Stats.SummarizerCalls++
    state.Stats.Summaries++
    e.metrics.Summaries.Add(1)
    if strings.TrimSpace(report.RollingSummary) != "" {
        state.RollingSummary = strings.TrimSpace(report.RollingSummary)
    }
    if len(report.StickyFacts) > 0 {
        state.StickyFacts = mergeUnique(append(state.StickyFacts, report.StickyFacts...))
    }
    if len(report.CarryOverPlan) > 0 {
        state.Plan = mergeUnique(report.CarryOverPlan)
    }
    if len(report.CarryOverTodo) > 0 {
        state.Todo = mergeUnique(report.CarryOverTodo)
    }
    if len(state.RecentEvents) > e.cfg.Memory.RecentEvents/2 {
        state.RecentEvents = append([]Event(nil), state.RecentEvents[len(state.RecentEvents)-e.cfg.Memory.RecentEvents/2:]...)
    }
    state.MemoryCompactions++
    state.LastSummaryAtStep = state.Step
    return e.addEvent(state, Event{Timestamp: time.Now().UTC(), Step: state.Step, Kind: "summary", Actor: "summarizer", Content: raw})
}

func generateJSON[T any](ctx context.Context, step int, role string, p provider.Provider, cfg config.ProviderConfig, messages []provider.Message, session *Session, runCfg config.RunConfig, metrics *observability.Metrics) (T, string, error) {
    var zero T
    req := provider.ChatRequest{
        Messages:        messages,
        Temperature:     cfg.Temperature,
        MaxOutputTokens: cfg.MaxOutputTokens,
        ForceJSON:       true,
    }
    if metrics != nil {
        metrics.ProviderCalls.Add(1)
    }
    response, err := p.Generate(ctx, req)
    if (runCfg.PersistPrompts || runCfg.PersistProviderIO) && session != nil {
        _ = session.SavePromptAudit(step, role, messages, response, response.Text, err)
    }
    if err != nil {
        if metrics != nil {
            metrics.ProviderErrors.Add(1)
        }
        return zero, response.Text, err
    }
    parsed, raw, err := decodeJSONStrict[T](response.Text)
    if err != nil {
        if metrics != nil {
            metrics.ProviderErrors.Add(1)
        }
        return zero, response.Text, err
    }
    return parsed, raw, nil
}

func (e *Engine) executeAction(ctx context.Context, state *State, actionIndex int, action Action) (tools.Result, string, error) {
    execCtx := ctx
    cancel := func() {}
    if action.TimeoutSeconds > 0 {
        execCtx, cancel = context.WithTimeout(ctx, time.Duration(action.TimeoutSeconds)*time.Second)
    }
    defer cancel()

    started := time.Now()
    result, err := e.tools.Execute(execCtx, action.Tool, action.Args)
    duration := time.Since(started)
    e.logger.Info("tool executed", "step", state.Step, "tool", action.Tool, "ok", result.OK, "duration_ms", duration.Milliseconds())
    state.Stats.ToolCalls++
    e.metrics.ToolCalls.Add(1)
    if err != nil {
        return tools.Result{}, "", err
    }
    artifact, artErr := e.session.SaveArtifact(state.Step, actionIndex, strings.ReplaceAll(action.Tool, ".", "-"), map[string]any{
        "action":   action,
        "result":   result,
        "duration_ms": duration.Milliseconds(),
    })
    if artErr != nil {
        return tools.Result{}, "", artErr
    }
    return result, artifact, nil
}

func (e *Engine) addEvent(state *State, event Event) error {
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now().UTC()
    }
    state.RecentEvents = append(state.RecentEvents, event)
    if len(state.RecentEvents) > e.cfg.Memory.RecentEvents {
        state.RecentEvents = append([]Event(nil), state.RecentEvents[len(state.RecentEvents)-e.cfg.Memory.RecentEvents:]...)
    }
    return e.session.AppendEvent(event)
}

func renderToolObservation(result tools.Result) string {
    var b strings.Builder
    if strings.TrimSpace(result.Summary) != "" {
        b.WriteString(result.Summary)
    }
    if result.Output != "" {
        if b.Len() > 0 {
            b.WriteString("\n")
        }
        content := result.Output
        if len(content) > 3000 {
            content = content[:3000] + "…"
        }
        b.WriteString(content)
    }
    return strings.TrimSpace(b.String())
}

func (e *Engine) buildRunReport(state *State) string {
    duration := state.UpdatedAt.Sub(state.StartedAt).Round(time.Second)
    var b strings.Builder
    b.WriteString("# Long-Run Harness Report\n\n")
    b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", state.RunID))
    b.WriteString(fmt.Sprintf("- Status: `%s`\n", state.Status))
    b.WriteString(fmt.Sprintf("- Started: `%s`\n", state.StartedAt.Format(time.RFC3339)))
    if !state.EndedAt.IsZero() {
        b.WriteString(fmt.Sprintf("- Ended: `%s`\n", state.EndedAt.Format(time.RFC3339)))
    }
    b.WriteString(fmt.Sprintf("- Duration: `%s`\n", duration))
    b.WriteString(fmt.Sprintf("- Final step: `%d`\n", state.Step))
    b.WriteString(fmt.Sprintf("- Resume count: `%d`\n", state.ResumeCount))
    b.WriteString("\n## Objective\n\n")
    b.WriteString(state.Objective)
    b.WriteString("\n\n## Final Answer\n\n")
    b.WriteString(state.FinalAnswer)
    b.WriteString("\n\n## Rolling Summary\n\n")
    b.WriteString(blankToDash(state.RollingSummary))
    b.WriteString("\n\n## Sticky Facts\n\n")
    b.WriteString(numberedList(state.StickyFacts))
    b.WriteString("\n\n## Current Plan\n\n")
    b.WriteString(numberedList(state.Plan))
    b.WriteString("\n\n## Current TODO\n\n")
    b.WriteString(numberedList(state.Todo))
    b.WriteString("\n\n## Stats\n\n")
    b.WriteString(fmt.Sprintf("- Steps: %d\n", state.Stats.Steps))
    b.WriteString(fmt.Sprintf("- Actor calls: %d\n", state.Stats.ActorCalls))
    b.WriteString(fmt.Sprintf("- Reviewer calls: %d\n", state.Stats.ReviewerCalls))
    b.WriteString(fmt.Sprintf("- Summarizer calls: %d\n", state.Stats.SummarizerCalls))
    b.WriteString(fmt.Sprintf("- Tool calls: %d\n", state.Stats.ToolCalls))
    b.WriteString(fmt.Sprintf("- Tool failures: %d\n", state.Stats.ToolFailures))
    b.WriteString(fmt.Sprintf("- Reviews: %d\n", state.Stats.Reviews))
    b.WriteString(fmt.Sprintf("- Summaries: %d\n", state.Stats.Summaries))
    b.WriteString(fmt.Sprintf("- Checkpoints: %d\n", state.Stats.Checkpoints))
    b.WriteString("\n## Recent Events\n\n")
    if len(state.RecentEvents) == 0 {
        b.WriteString("-\n")
    } else {
        for _, evt := range state.RecentEvents {
            b.WriteString(fmt.Sprintf("- [%s] step=%d kind=%s", evt.Timestamp.Format(time.RFC3339), evt.Step, evt.Kind))
            if evt.Actor != "" {
                b.WriteString(fmt.Sprintf(" actor=%s", evt.Actor))
            }
            if evt.Tool != "" {
                b.WriteString(fmt.Sprintf(" tool=%s", evt.Tool))
            }
            if evt.Artifact != "" {
                b.WriteString(fmt.Sprintf(" artifact=%s", filepath.ToSlash(evt.Artifact)))
            }
            b.WriteString("\n")
        }
    }
    b.WriteString("\n## Paths\n\n")
    b.WriteString(fmt.Sprintf("- Run directory: `%s`\n", e.session.RunDir))
    b.WriteString(fmt.Sprintf("- Workspace: `%s`\n", e.session.Workspace))
    b.WriteString(fmt.Sprintf("- State file: `%s`\n", filepath.Join(e.session.RunDir, "state.json")))
    b.WriteString(fmt.Sprintf("- Transcript: `%s`\n", filepath.Join(e.session.RunDir, "transcript.jsonl")))
    return b.String()
}
