// Package retry provides a simple mechanism to retry operations using a backoff strategy.
package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dohernandez/errors"
)

// Err is returned when the operation is retryable.
var Err = errors.New("retry")

// OnError checks if it should attempt a retry in case an error occurs. This feature is useful for handling
// expected errors and continuously retrying until successful.
type OnError func(error) bool

// NotifyOnError is a notify-on-error function. It receives an error returned by an operation and a
// duration representing the backoff delay, both provided by the backoff.RetryNotify whenever
// the operation failed (with an error).
type NotifyOnError func(context.Context, error, time.Duration)

type options struct {
	onError OnError
	notify  NotifyOnError
}

// Option is a function that configures an option.
type Option func(*options)

// WithOnError configures the OnError function.
func WithOnError(onError OnError) Option {
	return func(o *options) {
		o.onError = onError
	}
}

// WithNotifyOnError configures the NotifyOnError function.
func WithNotifyOnError(notify NotifyOnError) Option {
	return func(o *options) {
		o.notify = notify
	}
}

// UntilSuccess executes the operation periodically using backoff strategy until success or
// until the context is canceled or
// until chan stop received a signal.
//
// If the operation fails, the error can be accessed; configuring using
// WithNotifyOnError.
// If the operation fails, the operation can be stopped if the error is recognized; configuring using
// WithOnError.
func UntilSuccess(
	ctx context.Context,
	op func(context.Context) error,
	bo backoff.BackOff,
	stop <-chan struct{},
	opts ...Option,
) error {
	var o options

	for _, opt := range opts {
		opt(&o)
	}

	bo = backoff.WithContext(bo, ctx)

	backoffOp := func() error {
		select {
		case <-stop:
			return nil
		default:
			err := op(ctx)
			if err != nil && o.onError != nil && o.onError(err) {
				return backoff.Permanent(err)
			}

			return err
		}
	}

	if o.notify == nil {
		return backoff.Retry(
			backoffOp,
			bo,
		)
	}

	return backoff.RetryNotify(
		backoffOp,
		bo,
		func(err error, duration time.Duration) {
			o.notify(ctx, err, duration)
		},
	)
}

// UntilFail executes the operation periodically using backoff strategy until fail or
// until the context is canceled or
// until chan stop received a signal.
//
// If the operation fails, the error can be accessed; configuring using
// WithNotifyOnError.
// If the operation fails, the operation can still be retried if the error is recognized; configuring using
// WithOnError.
func UntilFail(
	ctx context.Context,
	op func(context.Context) error,
	bo backoff.BackOff,
	stop <-chan struct{},
	opts ...Option,
) error {
	var o options

	for _, opt := range opts {
		opt(&o)
	}

	bo = backoff.WithContext(bo, ctx)

	backoffOp := func() error {
		select {
		case <-stop:
			return nil
		default:
			err := op(ctx)
			if err == nil {
				return Err
			}

			if o.onError != nil && o.onError(err) {
				return err
			}

			return backoff.Permanent(err)
		}
	}

	if o.notify == nil {
		return backoff.Retry(
			backoffOp,
			bo,
		)
	}

	return backoff.RetryNotify(
		backoffOp,
		bo,
		func(err error, duration time.Duration) {
			if errors.Is(err, Err) {
				return
			}

			o.notify(ctx, err, duration)
		},
	)
}
