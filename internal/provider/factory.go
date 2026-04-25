package provider

import (
    "fmt"
    "strings"

    "longrunharness/internal/config"
)

func New(cfg config.ProviderConfig) (Provider, error) {
    var base Provider
    var err error
    switch strings.ToLower(strings.TrimSpace(cfg.Kind)) {
    case "openai_compatible":
        base, err = NewOpenAICompatible(cfg)
    case "generic_json":
        base, err = NewGenericJSON(cfg)
    case "mock":
        base, err = NewMock(cfg)
    default:
        return nil, fmt.Errorf("unsupported provider kind %q", cfg.Kind)
    }
    if err != nil {
        return nil, err
    }
    return NewRetryingProvider(base, cfg.Retry), nil
}
