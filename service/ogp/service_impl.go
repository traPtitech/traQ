package ogp

import (
	"context"
	"net/url"
	"time"

	"github.com/lthibault/jitterbug/v2"
	"github.com/motoki317/sc"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	ogpparser "github.com/traPtitech/traQ/service/ogp/parser"
)

const (
	inMemCacheSize = 1000
	inMemCacheTime = 1 * time.Minute
)

type fetchResult struct {
	ogp       *model.Ogp
	expiresAt time.Time
}

type ServiceImpl struct {
	repo   repository.Repository
	logger *zap.Logger

	cachePurger *jitterbug.Ticker
	serviceDone chan struct{}
	purgerDone  chan struct{}
	inMemCache  *sc.Cache[string, fetchResult]
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
	s.inMemCache = sc.NewMust(s.getMetaOrCreate, inMemCacheTime, inMemCacheTime, sc.WithLRUBackend(inMemCacheSize))
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
				if err := s.repo.DeleteStaleOgpCache(context.Background()); err != nil {
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

func (s *ServiceImpl) GetMeta(ctx context.Context, url *url.URL) (ogp *model.Ogp, expiresAt time.Time, err error) {
	res, err := s.inMemCache.Get(ctx, url.String())
	if err != nil {
		return nil, time.Time{}, err
	}
	return res.ogp, res.expiresAt, nil
}

// getMetaOrCreate OGP情報をDBのキャッシュから取得し、存在しなかった場合はリクエストを飛ばし新たに作成します。
func (s *ServiceImpl) getMetaOrCreate(ctx context.Context, urlStr string) (res fetchResult, err error) {
	cache, err := s.repo.GetOgpCache(ctx, urlStr)
	if err != nil && err != repository.ErrNotFound {
		return fetchResult{}, err
	}

	// インメモリキャッシュ分厳しく判定する
	now := time.Now().Add(inMemCacheTime)
	isCacheHit := err == nil && now.Before(cache.ExpiresAt)
	isCacheExpired := err == nil && !now.Before(cache.ExpiresAt)
	if isCacheHit {
		if cache.Valid {
			// 通常のキャッシュヒット
			return fetchResult{&cache.Content, cache.ExpiresAt}, nil
		}
		// ネガティブキャッシュヒット
		return fetchResult{nil, cache.ExpiresAt}, nil
	}
	if isCacheExpired {
		if err := s.repo.DeleteOgpCache(ctx, urlStr); err != nil && err != repository.ErrNotFound {
			return fetchResult{}, err
		}
	}

	// キャッシュが存在しなかったか期限切れだったので、リクエストを飛ばす
	u, err := url.Parse(urlStr)
	if err != nil {
		return fetchResult{}, err
	}
	og, meta, err := ogpparser.ParseMetaForURL(u)
	if err != nil {
		switch err {
		case ogpparser.ErrClient, ogpparser.ErrParse, ogpparser.ErrNetwork, ogpparser.ErrContentTypeNotSupported, ogpparser.ErrNotAllowed:
			// 4xxエラー、パースエラー、名前解決などのネットワークエラーの場合はネガティブキャッシュを作成
			cache, createErr := s.repo.CreateOgpCache(ctx, urlStr, nil, DefaultCacheDuration)
			if createErr != nil {
				return fetchResult{}, createErr
			}
			return fetchResult{nil, cache.ExpiresAt}, nil
		default:
			// このパスは5xxエラーなので短い期間のインメモリキャッシュに留める
			return fetchResult{nil, time.Now().Add(inMemCacheTime)}, nil
		}
	}

	// リクエストが成功した場合はキャッシュを作成
	content := ogpparser.MergeDefaultPageMetaAndOpenGraph(og, meta)
	cache, err = s.repo.CreateOgpCache(ctx, urlStr, content, DefaultCacheDuration)
	if err != nil {
		return fetchResult{}, err
	}

	return fetchResult{content, cache.ExpiresAt}, nil
}

func (s *ServiceImpl) DeleteCache(ctx context.Context, url *url.URL) error {
	err := s.repo.DeleteOgpCache(ctx, url.String())
	// キャッシュが見つからなかった場合でも、削除されてはいるので正常とみなす
	if err != nil && err != repository.ErrNotFound {
		return err
	}

	s.inMemCache.Forget(url.String())
	return nil
}
