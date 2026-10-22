package unread

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/traPtitech/traQ/repository"
)

type cleanerRepository struct {
	repository.Repository
	deleteUnreads func(context.Context, int) (int64, error)
}

func (r *cleanerRepository) DeleteDeactivatedUserUnreads(ctx context.Context, limit int) (int64, error) {
	return r.deleteUnreads(ctx, limit)
}

func TestServiceSchedule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		repo := &cleanerRepository{deleteUnreads: func(ctx context.Context, limit int) (int64, error) {
			require.NoError(t, ctx.Err())
			assert.Equal(t, deleteBatchSize, limit)
			calls++
			return 0, nil
		}}
		s := NewService(repo, zap.NewNop())
		defer s.Shutdown(context.Background())

		synctest.Wait()
		assert.Zero(t, calls, "must not delete at startup")
		time.Sleep(deleteMinInterval - time.Nanosecond)
		synctest.Wait()
		assert.Zero(t, calls)
		time.Sleep(deleteInterval - deleteMinInterval + time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, 1, calls)
		time.Sleep(deleteInterval)
		synctest.Wait()
		assert.Equal(t, 2, calls)

		require.NoError(t, s.Shutdown(context.Background()))
		time.Sleep(deleteInterval)
		synctest.Wait()
		assert.Equal(t, 2, calls, "must not delete after shutdown")
	})
}

func TestServiceDeleteUnreadsBatches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		batches := []int64{deleteBatchSize, 250, 0}
		repo := &cleanerRepository{deleteUnreads: func(_ context.Context, limit int) (int64, error) {
			assert.Equal(t, deleteBatchSize, limit)
			deleted := batches[calls]
			calls++
			return deleted, nil
		}}
		s := &Service{repo: repo}
		done := make(chan error, 1)
		go func() { done <- s.deleteUnreads(context.Background()) }()

		synctest.Wait()
		assert.Equal(t, 1, calls)
		time.Sleep(deleteBatchPause - time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, 1, calls, "must pause before the next batch")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, 2, calls)
		time.Sleep(deleteBatchPause)
		synctest.Wait()
		assert.Equal(t, 3, calls)
		require.NoError(t, <-done)
	})
}

func TestServiceDeleteUnreadsTimeout(t *testing.T) {
	for _, duringQuery := range []bool{true, false} {
		name := "batch pause"
		if duringQuery {
			name = "query"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started := time.Now()
				calls := 0
				repo := &cleanerRepository{deleteUnreads: func(ctx context.Context, _ int) (int64, error) {
					calls++
					deadline, ok := ctx.Deadline()
					assert.True(t, ok)
					assert.Equal(t, started.Add(deleteTimeout), deadline)
					if duringQuery {
						<-ctx.Done()
						return 0, ctx.Err()
					}
					time.Sleep(deleteTimeout - deleteBatchPause/2)
					return deleteBatchSize, nil
				}}
				s := &Service{repo: repo}

				err := s.deleteUnreads(context.Background())
				require.ErrorIs(t, err, context.DeadlineExceeded)
				assert.Equal(t, deleteTimeout, time.Since(started))
				assert.Equal(t, 1, calls, "must not start another batch after the time limit")
			})
		})
	}
}

func TestServiceContinuesAfterTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		repo := &cleanerRepository{deleteUnreads: func(ctx context.Context, _ int) (int64, error) {
			calls++
			if calls == 1 {
				<-ctx.Done()
				return 0, ctx.Err()
			}
			assert.NoError(t, ctx.Err(), "the next run must have a fresh context")
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)
			assert.Equal(t, time.Now().Add(deleteTimeout), deadline)
			return 0, nil
		}}
		core, logs := observer.New(zap.InfoLevel)
		s := &Service{repo: repo, logger: zap.New(core)}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.run(ctx, ticks)
		}()

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 1, calls)
		time.Sleep(deleteTimeout)
		synctest.Wait()
		assert.Equal(t, 1, calls, "must wait for the next tick instead of retrying immediately")
		require.Equal(t, 1, logs.Len())
		assert.Equal(t, zap.InfoLevel, logs.All()[0].Level, "the time limit is an expected interruption")

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 2, calls)
		assert.Equal(t, 1, logs.Len())
		close(ticks)
		<-done
	})
}

func TestServiceDeleteUnreadsCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		repo := &cleanerRepository{deleteUnreads: func(context.Context, int) (int64, error) {
			calls++
			return deleteBatchSize, nil
		}}
		s := &Service{repo: repo}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- s.deleteUnreads(ctx) }()

		synctest.Wait()
		assert.Equal(t, 1, calls)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		assert.Equal(t, 1, calls, "cancellation must interrupt the batch pause")
		require.ErrorIs(t, s.deleteUnreads(ctx), context.Canceled)
		assert.Equal(t, 1, calls, "must not query with an already canceled context")
	})
}

func TestServiceRetriesOnNextTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deleteErr := errors.New("database unavailable")
		calls := 0
		repo := &cleanerRepository{deleteUnreads: func(context.Context, int) (int64, error) {
			calls++
			if calls == 1 {
				return 0, deleteErr
			}
			return 0, nil
		}}
		core, logs := observer.New(zap.ErrorLevel)
		s := &Service{repo: repo, logger: zap.New(core)}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.run(ctx, ticks)
		}()

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 1, calls)
		require.Equal(t, 1, logs.Len())
		assert.Equal(t, deleteErr.Error(), logs.All()[0].ContextMap()["error"])

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 2, calls)
		assert.Equal(t, 1, logs.Len())
		close(ticks)
		<-done
	})
}

func TestServiceShutdownCancelsQuery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queryStarted := make(chan struct{})
		repo := &cleanerRepository{deleteUnreads: func(ctx context.Context, _ int) (int64, error) {
			close(queryStarted)
			<-ctx.Done()
			return 0, ctx.Err()
		}}
		core, logs := observer.New(zap.ErrorLevel)
		s := NewService(repo, zap.New(core))
		<-queryStarted
		require.NoError(t, s.Shutdown(context.Background()))
		assert.Zero(t, logs.Len(), "shutdown cancellation is not a cleanup failure")
		// jitterbug.Stop does not interrupt its internal sleep.
		time.Sleep(deleteInterval)
	})
}

func TestServiceRunDoesNotOverlap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		release := make(chan struct{})
		repo := &cleanerRepository{deleteUnreads: func(context.Context, int) (int64, error) {
			calls++
			<-release
			return 0, nil
		}}
		s := &Service{repo: repo, logger: zap.NewNop()}
		ticks := make(chan time.Time, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.run(ctx, ticks)
		}()

		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 1, calls)
		ticks <- time.Now()
		synctest.Wait()
		assert.Equal(t, 1, calls, "another tick must not start a concurrent deletion")
		cancel()
		close(release)
		<-done
		assert.Equal(t, 1, calls)
	})
}
