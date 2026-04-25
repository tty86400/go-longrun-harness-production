package provider

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "longrunharness/internal/config"
)

type GenericJSON struct {
    cfg    config.ProviderConfig
    client *http.Client
    url    string
    method string
    name   string
}

func NewGenericJSON(cfg config.ProviderConfig) (*GenericJSON, error) {
    url := strings.TrimSpace(cfg.Endpoint)
    if url == "" {
        url = strings.TrimSpace(cfg.BaseURL)
    }
    if url == "" {
        return nil, fmt.Errorf("generic_json provider requires endpoint or base_url")
    }
    method := strings.ToUpper(strings.TrimSpace(cfg.Method))
    if method == "" {
        method = http.MethodPost
    }
    name := cfg.Name
    if name == "" {
        name = "generic_json"
    }
    return &GenericJSON{
        cfg: cfg,
        client: &http.Client{Timeout: cfg.TimeoutDuration()},
        url: url,
        method: method,
        name: name,
    }, nil
}

func (p *GenericJSON) Name() string { return p.name }

func (p *GenericJSON) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    started := time.Now()
    body := deepCopyMap(p.cfg.Body)
    if body == nil {
        body = map[string]any{}
    }

    switch strings.ToLower(strings.TrimSpace(p.cfg.InputMode)) {
    case "prompt":
        prompt := formatPrompt(req.Messages, req.ForceJSON, p.cfg.PromptTemplate)
        if err := setPath(body, p.cfg.PromptFieldPath, prompt); err != nil {
            return ChatResponse{}, fmt.Errorf("set prompt_field_path: %w", err)
        }
    default:
        if err := setPath(body, p.cfg.MessageFieldPath, req.Messages); err != nil {
            return ChatResponse{}, fmt.Errorf("set message_field_path: %w", err)
        }
    }

    if p.cfg.Model != "" {
        if _, ok := body["model"]; !ok {
            body["model"] = p.cfg.Model
        }
    }
    if _, ok := body["temperature"]; !ok {
        body["temperature"] = chooseFloat(req.Temperature, p.cfg.Temperature, 0.2)
    }
    if _, ok := body["max_tokens"]; !ok {
        if req.MaxOutputTokens > 0 {
            body["max_tokens"] = req.MaxOutputTokens
        } else if p.cfg.MaxOutputTokens > 0 {
            body["max_tokens"] = p.cfg.MaxOutputTokens
        }
    }
    if req.ForceJSON && p.cfg.JSONModeFieldPath != "" {
        if err := setPath(body, p.cfg.JSONModeFieldPath, p.cfg.JSONModeValue); err != nil {
            return ChatResponse{}, fmt.Errorf("set json_mode_field_path: %w", err)
        }
    }

    payload, err := json.Marshal(body)
    if err != nil {
        return ChatResponse{}, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, p.method, p.url, bytes.NewReader(payload))
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

    var decoded map[string]any
    if err := json.Unmarshal(raw, &decoded); err != nil {
        return ChatResponse{}, fmt.Errorf("decode provider response: %w", err)
    }
    textValue, err := getPath(decoded, p.cfg.ResponseTextPath)
    if err != nil {
        return ChatResponse{}, fmt.Errorf("read response_text_path=%q: %w", p.cfg.ResponseTextPath, err)
    }
    idValue := ""
    if p.cfg.ResponseIDPath != "" {
        if v, err := getPath(decoded, p.cfg.ResponseIDPath); err == nil {
            idValue = stringifyAny(v)
        }
    }

    return ChatResponse{
        ID:       idValue,
        Text:     stringifyAny(textValue),
        Raw:      raw,
        Duration: time.Since(started),
    }, nil
}

func formatPrompt(messages []Message, forceJSON bool, template string) string {
    var b strings.Builder
    for _, msg := range messages {
        b.WriteString(strings.ToUpper(msg.Role))
        b.WriteString(":\n")
        b.WriteString(msg.Content)
        b.WriteString("\n\n")
    }
    formatted := strings.TrimSpace(b.String())
    if template != "" {
        formatted = strings.ReplaceAll(template, "{{messages}}", formatted)
        if forceJSON {
            formatted = strings.ReplaceAll(formatted, "{{json_hint}}", "Return exactly one JSON object.")
        } else {
            formatted = strings.ReplaceAll(formatted, "{{json_hint}}", "")
        }
        return formatted
    }
    if forceJSON {
        formatted += "\n\nReturn exactly one JSON object."
    }
    return formatted
}

func deepCopyMap(in map[string]any) map[string]any {
    if in == nil {
        return nil
    }
    raw, _ := json.Marshal(in)
    var out map[string]any
    _ = json.Unmarshal(raw, &out)
    return out
}

func setPath(root map[string]any, path string, value any) error {
    parts := strings.Split(path, ".")
    current := root
    for i, part := range parts {
        if strings.TrimSpace(part) == "" {
            return fmt.Errorf("empty path component")
        }
        if i == len(parts)-1 {
            current[part] = value
            return nil
        }
        next, ok := current[part]
        if !ok {
            child := map[string]any{}
            current[part] = child
            current = child
            continue
        }
        child, ok := next.(map[string]any)
        if !ok {
            return fmt.Errorf("path component %q is not an object", part)
        }
        current = child
    }
    return nil
}

func getPath(root any, path string) (any, error) {
    current := root
    for _, part := range strings.Split(path, ".") {
        if strings.TrimSpace(part) == "" {
            return nil, fmt.Errorf("empty path component")
        }
        switch typed := current.(type) {
        case map[string]any:
            next, ok := typed[part]
            if !ok {
                return nil, fmt.Errorf("missing object key %q", part)
            }
            current = next
        case []any:
            var idx int
            if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
                return nil, fmt.Errorf("path component %q is not an array index", part)
            }
            if idx < 0 || idx >= len(typed) {
                return nil, fmt.Errorf("array index %d out of range", idx)
            }
            current = typed[idx]
        default:
            return nil, fmt.Errorf("cannot descend into %T", current)
        }
    }
    return current, nil
}

func stringifyAny(v any) string {
    switch typed := v.(type) {
    case string:
        return typed
    case []any:
        var parts []string
        for _, item := range typed {
            switch t := item.(type) {
            case string:
                parts = append(parts, t)
            case map[string]any:
                if text, ok := t["text"].(string); ok {
                    parts = append(parts, text)
                } else {
                    raw, _ := json.Marshal(t)
                    parts = append(parts, string(raw))
                }
            default:
                raw, _ := json.Marshal(t)
                parts = append(parts, string(raw))
            }
        }
        return strings.Join(parts, "")
    default:
        raw, _ := json.Marshal(v)
        return string(raw)
    }
}
