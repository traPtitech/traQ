package storage

import (
	"fmt"
	"github.com/ncw/swift"
	"github.com/traPtitech/traQ/model"
	"io"
	"time"
)

// SwiftFileStorage OpenStack Swiftストレージ
type SwiftFileStorage struct {
	container  string
	tempURLKey string
	connection swift.Connection
}

// NewSwiftFileStorage 引数の情報でOpenStack Swiftストレージを生成します
func NewSwiftFileStorage(container, userName, apiKey, tenant, tenantID, authURL, tempURLKey string) (*SwiftFileStorage, error) {
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
	}

	if err := m.connection.Authenticate(); err != nil {
		return nil, err
	}

	containers, err := m.connection.ContainerNamesAll(nil)
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
func (fs *SwiftFileStorage) OpenFileByKey(key string) (file io.ReadCloser, err error) {
	file, _, err = fs.connection.ObjectOpen(fs.container, key, true, nil)
	if err == swift.ObjectNotFound {
		return nil, ErrFileNotFound
	}
	return
}

// SaveByKey srcの内容をkeyで指定されたファイルに書き込みます
func (fs *SwiftFileStorage) SaveByKey(src io.Reader, key, name, contentType, fileType string) (err error) {
	headers := swift.Headers{
		"Content-Disposition": fmt.Sprintf("attachment; filename=%s", name),
		"Cache-Control":       "private, max-age=31536000",
		"X-TRAQ-FILE-TYPE":    fileType,
	}
	switch fileType {
	case model.FileTypeStamp, model.FileTypeIcon, model.FileTypeThumbnail:
		headers["X-TRAQ-FILE-CACHE"] = "true"
	default:
		break
	}
	_, err = fs.connection.ObjectPut(fs.container, key, src, true, "", contentType, headers)
	return
}

// DeleteByKey ファイルを削除します
func (fs *SwiftFileStorage) DeleteByKey(key string) (err error) {
	err = fs.connection.ObjectDelete(fs.container, key)
	if err == swift.ObjectNotFound {
		return ErrFileNotFound
	}
	return err
}

// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。
func (fs *SwiftFileStorage) GenerateAccessURL(key string) (string, error) {
	if len(fs.tempURLKey) > 0 {
		return fs.connection.ObjectTempUrl(fs.container, key, fs.tempURLKey, "GET", time.Now().Add(5*time.Minute)), nil
	}
	return "", nil
}
