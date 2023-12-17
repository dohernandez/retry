package retry_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dohernandez/errors"
	"github.com/stretchr/testify/require"

	"github.com/dohernandez/retry"
)

func TestUntilSuccess(t *testing.T) {
	t.Parallel()

	t.Run("no retry", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilSuccess(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				return nil
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
		)
		require.NoError(t, err)

		require.Equal(t, 1, times)
	})

	t.Run("retry until stopped", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilSuccess(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				return errors.New("error")
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
		)
		require.NoError(t, err)

		require.Equal(t, 10, times)
	})

	t.Run("retry until recognized error", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilSuccess(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				if times == 5 {
					return errors.New("recognized error")
				}

				return errors.New("error")
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
			retry.WithOnError(func(err error) bool {
				return errors.Is(err, errors.New("recognized error"))
			}),
		)
		require.Error(t, err)
		require.Equal(t, "recognized error", err.Error())

		require.Equal(t, 5, times)
	})
}

func TestUntilFail(t *testing.T) {
	t.Parallel()

	t.Run("no retry", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilFail(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				return errors.New("error")
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
		)
		require.Error(t, err)
		require.Equal(t, "error", err.Error())

		require.Equal(t, 1, times)
	})

	t.Run("retry until stopped", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilFail(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				return nil
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
		)
		require.NoError(t, err)

		require.Equal(t, 10, times)
	})

	t.Run("retry while recognized error", func(t *testing.T) {
		t.Parallel()

		var (
			stop = make(chan struct{})

			sm    sync.Mutex
			times int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)

			close(stop)
		}()

		err := retry.UntilFail(
			context.Background(),
			func(ctx context.Context) error {
				sm.Lock()
				defer sm.Unlock()

				times++

				if times < 5 {
					return errors.New("recognized error")
				}

				return errors.New("error")
			},
			backoff.NewConstantBackOff(10*time.Millisecond),
			stop,
			retry.WithOnError(func(err error) bool {
				return errors.Is(err, errors.New("recognized error"))
			}),
		)
		require.Error(t, err)
		require.Equal(t, "error", err.Error())

		require.Equal(t, 5, times)
	})
}
