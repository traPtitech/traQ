package storage

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/ncw/swift"
	"io"
)

// SwiftFileStorage OpenStack Swiftストレージ
type SwiftFileStorage struct {
	container  string
	connection swift.Connection
}

// NewSwiftFileStorage 引数の情報でOpenStack Swiftストレージを生成します
func NewSwiftFileStorage(container, userName, apiKey, tenant, tenantID, authURL string) (*SwiftFileStorage, error) {
	m := &SwiftFileStorage{
		container: container,
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
func (fs *SwiftFileStorage) SaveByKey(src io.Reader, key, name, contentType string) (err error) {
	_, err = fs.connection.ObjectPut(fs.container, key, src, true, "", contentType, swift.Headers{
		echo.HeaderContentDisposition: fmt.Sprintf("attachment; filename=%s", name),
		"Cache-Control":               "private, max-age=31536000",
	})
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
