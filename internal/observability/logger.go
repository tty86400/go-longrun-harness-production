package observability

import (
    "io"
    "log/slog"
    "os"
    "path/filepath"
)

type LogHandle struct {
    Logger *slog.Logger
    file   *os.File
}

func NewLogger(jsonLogs bool, filePath string) (*LogHandle, error) {
    if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
        return nil, err
    }
    file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return nil, err
    }
    writer := io.MultiWriter(os.Stdout, file)
    var handler slog.Handler
    if jsonLogs {
        handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo})
    } else {
        handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo})
    }
    return &LogHandle{Logger: slog.New(handler), file: file}, nil
}

func (h *LogHandle) Close() error {
    if h == nil || h.file == nil {
        return nil
    }
    return h.file.Close()
}
