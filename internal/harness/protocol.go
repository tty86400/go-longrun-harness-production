package harness

import (
    "encoding/json"
    "fmt"
    "strings"

    "longrunharness/internal/provider"
    "longrunharness/internal/tools"
)

func decodeJSONStrict[T any](text string) (T, string, error) {
    var zero T
    extracted, err := provider.ExtractJSONObjectText(text)
    if err != nil {
        return zero, "", err
    }
    dec := json.NewDecoder(strings.NewReader(extracted))
    dec.DisallowUnknownFields()
    var out T
    if err := dec.Decode(&out); err != nil {
        return zero, extracted, err
    }
    return out, extracted, nil
}

func validateDecision(decision ActorDecision, registry *tools.Registry, maxActions int) error {
    if decision.Done {
        if len(decision.Actions) > 0 {
            return fmt.Errorf("done=true requires actions=[]")
        }
        if strings.TrimSpace(decision.FinalAnswer) == "" {
            return fmt.Errorf("done=true requires final_answer")
        }
        return nil
    }
    if len(decision.Actions) == 0 {
        return fmt.Errorf("decision has no actions and is not done")
    }
    if len(decision.Actions) > maxActions {
        return fmt.Errorf("decision has %d actions, exceeds limit %d", len(decision.Actions), maxActions)
    }
    for i, action := range decision.Actions {
        if strings.TrimSpace(action.Tool) == "" {
            return fmt.Errorf("action %d is missing tool", i)
        }
        if !registry.Exists(action.Tool) {
            return fmt.Errorf("action %d uses unknown tool %q", i, action.Tool)
        }
        if strings.TrimSpace(action.Reason) == "" {
            return fmt.Errorf("action %d is missing reason", i)
        }
        if action.TimeoutSeconds < 0 {
            return fmt.Errorf("action %d has invalid timeout_seconds", i)
        }
    }
    return nil
}
