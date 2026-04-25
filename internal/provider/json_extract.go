package provider

import (
    "encoding/json"
    "fmt"
    "strings"
)

func ExtractJSONObjectText(text string) (string, error) {
    cleaned := strings.TrimSpace(text)
    if cleaned == "" {
        return "", fmt.Errorf("empty model response")
    }
    if json.Valid([]byte(cleaned)) {
        return cleaned, nil
    }
    cleaned = stripCodeFence(cleaned)
    if json.Valid([]byte(cleaned)) {
        return cleaned, nil
    }
    extracted, ok := scanJSONObject(cleaned)
    if ok && json.Valid([]byte(extracted)) {
        return extracted, nil
    }
    return "", fmt.Errorf("could not extract a valid JSON object from model output")
}

func stripCodeFence(text string) string {
    trimmed := strings.TrimSpace(text)
    if !strings.HasPrefix(trimmed, "```") {
        return trimmed
    }
    trimmed = strings.TrimPrefix(trimmed, "```")
    if idx := strings.Index(trimmed, "\n"); idx >= 0 {
        header := strings.TrimSpace(trimmed[:idx])
        if header == "json" || header == "JSON" || header == "" {
            trimmed = trimmed[idx+1:]
        }
    }
    trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")
    return strings.TrimSpace(trimmed)
}

func scanJSONObject(text string) (string, bool) {
    start := -1
    depth := 0
    inString := false
    escaped := false
    runes := []rune(text)
    for i, r := range runes {
        if start == -1 {
            if r == '{' {
                start = i
                depth = 1
            }
            continue
        }
        if inString {
            if escaped {
                escaped = false
                continue
            }
            if r == '\\' {
                escaped = true
                continue
            }
            if r == '"' {
                inString = false
            }
            continue
        }
        switch r {
        case '"':
            inString = true
        case '{':
            depth++
        case '}':
            depth--
            if depth == 0 {
                return string(runes[start : i+1]), true
            }
        }
    }
    return "", false
}
