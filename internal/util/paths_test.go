package util

import (
    "path/filepath"
    "testing"
)

func TestResolveWithinBase(t *testing.T) {
    base := t.TempDir()
    good, err := ResolveWithinBase(base, filepath.Join("sub", "file.txt"))
    if err != nil {
        t.Fatalf("good path returned err: %v", err)
    }
    if good == "" {
        t.Fatalf("good path empty")
    }
    if _, err := ResolveWithinBase(base, "../escape.txt"); err == nil {
        t.Fatalf("expected escape path to fail")
    }
    if _, err := ResolveWithinBase(base, "/tmp/absolute.txt"); err == nil {
        t.Fatalf("expected absolute path to fail")
    }
}
