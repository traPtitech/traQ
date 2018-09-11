package storage

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
)

// InMemoryFileManager インメモリファイルマネージャー
type InMemoryFileManager struct {
	sync.RWMutex
	fileMap map[string][]byte
}

// NewInMemoryFileManager : インメモリのファイルマネージャーを生成します。主にテスト用
func NewInMemoryFileManager() *InMemoryFileManager {
	return &InMemoryFileManager{
		fileMap: make(map[string][]byte),
	}
}

// WriteByID srcの内容をIDで指定されたファイルに書き込みます
func (m *InMemoryFileManager) WriteByID(src io.Reader, ID, name, contentType string) error {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return err
	}
	m.Lock()
	m.fileMap[ID] = b
	m.Unlock()
	return nil
}

// OpenFileByID ファイルを取得します
func (m *InMemoryFileManager) OpenFileByID(ID string) (io.ReadCloser, error) {
	m.RLock()
	f, ok := m.fileMap[ID]
	m.RUnlock()
	if !ok {
		return nil, errors.New("not found")
	}
	return &closableByteReader{bytes.NewReader(f)}, nil
}

// DeleteByID ファイルを削除します
func (m *InMemoryFileManager) DeleteByID(ID string) error {
	m.Lock()
	delete(m.fileMap, ID)
	m.Unlock()
	return nil
}

// GetRedirectURL 常に空文字列を返します
func (m *InMemoryFileManager) GetRedirectURL(ID string) string {
	return ""
}

type closableByteReader struct {
	*bytes.Reader
}

// Close 何もしません
func (*closableByteReader) Close() error {
	return nil
}
