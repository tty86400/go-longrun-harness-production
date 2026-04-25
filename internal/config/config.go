package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type Config struct {
    Run           RunConfig           `json:"run"`
    Prompt        PromptConfig        `json:"prompt"`
    Loop          LoopConfig          `json:"loop"`
    Review        ReviewConfig        `json:"review"`
    Memory        MemoryConfig        `json:"memory"`
    Observability ObservabilityConfig `json:"observability"`
    Actor         ProviderConfig      `json:"actor"`
    Reviewer      *ProviderConfig     `json:"reviewer,omitempty"`
    Summarizer    *ProviderConfig     `json:"summarizer,omitempty"`
    Tools         ToolsConfig         `json:"tools"`
}

type RunConfig struct {
    IDPrefix          string `json:"id_prefix"`
    Workspace         string `json:"workspace"`
    RunsDir           string `json:"runs_dir"`
    ResumeRunID       string `json:"resume_run_id"`
    PersistPrompts    bool   `json:"persist_prompts"`
    PersistProviderIO bool   `json:"persist_provider_io"`
}

type PromptConfig struct {
    Language string `json:"language"`
}

type LoopConfig struct {
    MaxSteps               int `json:"max_steps"`
    MaxActionsPerStep      int `json:"max_actions_per_step"`
    StepTimeoutSeconds     int `json:"step_timeout_seconds"`
    MaxWallClockMinutes    int `json:"max_wall_clock_minutes"`
    MaxConsecutiveFailures int `json:"max_consecutive_failures"`
}

type ReviewConfig struct {
    EveryNSteps int `json:"every_n_steps"`
}

type MemoryConfig struct {
    RecentEvents          int `json:"recent_events"`
    EstimatedPromptBudget int `json:"estimated_prompt_budget"`
    SummaryTriggerTokens  int `json:"summary_trigger_tokens"`
    SummarizeEveryNSteps  int `json:"summarize_every_n_steps"`
}

type ObservabilityConfig struct {
    JSONLogs    bool   `json:"json_logs"`
    MetricsAddr string `json:"metrics_addr"`
    EnablePprof bool   `json:"enable_pprof"`
}

type RetryConfig struct {
    MaxAttempts      int `json:"max_attempts"`
    InitialBackoffMS int `json:"initial_backoff_ms"`
    MaxBackoffMS     int `json:"max_backoff_ms"`
    JitterMS         int `json:"jitter_ms"`
}

type ProviderConfig struct {
    Kind            string         `json:"kind"`
    Name            string         `json:"name"`
    Model           string         `json:"model"`
    BaseURL         string         `json:"base_url"`
    Endpoint        string         `json:"endpoint"`
    ChatPath        string         `json:"chat_path"`
    Method          string         `json:"method"`
    Headers         map[string]string `json:"headers"`
    APIKeyEnv       string         `json:"api_key_env"`
    AuthHeader      string         `json:"auth_header"`
    AuthPrefix      string         `json:"auth_prefix"`
    TimeoutSeconds  int            `json:"timeout_seconds"`
    Temperature     float64        `json:"temperature"`
    MaxOutputTokens int            `json:"max_output_tokens"`
    Body            map[string]any `json:"body"`

    InputMode        string `json:"input_mode"`
    MessageFieldPath string `json:"message_field_path"`
    PromptFieldPath  string `json:"prompt_field_path"`
    PromptTemplate   string `json:"prompt_template"`
    ResponseTextPath string `json:"response_text_path"`
    ResponseIDPath   string `json:"response_id_path"`
    JSONModeFieldPath string `json:"json_mode_field_path"`
    JSONModeValue    any    `json:"json_mode_value"`

    Retry RetryConfig `json:"retry"`
}

type ToolsConfig struct {
    Files     FilesConfig     `json:"files"`
    Shell     ShellConfig     `json:"shell"`
    Git       GitConfig       `json:"git"`
    HTTP      HTTPConfig      `json:"http"`
    Benchmark BenchmarkConfig `json:"benchmark"`
}

type FilesConfig struct {
    Enabled       bool `json:"enabled"`
    MaxReadBytes  int  `json:"max_read_bytes"`
    MaxWriteBytes int  `json:"max_write_bytes"`
    MaxListEntries int `json:"max_list_entries"`
}

type ShellConfig struct {
    Enabled               bool     `json:"enabled"`
    AllowedCommands       []string `json:"allowed_commands"`
    DeniedCommands        []string `json:"denied_commands"`
    DefaultTimeoutSeconds int      `json:"default_timeout_seconds"`
    MaxOutputBytes        int      `json:"max_output_bytes"`
    EnvAllowlist          []string `json:"env_allowlist"`
}

