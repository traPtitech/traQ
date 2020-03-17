package storage

import (
	"bytes"
	"github.com/traPtitech/traQ/utils/ioext"
	"io"
	"io/ioutil"
	"sync"
)

// InMemoryFileStorage インメモリファイルストレージ
type InMemoryFileStorage struct {
	sync.RWMutex
	fileMap map[string][]byte
}

// NewInMemoryFileStorage インメモリのファイルストレージを生成します。主にテスト用
func NewInMemoryFileStorage() *InMemoryFileStorage {
	return &InMemoryFileStorage{
		fileMap: make(map[string][]byte),
	}
}

// SaveByKey srcの内容をkeyで指定されたファイルに書き込みます
func (fs *InMemoryFileStorage) SaveByKey(src io.Reader, key, name, contentType, fileType string) error {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return err
	}
	fs.Lock()
	fs.fileMap[key] = b
	fs.Unlock()
	return nil
}

// OpenFileByKey ファイルを取得します
func (fs *InMemoryFileStorage) OpenFileByKey(key, fileType string) (ioext.ReadSeekCloser, error) {
	fs.RLock()
	f, ok := fs.fileMap[key]
	fs.RUnlock()
	if !ok {
		return nil, ErrFileNotFound
	}
	return &closableByteReader{bytes.NewReader(f)}, nil
}

// DeleteByKey ファイルを削除します
func (fs *InMemoryFileStorage) DeleteByKey(key, fileType string) error {
	fs.Lock()
	delete(fs.fileMap, key)
	fs.Unlock()
	return nil
}

// GenerateAccessURL "",nilを返します
func (fs *InMemoryFileStorage) GenerateAccessURL(key, fileType string) (string, error) {
	return "", nil
}

type closableByteReader struct {
	*bytes.Reader
}

// Close 何もしません
func (*closableByteReader) Close() error {
	return nil
}
