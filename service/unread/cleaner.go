package unread

import (
	"context"
	"errors"
	"time"

	"github.com/lthibault/jitterbug/v2"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
)

const (
	deleteInitialMinDelay = time.Minute
	deleteInitialMaxDelay = 5 * time.Minute
	deleteInterval        = 24 * time.Hour
	deleteMinInterval     = 23 * time.Hour
	deleteTimeout         = 10 * time.Minute
	deleteBatchSize       = 5000
	deleteBatchPause      = 100 * time.Millisecond
)

// Service 凍結ユーザーの未読を定期的に削除するサービス。
type Service struct {
	repo   repository.MessageRepository
	logger *zap.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

// NewService 未読清掃サービスを生成して起動します。
func NewService(repo repository.Repository, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		repo:   repo,
		logger: logger.Named("unread_cleaner"),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go func() {
		defer close(s.done)

		// 再起動が多くても清掃が先送りされ続けないよう、初回だけ短く待機する。
		delay := (&jitterbug.Uniform{Min: deleteInitialMinDelay}).Jitter(deleteInitialMaxDelay)
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			if ctx.Err() != nil {
				return
			}
		case <-ctx.Done():
			return
		}

		s.deleteOnce(ctx)
		if ctx.Err() != nil {
			return
		}

		ticker := jitterbug.New(deleteInterval, &jitterbug.Uniform{Min: deleteMinInterval})
		defer ticker.Stop()
		s.run(ctx, ticker.C)
	}()
	return s
}

func (s *Service) run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case _, ok := <-ticks:
			if !ok || ctx.Err() != nil {
				return
			}
			s.deleteOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) deleteOnce(ctx context.Context) {
	if err := s.deleteUnreads(ctx); err != nil && ctx.Err() == nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.logger.Info("unread deletion time limit reached; remaining unreads will be deleted on the next run")
			return
		}
		s.logger.Error("failed to delete deactivated user unreads", zap.Error(err))
	}
}

func (s *Service) deleteUnreads(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		deleted, err := s.repo.DeleteDeactivatedUserUnreads(ctx, deleteBatchSize)
		if err != nil {
			return err
		}
		if deleted == 0 {
			return nil
		}

		timer := time.NewTimer(deleteBatchPause)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}

// Shutdown 清掃処理をキャンセルし、終了を待ちます。
func (s *Service) Shutdown(ctx context.Context) error {
	s.cancel()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
