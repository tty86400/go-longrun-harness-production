package provider

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "time"

    "longrunharness/internal/config"
)

type OpenAICompatible struct {
    cfg    config.ProviderConfig
    client *http.Client
    url    string
    name   string
}

func NewOpenAICompatible(cfg config.ProviderConfig) (*OpenAICompatible, error) {
    baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
    url := strings.TrimSpace(cfg.Endpoint)
    if url == "" {
        if baseURL == "" {
            return nil, fmt.Errorf("openai_compatible provider requires base_url or endpoint")
        }
        path := cfg.ChatPath
        if path == "" {
            path = "/v1/chat/completions"
        }
        url = baseURL + path
    }
    name := cfg.Name
    if name == "" {
        name = "openai_compatible"
    }
    return &OpenAICompatible{
        cfg: cfg,
        client: &http.Client{Timeout: cfg.TimeoutDuration()},
        url: url,
        name: name,
    }, nil
}

func (p *OpenAICompatible) Name() string { return p.name }

func (p *OpenAICompatible) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    started := time.Now()
    payload := map[string]any{
        "model":       p.cfg.Model,
        "messages":    req.Messages,
        "temperature": chooseFloat(req.Temperature, p.cfg.Temperature, 0.2),
    }
    if req.MaxOutputTokens > 0 {
        payload["max_tokens"] = req.MaxOutputTokens
    } else if p.cfg.MaxOutputTokens > 0 {
        payload["max_tokens"] = p.cfg.MaxOutputTokens
    }
    if req.ForceJSON {
        payload["response_format"] = map[string]any{"type": "json_object"}
    }
    for k, v := range p.cfg.Body {
        payload[k] = v
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return ChatResponse{}, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
    if err != nil {
        return ChatResponse{}, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    applyAuth(httpReq, p.cfg)
    for k, v := range p.cfg.Headers {
        httpReq.Header.Set(k, v)
    }

    resp, err := p.client.Do(httpReq)
    if err != nil {
        return ChatResponse{}, RetryableError{Err: err}
    }
    defer resp.Body.Close()

    raw, err := io.ReadAll(resp.Body)
    if err != nil {
        return ChatResponse{}, RetryableError{Err: err}
    }

    if resp.StatusCode >= 500 || resp.StatusCode == 429 {
        return ChatResponse{}, RetryableError{Err: fmt.Errorf("provider %s HTTP %d: %s", p.name, resp.StatusCode, string(raw))}
    }
    if resp.StatusCode >= 300 {
        return ChatResponse{}, fmt.Errorf("provider %s HTTP %d: %s", p.name, resp.StatusCode, string(raw))
    }

    var decoded struct {
        ID string `json:"id"`
        Choices []struct {
            Message struct {
                Content json.RawMessage `json:"content"`
            } `json:"message"`
        } `json:"choices"`
        Usage struct {
            PromptTokens     int `json:"prompt_tokens"`
            CompletionTokens int `json:"completion_tokens"`
            TotalTokens      int `json:"total_tokens"`
        } `json:"usage"`
    }
    if err := json.Unmarshal(raw, &decoded); err != nil {
        return ChatResponse{}, fmt.Errorf("decode provider response: %w", err)
    }
    if len(decoded.Choices) == 0 {
        return ChatResponse{}, fmt.Errorf("provider %s returned zero choices", p.name)
    }

    return ChatResponse{
        ID:   decoded.ID,
        Text: extractOpenAIContent(decoded.Choices[0].Message.Content),
        Usage: Usage{
            InputTokens:  decoded.Usage.PromptTokens,
            OutputTokens: decoded.Usage.CompletionTokens,
            TotalTokens:  decoded.Usage.TotalTokens,
        },
        Raw:      raw,
        Duration: time.Since(started),
    }, nil
}

func extractOpenAIContent(raw json.RawMessage) string {
    if len(raw) == 0 {
        return ""
    }
    var asString string
    if err := json.Unmarshal(raw, &asString); err == nil {
        return asString
    }

    var parts []map[string]any
    if err := json.Unmarshal(raw, &parts); err == nil {
        var chunks []string
        for _, part := range parts {
            if text, ok := part["text"].(string); ok {
                chunks = append(chunks, text)
                continue
            }
            if t, ok := part["type"].(string); ok && (t == "text" || t == "output_text") {
                if text, ok := part["text"].(string); ok {
                    chunks = append(chunks, text)
                }
            }
        }
        return strings.Join(chunks, "")
    }
    return string(raw)
}

func applyAuth(req *http.Request, cfg config.ProviderConfig) {
    if strings.TrimSpace(cfg.APIKeyEnv) == "" {
        return
    }
    value := os.Getenv(cfg.APIKeyEnv)
    if strings.TrimSpace(value) == "" {
        return
    }
    header := cfg.AuthHeader
    if header == "" {
        header = "Authorization"
    }
    prefix := cfg.AuthPrefix
    if prefix == "" {
        prefix = "Bearer "
    }
    req.Header.Set(header, prefix+value)
}

func chooseFloat(values ...float64) float64 {
    for _, v := range values {
        if v != 0 {
            return v
        }
    }
    return 0
}
