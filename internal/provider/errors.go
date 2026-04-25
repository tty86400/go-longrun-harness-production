package provider

import "errors"

type RetryableError struct {
    Err error
}

func (e RetryableError) Error() string { return e.Err.Error() }
func (e RetryableError) Unwrap() error { return e.Err }

func IsRetryable(err error) bool {
    var target RetryableError
    return errors.As(err, &target)
}
