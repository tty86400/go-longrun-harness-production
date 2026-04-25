package tools

import "context"

type Result struct {
    OK        bool           `json:"ok"`
    Summary   string         `json:"summary"`
    Output    string         `json:"output,omitempty"`
    ExitCode  int            `json:"exit_code,omitempty"`
    Files     []string       `json:"files,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Truncated bool           `json:"truncated,omitempty"`
}

type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]any
    Execute(ctx context.Context, args map[string]any) (Result, error)
}
