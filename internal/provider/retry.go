package provider

import (
    "context"
    "math/rand"
    "time"

    "longrunharness/internal/config"
)

type retryingProvider struct {
    base  Provider
    retry config.RetryConfig
}

func NewRetryingProvider(base Provider, retry config.RetryConfig) Provider {
    return &retryingProvider{base: base, retry: retry}
}

func (p *retryingProvider) Name() string { return p.base.Name() }

func (p *retryingProvider) Generate(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    attempts := p.retry.MaxAttempts
    if attempts <= 1 {
        return p.base.Generate(ctx, req)
    }

    var lastErr error
    for attempt := 1; attempt <= attempts; attempt++ {
        resp, err := p.base.Generate(ctx, req)
        if err == nil {
            return resp, nil
        }
        lastErr = err
        if ctx.Err() != nil {
            return ChatResponse{}, ctx.Err()
        }
        if !IsRetryable(err) || attempt == attempts {
            return ChatResponse{}, err
        }
        backoff := time.Duration(p.retry.InitialBackoffMS) * time.Millisecond
        if backoff <= 0 {
            backoff = 500 * time.Millisecond
        }
        for i := 1; i < attempt; i++ {
            backoff *= 2
            maxBackoff := time.Duration(p.retry.MaxBackoffMS) * time.Millisecond
            if maxBackoff > 0 && backoff > maxBackoff {
                backoff = maxBackoff
                break
            }
        }
        if p.retry.JitterMS > 0 {
            backoff += time.Duration(rand.Intn(p.retry.JitterMS)) * time.Millisecond
        }
        timer := time.NewTimer(backoff)
        select {
        case <-ctx.Done():
            timer.Stop()
            return ChatResponse{}, ctx.Err()
        case <-timer.C:
        }
    }
    return ChatResponse{}, lastErr
}
