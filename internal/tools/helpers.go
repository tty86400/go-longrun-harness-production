package tools

import (
    "bytes"
    "fmt"
    "io"
    "strings"
)

func getStringArg(args map[string]any, key string) string {
    if args == nil {
        return ""
    }
    if v, ok := args[key]; ok {
        return strings.TrimSpace(fmt.Sprintf("%v", v))
    }
    return ""
}

func getBoolArg(args map[string]any, key string, fallback bool) bool {
    if args == nil {
        return fallback
    }
    v, ok := args[key]
    if !ok {
        return fallback
    }
    switch typed := v.(type) {
    case bool:
        return typed
    case string:
        lower := strings.ToLower(strings.TrimSpace(typed))
        return lower == "true" || lower == "1" || lower == "yes" || lower == "y"
    default:
        return fallback
    }
}

func getIntArg(args map[string]any, key string, fallback int) int {
    if args == nil {
        return fallback
    }
    v, ok := args[key]
    if !ok {
        return fallback
    }
    switch typed := v.(type) {
    case int:
        return typed
    case int64:
        return int(typed)
    case float64:
        return int(typed)
    case string:
        var out int
        if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &out); err == nil {
            return out
        }
    }
    return fallback
}

func getStringSliceArg(args map[string]any, key string) []string {
    if args == nil {
        return nil
    }
    raw, ok := args[key]
    if !ok {
        return nil
    }
    switch typed := raw.(type) {
    case []string:
        out := make([]string, 0, len(typed))
        for _, item := range typed {
            out = append(out, strings.TrimSpace(item))
        }
        return out
    case []any:
        out := make([]string, 0, len(typed))
        for _, item := range typed {
            out = append(out, strings.TrimSpace(fmt.Sprintf("%v", item)))
        }
        return out
    default:
        return nil
    }
}

type limitedBuffer struct {
    buf       bytes.Buffer
    maxBytes  int
    truncated bool
}

func newLimitedBuffer(maxBytes int) *limitedBuffer {
    return &limitedBuffer{maxBytes: maxBytes}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
    if b.maxBytes <= 0 {
        return b.buf.Write(p)
    }
    remaining := b.maxBytes - b.buf.Len()
    if remaining <= 0 {
        b.truncated = true
        return len(p), nil
    }
    if len(p) > remaining {
        b.truncated = true
        _, _ = b.buf.Write(p[:remaining])
        return len(p), nil
    }
    return b.buf.Write(p)
}

func (b *limitedBuffer) String() string { return b.buf.String() }
func (b *limitedBuffer) Truncated() bool { return b.truncated }

func combineOutput(writers ...*limitedBuffer) (string, bool) {
    var chunks []string
    truncated := false
    for _, w := range writers {
        if w == nil {
            continue
        }
        content := strings.TrimSpace(w.String())
        if content != "" {
            chunks = append(chunks, content)
        }
        truncated = truncated || w.Truncated()
    }
    return strings.Join(chunks, "\n"), truncated
}

func teeWriters(ws ...io.Writer) io.Writer {
    return io.MultiWriter(ws...)
}
