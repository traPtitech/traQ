package external

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/ncw/swift"
	"io"
)

// SwiftFileManager OpenStack Swiftのファイルマネージャー構造体
type SwiftFileManager struct {
	container  string
	redirect   bool
	connection swift.Connection
}

// NewSwiftFileManager 引数の情報でファイルマネージャーを生成します
func NewSwiftFileManager(container, userName, apiKey, tenant, tenantID, authURL string, redirect bool) (*SwiftFileManager, error) {
	m := &SwiftFileManager{
		container: container,
		redirect:  redirect,
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

// OpenFileByID ファイルを取得します
func (m *SwiftFileManager) OpenFileByID(ID string) (file io.ReadCloser, err error) {
	file, _, err = m.connection.ObjectOpen(m.container, ID, true, nil)
	return
}

// WriteByID srcの内容をIDで指定されたファイルに書き込みます
func (m *SwiftFileManager) WriteByID(src io.Reader, ID, name, contentType string) (err error) {
	_, err = m.connection.ObjectPut(m.container, ID, src, true, "", contentType, swift.Headers{echo.HeaderContentDisposition: fmt.Sprintf("attachment; filename=%s", name)})
	return
}

// DeleteByID ファイルを削除します
func (m *SwiftFileManager) DeleteByID(ID string) (err error) {
	err = m.connection.ObjectDelete(m.container, ID)
	return
}

// GetRedirectURL オブジェクトストレージへのリダイレクト先URLを返します
func (m *SwiftFileManager) GetRedirectURL(ID string) string {
	if !m.redirect {
		return ""
	}

	return "" //TODO
}
