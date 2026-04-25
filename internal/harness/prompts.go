package harness

import (
    "fmt"
    "strings"
)

func buildActorSystemPrompt(toolCatalog string, maxActions int, language string) string {
    return fmt.Sprintf(`[ROLE:ACTOR]
You are the execution brain of a long-horizon engineering harness.
Operate like a careful production engineer, not a one-shot chatbot.

Return exactly one JSON object and nothing else.

JSON schema:
{
  "situation_assessment": "string",
  "updated_plan": ["string"],
  "updated_todo": ["string"],
  "actions": [
    {
      "tool": "tool.name",
      "args": {},
      "reason": "why this step is highest leverage",
      "timeout_seconds": 60
    }
  ],
  "done": false,
  "final_answer": ""
}

Rules:
- Use only tool names from the catalog.
- Never invent tool results.
- Prefer inspect -> change -> test -> verify loops.
- Make small, auditable, reversible steps.
- Trust the workspace and run artifacts more than prompt history.
- Use at most %d actions.
- If the objective is complete, set done=true, actions=[], and provide final_answer.
- Free-text values may be written in %s, but JSON keys must stay exactly as shown.

Available tools:
%s
`, maxActions, language, toolCatalog)
}

func buildActorUserPrompt(state *State, maxSteps int) string {
    return fmt.Sprintf(`Objective:
%s

Current step:
%d / %d

Rolling summary:
%s

Sticky facts:
%s

Current plan:
%s

Current todo:
%s

Recent transcript:
%s

Return one JSON object only.
`, state.Objective, state.Step, maxSteps, blankToDash(state.RollingSummary), numberedList(state.StickyFacts), numberedList(state.Plan), numberedList(state.Todo), renderEvents(state.RecentEvents))
}

func buildReviewerSystemPrompt(language string) string {
    return fmt.Sprintf(`[ROLE:REVIEWER]
You are the self-critique loop of a long-horizon harness.
Your job is to detect drift, missing evidence, weak verification, and the single highest-leverage next step.

Return exactly one JSON object and nothing else.

JSON schema:
{
  "progress_assessment": "string",
  "done_items": ["string"],
  "missing_items": ["string"],
  "risks": ["string"],
  "next_priority": "string",
  "revised_plan": ["string"],
  "updated_todo": ["string"],
  "should_summarize": false
}

Free-text values may be written in %s.
`, language)
}

func buildReviewerUserPrompt(state *State) string {
    return fmt.Sprintf(`Objective:
%s

Rolling summary:
%s

Current plan:
%s

Current todo:
%s

Recent transcript:
%s

Return one JSON object only.
`, state.Objective, blankToDash(state.RollingSummary), numberedList(state.Plan), numberedList(state.Todo), renderEvents(state.RecentEvents))
}

func buildSummarizerSystemPrompt(language string) string {
    return fmt.Sprintf(`[ROLE:SUMMARIZER]
You compress long history into a future-facing state summary.
Keep facts that matter. Drop chatter that does not matter.

Return exactly one JSON object and nothing else.

JSON schema:
{
  "rolling_summary": "string",
  "sticky_facts": ["string"],
  "carry_over_plan": ["string"],
  "carry_over_todo": ["string"]
}

Free-text values may be written in %s.
`, language)
}

func buildSummarizerUserPrompt(state *State) string {
    return fmt.Sprintf(`Objective:
%s

Rolling summary so far:
%s

Sticky facts:
%s

Current plan:
%s

Current todo:
%s

Recent transcript:
%s

Return one JSON object only.
`, state.Objective, blankToDash(state.RollingSummary), numberedList(state.StickyFacts), numberedList(state.Plan), numberedList(state.Todo), renderEvents(state.RecentEvents))
}

func numberedList(items []string) string {
    if len(items) == 0 {
        return "-"
    }
    var b strings.Builder
    for i, item := range items {
        b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(item)))
    }
    return strings.TrimSpace(b.String())
}

func blankToDash(s string) string {
    if strings.TrimSpace(s) == "" {
        return "-"
    }
    return s
}

func renderEvents(events []Event) string {
    if len(events) == 0 {
        return "-"
    }
    var b strings.Builder
    for _, evt := range events {
        line := fmt.Sprintf("[%s step=%d kind=%s", evt.Timestamp.Format("2006-01-02 15:04:05"), evt.Step, evt.Kind)
        if evt.Actor != "" {
            line += " actor=" + evt.Actor
        }
        if evt.Tool != "" {
            line += " tool=" + evt.Tool
        }
        if evt.Title != "" {
            line += " title=" + evt.Title
        }
        line += "] "
        content := strings.TrimSpace(evt.Content)
        if len(content) > 600 {
            content = content[:600] + "…"
        }
        line += content
        b.WriteString(line)
        b.WriteString("\n")
    }
    return strings.TrimSpace(b.String())
}
