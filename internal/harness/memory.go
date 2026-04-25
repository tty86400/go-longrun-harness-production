package harness

import "strings"

func shouldSummarize(state *State, triggerTokens int, everyNSteps int) bool {
    if everyNSteps > 0 && state.Step > 0 && state.Step%everyNSteps == 0 && state.Step != state.LastSummaryAtStep {
        return true
    }
    return estimateStateTokens(state) >= triggerTokens
}

func estimateStateTokens(state *State) int {
    var b strings.Builder
    b.WriteString(state.Objective)
    b.WriteString("\n")
    b.WriteString(state.RollingSummary)
    b.WriteString("\n")
    for _, item := range state.StickyFacts {
        b.WriteString(item)
        b.WriteString("\n")
    }
    for _, item := range state.Plan {
        b.WriteString(item)
        b.WriteString("\n")
    }
    for _, item := range state.Todo {
        b.WriteString(item)
        b.WriteString("\n")
    }
    for _, evt := range state.RecentEvents {
        b.WriteString(evt.Content)
        b.WriteString("\n")
    }
    runeCount := len([]rune(b.String()))
    if runeCount == 0 {
        return 0
    }
    return runeCount/4 + 1
}

func mergeUnique(items []string) []string {
    seen := map[string]struct{}{}
    out := make([]string, 0, len(items))
    for _, item := range items {
        cleaned := strings.TrimSpace(item)
        if cleaned == "" {
            continue
        }
        if _, ok := seen[cleaned]; ok {
            continue
        }
        seen[cleaned] = struct{}{}
        out = append(out, cleaned)
    }
    return out
}

func mergeText(parts ...string) string {
    merged := mergeUnique(parts)
    return strings.Join(merged, "\n")
}
