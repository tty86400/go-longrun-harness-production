package observability

import (
    "context"
    "expvar"
    "fmt"
    "log/slog"
    "net/http"
    "net/http/pprof"
    "time"
)

type Metrics struct {
    Steps           *expvar.Int
    ToolCalls       *expvar.Int
    ToolFailures    *expvar.Int
    ProviderCalls   *expvar.Int
    ProviderErrors  *expvar.Int
    Summaries       *expvar.Int
    Reviews         *expvar.Int
}

func NewMetrics(namespace string) *Metrics {
    return &Metrics{
        Steps:          expvar.NewInt(namespace + "_steps"),
        ToolCalls:      expvar.NewInt(namespace + "_tool_calls"),
        ToolFailures:   expvar.NewInt(namespace + "_tool_failures"),
        ProviderCalls:  expvar.NewInt(namespace + "_provider_calls"),
        ProviderErrors: expvar.NewInt(namespace + "_provider_errors"),
        Summaries:      expvar.NewInt(namespace + "_summaries"),
        Reviews:        expvar.NewInt(namespace + "_reviews"),
    }
}

func StartServer(addr string, enablePprof bool, logger *slog.Logger) (func(context.Context) error, error) {
    if addr == "" {
        return func(context.Context) error { return nil }, nil
    }
    mux := http.NewServeMux()
    mux.Handle("/debug/vars", expvar.Handler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok\n"))
    })
    if enablePprof {
        mux.HandleFunc("/debug/pprof/", pprof.Index)
        mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
        mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
        mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
        mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
    }

    srv := &http.Server{
        Addr:              addr,
        Handler:           mux,
        ReadHeaderTimeout: 5 * time.Second,
    }

    go func() {
        logger.Info("observability server started", "addr", addr, "pprof", enablePprof)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("observability server failed", "error", err)
        }
    }()

    return func(ctx context.Context) error {
        shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        err := srv.Shutdown(shutdownCtx)
        if err == nil {
            logger.Info("observability server stopped", "addr", addr)
        }
        return err
    }, nil
}

func RunNamespace(runID string) string {
    return fmt.Sprintf("longrun_%s", runID)
}
