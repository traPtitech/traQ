package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/ncw/swift/v2"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
)

// SwiftFileStorage OpenStack Swiftストレージ
type SwiftFileStorage struct {
	container  string
	tempURLKey string
	connection swift.Connection
	cacheDir   string
	mutexes    *utils.KeyMutex
}

// NewSwiftFileStorage 引数の情報でOpenStack Swiftストレージを生成します
func NewSwiftFileStorage(container, userName, apiKey, tenant, tenantID, authURL, tempURLKey, cacheDir string) (*SwiftFileStorage, error) {
	m := &SwiftFileStorage{
		container:  container,
		tempURLKey: tempURLKey,
		connection: swift.Connection{
			AuthUrl:  authURL,
			UserName: userName,
			ApiKey:   apiKey,
			Tenant:   tenant,
			TenantId: tenantID,
		},
		cacheDir: cacheDir,
		mutexes:  utils.NewKeyMutex(256),
	}

	ctx := context.Background()

	if err := m.connection.Authenticate(ctx); err != nil {
		return nil, err
	}

	containers, err := m.connection.ContainerNamesAll(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, v := range containers {
		if v == container {
			return m, nil
		}
	}

	return nil, fmt.Errorf("container %s is not found", container)
}

// OpenFileByKey ファイルを取得します
func (fs *SwiftFileStorage) OpenFileByKey(key string, fileType model.FileType) (reader io.ReadSeekCloser, err error) {
	cacheName := fs.getCacheFilePath(key)

	ctx := context.Background()

	if !fs.cacheable(fileType) {
		file, _, err := fs.connection.ObjectOpen(ctx, fs.container, key, true, nil)
		if err != nil {
			if err == swift.ObjectNotFound {
				return nil, ErrFileNotFound
			}
			return nil, err
		}
		return file, nil
	}

	fs.mutexes.Lock(key)
	if _, err := os.Stat(cacheName); os.IsNotExist(err) {
		defer fs.mutexes.Unlock(key)
		remote, _, err := fs.connection.ObjectOpen(ctx, fs.container, key, true, nil)
		if err != nil {
			if err == swift.ObjectNotFound {
				return nil, ErrFileNotFound
			}
			return nil, err
		}

		// save cache
		file, err := os.OpenFile(cacheName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666) // ファイルが存在していた場合はエラーにしてremoteを返す
		if err != nil {
			return remote, nil
		}

		if _, err := io.Copy(file, remote); err != nil {
			file.Close()
			_ = os.Remove(cacheName)
			return nil, err
		}

		_, _ = file.Seek(0, 0)
		return file, nil
	}
	fs.mutexes.Unlock(key)

	// from cache
	reader, err = os.Open(cacheName)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return reader, nil
}

// SaveByKey srcの内容をkeyで指定されたファイルに書き込みます
func (fs *SwiftFileStorage) SaveByKey(src io.Reader, key, name, contentType string, fileType model.FileType) (err error) {
	if fs.cacheable(fileType) {
		cacheName := fs.getCacheFilePath(key)

		file, fe := os.Create(cacheName)
		if fe == nil {
			defer func() {
				file.Close()
				if err != nil {
					_ = os.Remove(cacheName)
				}
			}()
			src = io.TeeReader(src, file)
		}
	}

	_, err = fs.connection.ObjectPut(context.Background(), fs.container, key, src, true, "", contentType, swift.Headers{
		"Content-Disposition": fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name)),
	})
	return
}

// DeleteByKey ファイルを削除します
func (fs *SwiftFileStorage) DeleteByKey(key string, _ model.FileType) (err error) {
	err = fs.connection.ObjectDelete(context.Background(), fs.container, key)
	if err != nil {
		if err == swift.ObjectNotFound {
			return ErrFileNotFound
		}
		return err
	}

	// delete cache
	cacheName := fs.getCacheFilePath(key)
	if _, err := os.Stat(cacheName); err == nil {
		_ = os.Remove(cacheName)
	}
	return nil
}

// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。
func (fs *SwiftFileStorage) GenerateAccessURL(key string, fileType model.FileType) (string, error) {
	if !fs.cacheable(fileType) && len(fs.tempURLKey) > 0 {
		if _, err := os.Stat(fs.getCacheFilePath(key)); os.IsNotExist(err) {
			return fs.connection.ObjectTempUrl(fs.container, key, fs.tempURLKey, "GET", time.Now().Add(5*time.Minute)), nil
		}
	}
	return "", nil
}

func (fs *SwiftFileStorage) getCacheFilePath(key string) string {
	return fs.cacheDir + "/" + key
}

func (fs *SwiftFileStorage) cacheable(fileType model.FileType) bool {
	return fileType == model.FileTypeIcon || fileType == model.FileTypeStamp || fileType == model.FileTypeThumbnail
}
