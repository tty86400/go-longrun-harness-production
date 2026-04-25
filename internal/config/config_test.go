package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadExpandsEnv(t *testing.T) {
    t.Setenv("HARNESS_WORKSPACE", filepath.Join(t.TempDir(), "ws"))
    dir := t.TempDir()
    path := filepath.Join(dir, "config.json")
    payload := `{
      "run": {"workspace": "${HARNESS_WORKSPACE}", "runs_dir": "./runs"},
      "actor": {"kind": "mock"},
      "tools": {"files": {"enabled": true}}
    }`
    if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
        t.Fatal(err)
    }
    cfg, err := Load(path)
    if err != nil {
        t.Fatalf("load config: %v", err)
    }
    if cfg.Run.Workspace == "${HARNESS_WORKSPACE}" {
        t.Fatalf("workspace was not expanded")
    }
}
