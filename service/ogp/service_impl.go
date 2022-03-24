package ogp

import (
	"errors"
	"net/url"
	"time"

	"github.com/lthibault/jitterbug/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/ogp/parser"
)

type ServiceImpl struct {
	repo   repository.Repository
	logger *zap.Logger

	cachePurger *jitterbug.Ticker
	serviceDone chan struct{}
	purgerDone  chan struct{}
	sfGroup     singleflight.Group
}

func NewServiceImpl(repo repository.Repository, logger *zap.Logger) (Service, error) {
	s := &ServiceImpl{
		repo:   repo,
		logger: logger,

		cachePurger: jitterbug.New(time.Hour*24, &jitterbug.Uniform{
			Min: time.Hour * 23,
		}),
		serviceDone: make(chan struct{}),
		purgerDone:  make(chan struct{}),
	}
	if err := s.start(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ServiceImpl) start() error {
	go func() {
		defer close(s.purgerDone)
		for {
			select {
			case _, ok := <-s.cachePurger.C:
				if !ok {
					return
				}
				if err := s.repo.DeleteStaleOgpCache(); err != nil {
					s.logger.Error("an error occurred while deleting stale ogp caches", zap.Error(err))
				}
			case <-s.serviceDone:
				return
			}
		}
	}()

	s.logger.Info("OGP service started")
	return nil
}

func (s *ServiceImpl) Shutdown() error {
	s.cachePurger.Stop()
	close(s.serviceDone)
	<-s.purgerDone
	return nil
}

func (s *ServiceImpl) GetMeta(url *url.URL) (ogp *model.Ogp, expiresIn time.Duration, err error) {
	type cacheResult struct {
		ogp       *model.Ogp
		expiresIn time.Duration
	}

	crInt, err, _ := s.sfGroup.Do(url.String(), func() (interface{}, error) {
		ogp, expiresIn, err := s.getMeta(url)
		return cacheResult{ogp: ogp, expiresIn: expiresIn}, err
	})
	cr, ok := crInt.(cacheResult)
	if !ok {
		return nil, time.Duration(0), errors.New("assertion to cacheResult failed")
	}
	return cr.ogp, cr.expiresIn, err
}

func (s *ServiceImpl) getMeta(url *url.URL) (ogp *model.Ogp, expiresIn time.Duration, err error) {
	cacheURL := url.String()
	cache, err := s.repo.GetOgpCache(cacheURL)
	if err != nil && err != repository.ErrNotFound {
		return nil, 0, err
	}

	now := time.Now()
	isCacheHit := err == nil && now.Before(cache.ExpiresAt)
	isCacheExpired := err == nil && !now.Before(cache.ExpiresAt)
	if isCacheHit {
		if cache.Valid {
			// 通常のキャッシュヒット
			return &cache.Content, time.Until(cache.ExpiresAt), nil
		} else {
			// ネガティブキャッシュヒット
			return nil, time.Until(cache.ExpiresAt), nil
		}
	}
	if isCacheExpired {
		if err := s.repo.DeleteOgpCache(cacheURL); err != nil && err != repository.ErrNotFound {
			return nil, 0, err
		}
	}

	// キャッシュが存在しなかったので、リクエストを飛ばす
	og, meta, err := parser.ParseMetaForURL(url)

	if err != nil {
		switch err {
		case parser.ErrClient, parser.ErrParse, parser.ErrNetwork, parser.ErrContentTypeNotSupported:
			// 4xxエラー、パースエラー、名前解決などのネットワークエラーの場合はネガティブキャッシュを作成
			_, createErr := s.repo.CreateOgpCache(cacheURL, nil)
			if createErr != nil {
				return nil, 0, createErr
			}
			return nil, CacheDuration, nil
		default:
			// このパスは5xxエラーなのでクライアント側キャッシュつけない
			return nil, 0, nil
		}
	}

	// リクエストが成功した場合はキャッシュを作成
	content := parser.MergeDefaultPageMetaAndOpenGraph(og, meta)
	_, err = s.repo.CreateOgpCache(cacheURL, content)
	if err != nil {
		return nil, 0, err
	}

	return content, CacheDuration, nil
}
