package provider

import (
    "context"
    "time"
)

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Messages        []Message         `json:"messages"`
    Temperature     float64           `json:"temperature,omitempty"`
    MaxOutputTokens int               `json:"max_output_tokens,omitempty"`
    ForceJSON       bool              `json:"force_json,omitempty"`
    Metadata        map[string]string `json:"metadata,omitempty"`
}

type Usage struct {
    InputTokens  int `json:"input_tokens,omitempty"`
    OutputTokens int `json:"output_tokens,omitempty"`
    TotalTokens  int `json:"total_tokens,omitempty"`
}

type ChatResponse struct {
    ID       string        `json:"id,omitempty"`
    Text     string        `json:"text"`
    Usage    Usage         `json:"usage,omitempty"`
    Raw      []byte        `json:"raw,omitempty"`
    Duration time.Duration `json:"duration,omitempty"`
}

type Provider interface {
    Name() string
    Generate(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
