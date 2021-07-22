package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/ioext"
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
func (fs *InMemoryFileStorage) SaveByKey(src io.Reader, key, name, contentType string, fileType model.FileType) error {
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
func (fs *InMemoryFileStorage) OpenFileByKey(key string, fileType model.FileType) (ioext.ReadSeekCloser, error) {
	fs.RLock()
	f, ok := fs.fileMap[key]
	fs.RUnlock()
	if !ok {
		return nil, ErrFileNotFound
	}
	return &closableByteReader{bytes.NewReader(f)}, nil
}

// DeleteByKey ファイルを削除します
func (fs *InMemoryFileStorage) DeleteByKey(key string, fileType model.FileType) error {
	fs.Lock()
	defer fs.Unlock()
	if _, ok := fs.fileMap[key]; !ok {
		return ErrFileNotFound
	}
	delete(fs.fileMap, key)
	return nil
}

// GenerateAccessURL "",nilを返します
func (fs *InMemoryFileStorage) GenerateAccessURL(key string, fileType model.FileType) (string, error) {
	return "", nil
}

type closableByteReader struct {
	*bytes.Reader
}

// Close 何もしません
func (*closableByteReader) Close() error {
	return nil
}
