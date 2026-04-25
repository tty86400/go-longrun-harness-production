package harness

import "time"

type State struct {
    RunID                string    `json:"run_id"`
    Status               string    `json:"status"`
    Objective            string    `json:"objective"`
    StartedAt            time.Time `json:"started_at"`
    UpdatedAt            time.Time `json:"updated_at"`
    EndedAt              time.Time `json:"ended_at,omitempty"`
    Step                 int       `json:"step"`
    ResumeCount          int       `json:"resume_count"`
    ConsecutiveFailures  int       `json:"consecutive_failures"`
    Completed            bool      `json:"completed"`
    FinalAnswer          string    `json:"final_answer,omitempty"`
    RollingSummary       string    `json:"rolling_summary,omitempty"`
    StickyFacts          []string  `json:"sticky_facts,omitempty"`
    Plan                 []string  `json:"plan,omitempty"`
    Todo                 []string  `json:"todo,omitempty"`
    RecentEvents         []Event   `json:"recent_events,omitempty"`
    LastError            string    `json:"last_error,omitempty"`
    MemoryCompactions    int       `json:"memory_compactions"`
    LastSummaryAtStep    int       `json:"last_summary_at_step"`
    Stats                RunStats  `json:"stats"`
}

type RunStats struct {
    Steps          int `json:"steps"`
    ActorCalls     int `json:"actor_calls"`
    ReviewerCalls  int `json:"reviewer_calls"`
    SummarizerCalls int `json:"summarizer_calls"`
    ToolCalls      int `json:"tool_calls"`
    ToolFailures   int `json:"tool_failures"`
    Reviews        int `json:"reviews"`
    Summaries      int `json:"summaries"`
    Checkpoints    int `json:"checkpoints"`
}

type Event struct {
    Timestamp  time.Time      `json:"timestamp"`
    Step       int            `json:"step"`
    Kind       string         `json:"kind"`
    Actor      string         `json:"actor,omitempty"`
    Tool       string         `json:"tool,omitempty"`
    OK         bool           `json:"ok,omitempty"`
    Title      string         `json:"title,omitempty"`
    Content    string         `json:"content,omitempty"`
    Artifact   string         `json:"artifact,omitempty"`
    DurationMS int64          `json:"duration_ms,omitempty"`
    Metadata   map[string]any `json:"metadata,omitempty"`
}

type RunResult struct {
    State *State `json:"state"`
}

type ActorDecision struct {
    SituationAssessment string        `json:"situation_assessment"`
    UpdatedPlan         []string      `json:"updated_plan"`
    UpdatedTodo         []string      `json:"updated_todo,omitempty"`
    Actions             []Action      `json:"actions"`
    Done                bool          `json:"done"`
    FinalAnswer         string        `json:"final_answer"`
}

type Action struct {
    ID             string         `json:"id,omitempty"`
    Tool           string         `json:"tool"`
    Args           map[string]any `json:"args"`
    Reason         string         `json:"reason"`
    TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

type ReviewReport struct {
    ProgressAssessment string   `json:"progress_assessment"`
    DoneItems          []string `json:"done_items"`
    MissingItems       []string `json:"missing_items"`
    Risks              []string `json:"risks"`
    NextPriority       string   `json:"next_priority"`
    RevisedPlan        []string `json:"revised_plan"`
    UpdatedTodo        []string `json:"updated_todo"`
    ShouldSummarize    bool     `json:"should_summarize"`
}

type SummaryReport struct {
    RollingSummary string   `json:"rolling_summary"`
    StickyFacts    []string `json:"sticky_facts"`
    CarryOverPlan  []string `json:"carry_over_plan"`
    CarryOverTodo  []string `json:"carry_over_todo"`
}
