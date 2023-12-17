package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dohernandez/errors"
)

// Err is returned when the operation is retryable.
var Err = errors.New("retry")

// An operation is executing by retryOperation().
// The operation will be retried using a backoff policy if it returns an error.
type operation func(context.Context) error

// OnError checks if it should attempt a retry in case an error occurs. This feature is useful for handling
// expected errors and continuously retrying until successful.
type OnError func(error) bool

// NotifyOnError is a notify-on-error function. It receives an error returned by an operation and a
// duration representing the backoff delay, both provided by the backoff.RetryNotify whenever
// the operation failed (with an error).
type NotifyOnError func(context.Context, error, time.Duration)

// Operation executes the operation until it does not return an error and runs successfully using backoff strategy.
//
// If the operation fails, the operation can be retried if the error is recognized through the OnError.
//
// When chan stop received a signal the operation is stopped automatically.
func Operation(ctx context.Context, op operation, bo backoff.BackOff, onError OnError, notify NotifyOnError, stop <-chan struct{}) error {
	bo = backoff.WithContext(bo, ctx)

	if onError == nil {
		onError = defaultOnError()
	}

	backoffOp := func() error {
		select {
		case <-stop:
			return nil
		default:
			err := op(ctx)
			if err != nil && !onError(err) {
				return backoff.Permanent(err)
			}

			return err
		}
	}

	if notify == nil {
		return backoff.Retry(
			backoffOp,
			bo,
		)
	}

	return backoff.RetryNotify(
		backoffOp,
		bo,
		func(err error, duration time.Duration) {
			notify(ctx, err, duration)
		},
	)
}

func defaultOnError() OnError {
	return func(err error) bool {
		return errors.Is(err, Err)
	}
}
