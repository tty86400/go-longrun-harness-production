package tools

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    "longrunharness/internal/config"
)

type httpFetchTool struct {
    cfg config.HTTPConfig
}

func NewHTTPFetchTool(cfg config.HTTPConfig) Tool {
    return &httpFetchTool{cfg: cfg}
}

func (t *httpFetchTool) Name() string { return "http.fetch" }
func (t *httpFetchTool) Description() string { return "Fetch a URL with GET, subject to domain allowlisting and response-size limits." }
func (t *httpFetchTool) JSONSchema() map[string]any {
    return map[string]any{"type": "object", "required": []string{"url"}, "properties": map[string]any{"url": map[string]any{"type": "string"}}}
}
func (t *httpFetchTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
    rawURL := getStringArg(args, "url")
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return Result{}, err
    }
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return Result{}, fmt.Errorf("only http/https URLs are allowed")
    }
    if !t.hostAllowed(parsed.Hostname()) {
        return Result{}, fmt.Errorf("host %q is not allowed", parsed.Hostname())
    }

    client := &http.Client{Timeout: time.Duration(t.cfg.TimeoutSeconds) * time.Second}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
    if err != nil {
        return Result{}, err
    }
    resp, err := client.Do(req)
    if err != nil {
        return Result{}, err
    }
    defer resp.Body.Close()

    lr := &io.LimitedReader{R: resp.Body, N: int64(t.cfg.MaxResponseBytes)}
    body, err := io.ReadAll(lr)
    if err != nil {
        return Result{}, err
    }
    truncated := lr.N == 0
    return Result{OK: resp.StatusCode < 300, Summary: fmt.Sprintf("HTTP %d", resp.StatusCode), Output: string(body), Metadata: map[string]any{"status_code": resp.StatusCode, "url": parsed.String()}, Truncated: truncated}, nil
}

func (t *httpFetchTool) hostAllowed(host string) bool {
    if len(t.cfg.AllowedDomains) == 0 {
        return false
    }
    lower := strings.ToLower(host)
    for _, allowed := range t.cfg.AllowedDomains {
        allowed = strings.ToLower(strings.TrimSpace(allowed))
        if lower == allowed || strings.HasSuffix(lower, "."+allowed) {
            return true
        }
    }
    return false
}
