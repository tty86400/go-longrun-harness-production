package harness

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"

    "longrunharness/internal/config"
    "longrunharness/internal/provider"
    "longrunharness/internal/util"
)

type Session struct {
    RunID          string
    RunDir         string
    Workspace      string
    statePath      string
    transcriptPath string
    promptsDir     string
    artifactsDir   string
    checkpointsDir string
    auditDir       string
    lockPath       string
    lockFile       *os.File
    mu             sync.Mutex
}

type PromptAudit struct {
    Timestamp   time.Time          `json:"timestamp"`
    Step        int                `json:"step"`
    Role        string             `json:"role"`
    Messages    []provider.Message `json:"messages"`
    ResponseID  string             `json:"response_id,omitempty"`
    ResponseText string            `json:"response_text,omitempty"`
    RawProvider string             `json:"raw_provider,omitempty"`
    Error       string             `json:"error,omitempty"`
}

func NewSession(cfg config.RunConfig, runID string, objective string) (*Session, *State, error) {
    if runID == "" {
        runID = generateRunID(cfg.IDPrefix)
    }
    if err := os.MkdirAll(cfg.Workspace, 0o755); err != nil {
        return nil, nil, err
    }
    if err := os.MkdirAll(cfg.RunsDir, 0o755); err != nil {
        return nil, nil, err
    }

    runDir := filepath.Join(cfg.RunsDir, runID)
    s := &Session{
        RunID:          runID,
        RunDir:         runDir,
        Workspace:      cfg.Workspace,
        statePath:      filepath.Join(runDir, "state.json"),
        transcriptPath: filepath.Join(runDir, "transcript.jsonl"),
        promptsDir:     filepath.Join(runDir, "prompts"),
        artifactsDir:   filepath.Join(runDir, "artifacts"),
        checkpointsDir: filepath.Join(runDir, "checkpoints"),
        auditDir:       filepath.Join(runDir, "audit"),
        lockPath:       filepath.Join(runDir, ".lock"),
    }
    if err := s.initDirs(); err != nil {
        return nil, nil, err
    }
    if err := s.acquireLock(); err != nil {
        return nil, nil, err
    }

    now := time.Now().UTC()
    state := &State{
        RunID:      runID,
        Status:     "running",
        Objective:  objective,
        StartedAt:  now,
        UpdatedAt:  now,
        Plan:       []string{"Inspect current state before making changes.", "Make one small auditable move at a time.", "Verify outcomes before claiming success."},
        Todo:       []string{"Reach the objective with durable evidence in the workspace and run artifacts."},
        StickyFacts: []string{"Workspace and artifacts are the durable source of truth.", "Keep prompt context compact; summarize aggressively when needed."},
        RecentEvents: []Event{},
    }
    if err := s.SaveState(state); err != nil {
        s.Close()
        return nil, nil, err
    }
    return s, state, nil
}

func ResumeSession(cfg config.RunConfig, runID string) (*Session, *State, error) {
    if runID == "" {
        return nil, nil, fmt.Errorf("resume run id is required")
    }
    runDir := filepath.Join(cfg.RunsDir, runID)
    s := &Session{
        RunID:          runID,
        RunDir:         runDir,
        Workspace:      cfg.Workspace,
        statePath:      filepath.Join(runDir, "state.json"),
        transcriptPath: filepath.Join(runDir, "transcript.jsonl"),
        promptsDir:     filepath.Join(runDir, "prompts"),
        artifactsDir:   filepath.Join(runDir, "artifacts"),
        checkpointsDir: filepath.Join(runDir, "checkpoints"),
        auditDir:       filepath.Join(runDir, "audit"),
        lockPath:       filepath.Join(runDir, ".lock"),
    }
    if err := s.initDirs(); err != nil {
        return nil, nil, err
    }
    if err := s.acquireLock(); err != nil {
        return nil, nil, err
    }

    var state State
    raw, err := os.ReadFile(s.statePath)
    if err != nil {
        s.Close()
        return nil, nil, err
    }
    if err := json.Unmarshal(raw, &state); err != nil {
        s.Close()
        return nil, nil, err
    }
    state.Status = "running"
    state.ResumeCount++
    state.UpdatedAt = time.Now().UTC()
    if err := s.SaveState(&state); err != nil {
        s.Close()
        return nil, nil, err
    }
    return s, &state, nil
}

