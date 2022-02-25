package ogp

import (
	"errors"
	"net/url"
	"sync"
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
	wg          sync.WaitGroup
	sfGroup     singleflight.Group
}

func NewServiceImpl(repo repository.Repository, logger *zap.Logger) (Service, error) {
	return &ServiceImpl{
		repo:   repo,
		logger: logger,
	}, nil
}

func (s *ServiceImpl) Start() error {
	s.cachePurger = jitterbug.New(time.Hour*24, &jitterbug.Uniform{
		Min: time.Hour * 23,
	})
	go func() {
		for range s.cachePurger.C {
			s.wg.Add(1)
			if err := s.repo.DeleteStaleOgpCache(); err != nil {
				s.logger.Error("an error occurred while deleting stale ogp caches", zap.Error(err))
			}
			s.wg.Done()
		}
	}()

	s.logger.Info("OGP service started")
	return nil
}

func (s *ServiceImpl) Shutdown() error {
	s.cachePurger.Stop()
	s.wg.Wait()
	s.logger.Info("OGP service shutdown")
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

	shouldUpdateCache := err == nil &&
		time.Now().After(cache.ExpiresAt)
	shouldCreateCache := err != nil

	if !shouldUpdateCache && !shouldCreateCache && err == nil {
		if cache.Valid {
			return &cache.Content, time.Until(cache.ExpiresAt), nil
		}
		// キャッシュがヒットしたがネガティブキャッシュだった
		return nil, time.Until(cache.ExpiresAt), nil
	}

	og, meta, err := parser.ParseMetaForURL(url)
	if err == parser.ErrClient || err == parser.ErrParse || err == parser.ErrNetwork || err == parser.ErrContentTypeNotSupported {
		// 4xxエラー、パースエラー、名前解決などのネットワークエラーの場合はネガティブキャッシュを作成
		if shouldUpdateCache {
			updateErr := s.repo.UpdateOgpCache(cacheURL, nil)
			if updateErr != nil {
				return nil, time.Duration(0), updateErr
			}
		} else if shouldCreateCache {
			_, createErr := s.repo.CreateOgpCache(cacheURL, nil)
			if createErr != nil {
				return nil, time.Duration(0), createErr
			}
		}
		return nil, CacheDuration, nil
	} else if err != nil {
		// このパスは5xxエラーなのでクライアント側キャッシュつけない
		return nil, time.Duration(0), nil
	}

	content := parser.MergeDefaultPageMetaAndOpenGraph(og, meta)

	if shouldUpdateCache {
		err = s.repo.UpdateOgpCache(cacheURL, content)
		if err != nil {
			return nil, time.Duration(0), err
		}
	} else if shouldCreateCache {
		_, err = s.repo.CreateOgpCache(cacheURL, content)
		if err != nil {
			return nil, time.Duration(0), err
		}
	}

	return content, CacheDuration, nil
}