type GitConfig struct {
    Enabled bool `json:"enabled"`
}

type HTTPConfig struct {
    Enabled          bool     `json:"enabled"`
    AllowedDomains   []string `json:"allowed_domains"`
    TimeoutSeconds   int      `json:"timeout_seconds"`
    MaxResponseBytes int      `json:"max_response_bytes"`
}

type BenchmarkConfig struct {
    Enabled          bool              `json:"enabled"`
    Scripts          map[string]string `json:"scripts"`
    TimeoutSeconds   int               `json:"timeout_seconds"`
    MaxOutputBytes   int               `json:"max_output_bytes"`
}

func Load(path string) (*Config, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var intermediate any
    if err := json.Unmarshal(raw, &intermediate); err != nil {
        return nil, fmt.Errorf("parse config as JSON: %w", err)
    }
    intermediate = expandEnv(intermediate)

    normalized, err := json.Marshal(intermediate)
    if err != nil {
        return nil, fmt.Errorf("re-encode config: %w", err)
    }

    var cfg Config
    if err := json.Unmarshal(normalized, &cfg); err != nil {
        return nil, fmt.Errorf("decode config: %w", err)
    }

    cfg.applyDefaults(filepath.Dir(path))
    if err := cfg.validate(); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func expandEnv(v any) any {
    switch typed := v.(type) {
    case map[string]any:
        out := make(map[string]any, len(typed))
        for k, value := range typed {
            out[k] = expandEnv(value)
        }
        return out
    case []any:
        out := make([]any, 0, len(typed))
        for _, value := range typed {
            out = append(out, expandEnv(value))
        }
        return out
    case string:
        return os.ExpandEnv(typed)
    default:
        return v
    }
}

func (c *Config) applyDefaults(baseDir string) {
    if strings.TrimSpace(c.Run.IDPrefix) == "" {
        c.Run.IDPrefix = "run"
    }
    if strings.TrimSpace(c.Run.Workspace) == "" {
        c.Run.Workspace = filepath.Join(baseDir, "workspace")
    }
    if strings.TrimSpace(c.Run.RunsDir) == "" {
        c.Run.RunsDir = filepath.Join(baseDir, "runs")
    }
    c.Run.Workspace = resolveRelative(baseDir, c.Run.Workspace)
    c.Run.RunsDir = resolveRelative(baseDir, c.Run.RunsDir)

    if strings.TrimSpace(c.Prompt.Language) == "" {
        c.Prompt.Language = "zh-CN"
    }

    if c.Loop.MaxSteps <= 0 {
        c.Loop.MaxSteps = 120
    }
    if c.Loop.MaxActionsPerStep <= 0 {
        c.Loop.MaxActionsPerStep = 3
    }
    if c.Loop.StepTimeoutSeconds <= 0 {
        c.Loop.StepTimeoutSeconds = 180
    }
    if c.Loop.MaxWallClockMinutes <= 0 {
        c.Loop.MaxWallClockMinutes = 240
    }
    if c.Loop.MaxConsecutiveFailures <= 0 {
        c.Loop.MaxConsecutiveFailures = 6
    }

    if c.Review.EveryNSteps < 0 {
        c.Review.EveryNSteps = 0
    }

    if c.Memory.RecentEvents <= 0 {
        c.Memory.RecentEvents = 16
    }
    if c.Memory.EstimatedPromptBudget <= 0 {
        c.Memory.EstimatedPromptBudget = 12000
    }
    if c.Memory.SummaryTriggerTokens <= 0 {
        c.Memory.SummaryTriggerTokens = 9000
    }
    if c.Memory.SummarizeEveryNSteps <= 0 {
        c.Memory.SummarizeEveryNSteps = 8
    }

    if c.Actor.Name == "" {
        c.Actor.Name = "actor"
    }
    c.Actor.applyDefaults()
    if c.Reviewer != nil {
        if c.Reviewer.Name == "" {
            c.Reviewer.Name = "reviewer"
        }
        c.Reviewer.applyDefaults()
    }
    if c.Summarizer != nil {
        if c.Summarizer.Name == "" {
            c.Summarizer.Name = "summarizer"
        }
        c.Summarizer.applyDefaults()
    }

    if c.Tools.Files.MaxReadBytes <= 0 {
        c.Tools.Files.MaxReadBytes = 1 << 20
    }
    if c.Tools.Files.MaxWriteBytes <= 0 {
        c.Tools.Files.MaxWriteBytes = 1 << 20
    }
    if c.Tools.Files.MaxListEntries <= 0 {
        c.Tools.Files.MaxListEntries = 200
    }

    if c.Tools.Shell.DefaultTimeoutSeconds <= 0 {
        c.Tools.Shell.DefaultTimeoutSeconds = 60
    }
    if c.Tools.Shell.MaxOutputBytes <= 0 {
        c.Tools.Shell.MaxOutputBytes = 64 << 10
    }

    if c.Tools.HTTP.TimeoutSeconds <= 0 {
        c.Tools.HTTP.TimeoutSeconds = 30
    }
    if c.Tools.HTTP.MaxResponseBytes <= 0 {
        c.Tools.HTTP.MaxResponseBytes = 512 << 10
    }

    if c.Tools.Benchmark.TimeoutSeconds <= 0 {
        c.Tools.Benchmark.TimeoutSeconds = 300
    }
    if c.Tools.Benchmark.MaxOutputBytes <= 0 {
        c.Tools.Benchmark.MaxOutputBytes = 128 << 10
    }
}

func (p *ProviderConfig) applyDefaults() {
    if strings.TrimSpace(p.AuthHeader) == "" {
        p.AuthHeader = "Authorization"
    }
    if strings.TrimSpace(p.AuthPrefix) == "" {
        p.AuthPrefix = "Bearer "
    }
    if p.TimeoutSeconds <= 0 {
        p.TimeoutSeconds = 90
    }
    if p.Temperature == 0 {
        p.Temperature = 0.2
    }
    if p.MaxOutputTokens <= 0 {
        p.MaxOutputTokens = 1200
    }
    if p.Retry.MaxAttempts <= 0 {
        p.Retry.MaxAttempts = 4
    }
    if p.Retry.InitialBackoffMS <= 0 {
        p.Retry.InitialBackoffMS = 500
    }
    if p.Retry.MaxBackoffMS <= 0 {
        p.Retry.MaxBackoffMS = 5000
    }
    if p.Retry.JitterMS <= 0 {
        p.Retry.JitterMS = 250
    }

    if strings.TrimSpace(p.ChatPath) == "" {
        p.ChatPath = "/v1/chat/completions"
    }
    if strings.TrimSpace(p.Method) == "" {
        p.Method = "POST"
    }
    if strings.TrimSpace(p.InputMode) == "" {
        p.InputMode = "messages"
    }
    if strings.TrimSpace(p.MessageFieldPath) == "" {
        p.MessageFieldPath = "messages"
    }
    if strings.TrimSpace(p.PromptFieldPath) == "" {
        p.PromptFieldPath = "prompt"
    }
    if strings.TrimSpace(p.ResponseTextPath) == "" {
        p.ResponseTextPath = "choices.0.message.content"
    }
}

func (c *Config) validate() error {
    if strings.TrimSpace(c.Run.Workspace) == "" {
        return fmt.Errorf("run.workspace is required")
    }
    if strings.TrimSpace(c.Run.RunsDir) == "" {
        return fmt.Errorf("run.runs_dir is required")
    }
    if strings.TrimSpace(c.Actor.Kind) == "" {
        return fmt.Errorf("actor.kind is required")
    }
    if c.Loop.MaxSteps < 1 {
        return fmt.Errorf("loop.max_steps must be >= 1")
    }
    if c.Loop.MaxActionsPerStep < 1 {
        return fmt.Errorf("loop.max_actions_per_step must be >= 1")
    }
    if c.Memory.RecentEvents < 1 {
        return fmt.Errorf("memory.recent_events must be >= 1")
    }
    for _, p := range []*ProviderConfig{&c.Actor, c.Reviewer, c.Summarizer} {
        if p == nil {
            continue
        }
        switch strings.TrimSpace(strings.ToLower(p.Kind)) {
        case "openai_compatible", "generic_json", "mock":
        default:
            return fmt.Errorf("unsupported provider kind %q", p.Kind)
        }
        if p.Kind != "mock" && strings.TrimSpace(p.Endpoint) == "" && strings.TrimSpace(p.BaseURL) == "" {
            return fmt.Errorf("provider %q requires endpoint or base_url", p.Name)
        }
    }
    return nil
}

func resolveRelative(baseDir, maybeRelative string) string {
    if filepath.IsAbs(maybeRelative) {
        return filepath.Clean(maybeRelative)
    }
    return filepath.Clean(filepath.Join(baseDir, maybeRelative))
}

func (p ProviderConfig) TimeoutDuration() time.Duration {
    return time.Duration(p.TimeoutSeconds) * time.Second
}

func (l LoopConfig) MaxWallClockDuration() time.Duration {
    return time.Duration(l.MaxWallClockMinutes) * time.Minute
}