func (s *Session) initDirs() error {
    for _, dir := range []string{s.RunDir, s.promptsDir, s.artifactsDir, s.checkpointsDir, s.auditDir} {
        if err := os.MkdirAll(dir, 0o755); err != nil {
            return err
        }
    }
    return nil
}

func (s *Session) acquireLock() error {
    file, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
    if err != nil {
        if os.IsExist(err) {
            return fmt.Errorf("run %s is locked; remove %s only if the previous process is definitely gone", s.RunID, s.lockPath)
        }
        return err
    }
    s.lockFile = file
    _, _ = file.WriteString(fmt.Sprintf("pid=%d\ntime=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339)))
    return nil
}

func (s *Session) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.lockFile != nil {
        _ = s.lockFile.Close()
        _ = os.Remove(s.lockPath)
        s.lockFile = nil
    }
    return nil
}

func (s *Session) SaveState(state *State) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return util.WriteJSONAtomic(s.statePath, state, 0o644)
}

func (s *Session) SaveCheckpoint(state *State) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    path := filepath.Join(s.checkpointsDir, fmt.Sprintf("state-step-%04d.json", state.Step))
    return util.WriteJSONAtomic(path, state, 0o644)
}

func (s *Session) AppendEvent(event Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    f, err := os.OpenFile(s.transcriptPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    if _, err := f.Write(append(data, '\n')); err != nil {
        return err
    }
    return f.Sync()
}

func (s *Session) SavePromptAudit(step int, role string, messages []provider.Message, response provider.ChatResponse, rawText string, err error) error {
    audit := PromptAudit{
        Timestamp:   time.Now().UTC(),
        Step:        step,
        Role:        role,
        Messages:    messages,
        ResponseID:  response.ID,
        ResponseText: rawText,
    }
    if len(response.Raw) > 0 {
        audit.RawProvider = string(response.Raw)
    }
    if err != nil {
        audit.Error = err.Error()
    }
    filePath := filepath.Join(s.promptsDir, fmt.Sprintf("step-%04d-%s.json", step, role))
    return util.WriteJSONAtomic(filePath, audit, 0o644)
}

func (s *Session) SaveArtifact(step int, actionIndex int, name string, value any) (string, error) {
    rel := filepath.ToSlash(filepath.Join("artifacts", fmt.Sprintf("step-%04d-action-%02d-%s.json", step, actionIndex, sanitizeName(name))))
    abs := filepath.Join(s.RunDir, filepath.FromSlash(rel))
    if err := util.WriteJSONAtomic(abs, value, 0o644); err != nil {
        return "", err
    }
    return rel, nil
}

func (s *Session) WriteRunReport(markdown string) (string, error) {
    path := filepath.Join(s.RunDir, "RUN_REPORT.md")
    if err := util.WriteFileAtomic(path, []byte(markdown), 0o644); err != nil {
        return "", err
    }
    return path, nil
}

func sanitizeName(name string) string {
    clean := make([]rune, 0, len(name))
    for _, r := range name {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
            clean = append(clean, r)
            continue
        }
        clean = append(clean, '-')
    }
    if len(clean) == 0 {
        return "artifact"
    }
    return string(clean)
}

func generateRunID(prefix string) string {
    if prefix == "" {
        prefix = "run"
    }
    randBytes := make([]byte, 3)
    _, _ = rand.Read(randBytes)
    return fmt.Sprintf("%s-%s-%s", prefix, time.Now().UTC().Format("20060102-150405"), hex.EncodeToString(randBytes))
}
