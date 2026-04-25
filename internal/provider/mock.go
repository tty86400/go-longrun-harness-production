package provider

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "time"

    "longrunharness/internal/config"
)

type Mock struct {
    name string
}

func NewMock(cfg config.ProviderConfig) (*Mock, error) {
    name := cfg.Name
    if name == "" {
        name = "mock"
    }
    return &Mock{name: name}, nil
}

func (m *Mock) Name() string { return m.name }

func (m *Mock) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    _ = ctx
    role := ""
    prompt := flattenMessages(req.Messages)
    if strings.Contains(prompt, "[ROLE:ACTOR]") {
        role = "actor"
    } else if strings.Contains(prompt, "[ROLE:REVIEWER]") {
        role = "reviewer"
    } else if strings.Contains(prompt, "[ROLE:SUMMARIZER]") {
        role = "summarizer"
    }

    step := 0
    re := regexp.MustCompile(`Current step:\s*(\d+)`)
    if matches := re.FindStringSubmatch(prompt); len(matches) == 2 {
        fmt.Sscanf(matches[1], "%d", &step)
    }

    var text string
    switch role {
    case "actor":
        switch step {
        case 1:
            text = `{"situation_assessment":"Workspace is empty. Create a deterministic demo artifact first.","updated_plan":["Create demo file","Inspect the file","Finish with a concise summary"],"updated_todo":["Write demo.txt","Read demo.txt","Return final answer"],"actions":[{"tool":"files.write","args":{"path":"demo.txt","content":"hello from production harness\n","create_dirs":true},"reason":"Seed the workspace with a verifiable artifact.","timeout_seconds":10}],"done":false,"final_answer":""}`
        case 2:
            text = `{"situation_assessment":"The file should now exist. Read it back as evidence.","updated_plan":["Create demo file","Inspect the file","Finish with a concise summary"],"updated_todo":["Read demo.txt","Return final answer"],"actions":[{"tool":"files.read","args":{"path":"demo.txt"},"reason":"Verify that the previous action really changed the workspace.","timeout_seconds":10}],"done":false,"final_answer":""}`
        default:
            text = `{"situation_assessment":"The artifact was created and verified.","updated_plan":["Create demo file","Inspect the file","Finish with a concise summary"],"updated_todo":[],"actions":[],"done":true,"final_answer":"Completed a deterministic end-to-end run: wrote demo.txt, read it back, and persisted the run state, transcript, checkpoints, and report."}`
        }
    case "reviewer":
        text = `{"progress_assessment":"The run is making steady progress and has real evidence in the workspace.","done_items":["Created or verified the demo artifact"],"missing_items":["Return the final answer once verification is complete"],"risks":["Do not keep issuing write actions after evidence is enough"],"next_priority":"Prefer evidence gathering over speculative edits.","revised_plan":["Create demo file","Inspect the file","Finish with a concise summary"],"updated_todo":["Read demo.txt","Return final answer"],"should_summarize":false}`
    case "summarizer":
        text = `{"rolling_summary":"The harness created demo.txt and verified its contents. The next useful action is to stop and report success.","sticky_facts":["Workspace changes are the durable source of truth","Prompt history should stay small"],"carry_over_plan":["Return final answer"],"carry_over_todo":["Return final answer"]}`
    default:
        text = `{}`
    }

    return ChatResponse{
        ID:       fmt.Sprintf("mock-%s-%d", role, step),
        Text:     text,
        Duration: 10 * time.Millisecond,
    }, nil
}

func flattenMessages(messages []Message) string {
    var b strings.Builder
    for _, msg := range messages {
        b.WriteString(msg.Role)
        b.WriteString(":")
        b.WriteString(msg.Content)
        b.WriteString("\n")
    }
    return b.String()
}
