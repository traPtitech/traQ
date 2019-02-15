package impl

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2"
	"time"
)

// DefaultStore : OAuth2用のデフォルトのデータストア(mysql)
type DefaultStore struct {
	db *gorm.DB
}

// NewDefaultStore OAuth2用のデフォルトのデータストアを生成します
func NewDefaultStore(db *gorm.DB) (*DefaultStore, error) {
	s := &DefaultStore{
		db: db,
	}
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(&OAuth2Client{}, &OAuth2Authorize{}, &OAuth2Token{}).Error; err != nil {
		return nil, fmt.Errorf("failed to sync Table schema: %v", err)
	}
	return s, nil
}

// OAuth2Authorize OAuth2 認可データの構造体
type OAuth2Authorize struct {
	Code                string    `gorm:"type:varchar(36);primary_key"`
	ClientID            string    `gorm:"type:char(36)"`
	UserID              uuid.UUID `gorm:"type:char(36)"`
	ExpiresIn           int
	RedirectURI         string    `gorm:"type:text"`
	Scopes              string    `gorm:"type:text"`
	OriginalScopes      string    `gorm:"type:text"`
	CodeChallenge       string    `gorm:"type:varchar(128)"`
	CodeChallengeMethod string    `gorm:"type:text"`
	Nonce               string    `gorm:"type:text"`
	CreatedAt           time.Time `gorm:"precision:6"`
}

// TableName OAuth2Authorizeのテーブル名
func (*OAuth2Authorize) TableName() string {
	return "oauth2_authorizes"
}

// OAuth2Client OAuth2 クライアント構造体
type OAuth2Client struct {
	ID           string `gorm:"type:char(36);primary_key"`
	Name         string `gorm:"type:varchar(32)"`
	Description  string `gorm:"type:text"`
	Confidential bool
	CreatorID    uuid.UUID  `gorm:"type:char(36)"`
	Secret       string     `gorm:"type:varchar(36)"`
	RedirectURI  string     `gorm:"type:text"`
	Scopes       string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"precision:6"`
	UpdatedAt    time.Time  `gorm:"precision:6"`
	DeletedAt    *time.Time `gorm:"precision:6"`
}

// TableName OAuth2Clientのテーブル名
func (*OAuth2Client) TableName() string {
	return "oauth2_clients"
}

// OAuth2Token : OAuth2 トークンの構造体
type OAuth2Token struct {
	ID           uuid.UUID `gorm:"type:char(36);primary_key"`
	ClientID     string    `gorm:"type:char(36)"`
	UserID       uuid.UUID `gorm:"type:char(36)"`
	RedirectURI  string    `gorm:"type:text"`
	AccessToken  string    `gorm:"type:varchar(36);unique"`
	RefreshToken string    `gorm:"type:varchar(36);unique"`
	Scopes       string    `gorm:"type:text"`
	ExpiresIn    int
	CreatedAt    time.Time  `gorm:"precision:6"`
	DeletedAt    *time.Time `gorm:"precision:6"`
}

// TableName : OAuth2Tokenのテーブル名
func (*OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// GetClient : クライアントIDからクライアントを取得します
func (s *DefaultStore) GetClient(id string) (*oauth2.Client, error) {
	oc := &OAuth2Client{}
	if err := s.db.Where(OAuth2Client{ID: id}).Take(oc).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, oauth2.ErrClientNotFound
		}
		return nil, err
	}

	scope, err := oauth2.SplitAndValidateScope(oc.Scopes)
	if err != nil {
		return nil, err
	}

	client := &oauth2.Client{
		ID:           oc.ID,
		Name:         oc.Name,
		Description:  oc.Description,
		Confidential: oc.Confidential,
		CreatorID:    oc.CreatorID,
		Secret:       oc.Secret,
		RedirectURI:  oc.RedirectURI,
		Scopes:       scope,
	}
	return client, nil
}

// GetClientsByUser : 指定した登録者のクライアントを全て取得します
func (s *DefaultStore) GetClientsByUser(userID uuid.UUID) ([]*oauth2.Client, error) {
	cs := make([]*OAuth2Client, 0)
	if err := s.db.Where(&OAuth2Client{CreatorID: userID}).Find(&cs).Error; err != nil {
		return nil, err
	}

	clients := make([]*oauth2.Client, len(cs))
	for i, v := range cs {
		scope, err := oauth2.SplitAndValidateScope(v.Scopes)
		if err != nil {
			return nil, err
		}

		clients[i] = &oauth2.Client{
			ID:           v.ID,
			Name:         v.Name,
			Description:  v.Description,
			Confidential: v.Confidential,
			CreatorID:    v.CreatorID,
			Secret:       v.Secret,
			RedirectURI:  v.RedirectURI,
			Scopes:       scope,
		}
	}

	return clients, nil
}

// SaveClient : クライアントを保存します
func (s *DefaultStore) SaveClient(client *oauth2.Client) error {
	oc := &OAuth2Client{
		ID:           client.ID,
		Name:         client.Name,
		Description:  client.Description,
		Confidential: client.Confidential,
		CreatorID:    client.CreatorID,
		Secret:       client.Secret,
		RedirectURI:  client.RedirectURI,
		Scopes:       client.Scopes.String(),
	}
	return s.db.Create(oc).Error
}

// UpdateClient : クライアント情報を更新します
func (s *DefaultStore) UpdateClient(client *oauth2.Client) error {
	return s.db.Model(&OAuth2Client{ID: client.ID}).Updates(map[string]interface{}{
		"name":         client.Name,
		"description":  client.Description,
		"confidential": client.Confidential,
		"creator_id":   client.CreatorID,
		"secret":       client.Secret,
		"redirect_uri": client.RedirectURI,
		"scopes":       client.Scopes.String(),
	}).Error
}

// DeleteClient : クライアントを削除します
func (s *DefaultStore) DeleteClient(id string) error {
	return s.db.Delete(&OAuth2Client{ID: id}).Error
}

// SaveAuthorize : 認可データを保存します
func (s *DefaultStore) SaveAuthorize(data *oauth2.AuthorizeData) error {
	oa := &OAuth2Authorize{
		Code:                data.Code,
		ClientID:            data.ClientID,
		UserID:              data.UserID,
		ExpiresIn:           data.ExpiresIn,
		RedirectURI:         data.RedirectURI,
		Scopes:              data.Scopes.String(),
		OriginalScopes:      data.OriginalScopes.String(),
		CodeChallenge:       data.CodeChallenge,
		CodeChallengeMethod: data.CodeChallengeMethod,
		Nonce:               data.Nonce,
		CreatedAt:           data.CreatedAt,
	}
	return s.db.Create(oa).Error
}

// GetAuthorize : 認可コードから認可データを取得します
func (s *DefaultStore) GetAuthorize(code string) (*oauth2.AuthorizeData, error) {
	oa := &OAuth2Authorize{}
	if err := s.db.Where(&OAuth2Authorize{Code: code}).Take(oa).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, oauth2.ErrAuthorizeNotFound
		}
		return nil, err
	}

	scope, err := oauth2.SplitAndValidateScope(oa.Scopes)
	if err != nil {
		return nil, err
	}
	origScope, err := oauth2.SplitAndValidateScope(oa.OriginalScopes)
	if err != nil {
		return nil, err
	}

	data := &oauth2.AuthorizeData{
		Code:                oa.Code,
		ClientID:            oa.ClientID,
		UserID:              oa.UserID,
		CreatedAt:           oa.CreatedAt,
		ExpiresIn:           oa.ExpiresIn,
		RedirectURI:         oa.RedirectURI,
		Scopes:              scope,
		OriginalScopes:      origScope,
		CodeChallenge:       oa.CodeChallenge,
		CodeChallengeMethod: oa.CodeChallengeMethod,
		Nonce:               oa.Nonce,
	}
	return data, nil
}

// DeleteAuthorize : 認可コードから認可データを削除します
func (s *DefaultStore) DeleteAuthorize(code string) error {
	return s.db.Where(&OAuth2Authorize{Code: code}).Delete(&OAuth2Authorize{}).Error
}

// SaveToken : トークンを保存します
func (s *DefaultStore) SaveToken(token *oauth2.Token) error {
	ot := &OAuth2Token{
		ID:           token.ID,
		ClientID:     token.ClientID,
		UserID:       token.UserID,
		RedirectURI:  token.RedirectURI,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Scopes:       token.Scopes.String(),
		ExpiresIn:    token.ExpiresIn,
		CreatedAt:    token.CreatedAt,
	}
	return s.db.Create(ot).Error
}

// GetTokenByID : トークンIDからトークンを取得します
func (s *DefaultStore) GetTokenByID(id uuid.UUID) (*oauth2.Token, error) {
	ot := &OAuth2Token{}
	if err := s.db.Where(&OAuth2Token{ID: id}).Take(ot).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, oauth2.ErrTokenNotFound
		}
		return nil, nil
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           ot.ID,
		ClientID:     ot.ClientID,
		UserID:       ot.UserID,
		RedirectURI:  ot.RedirectURI,
		AccessToken:  ot.AccessToken,
		RefreshToken: ot.RefreshToken,
		CreatedAt:    ot.CreatedAt,
		ExpiresIn:    ot.ExpiresIn,
		Scopes:       scope,
	}
	return token, nil
}

// DeleteTokenByID : トークンIDからトークンを削除します
func (s *DefaultStore) DeleteTokenByID(id uuid.UUID) error {
	return s.db.Where(&OAuth2Token{ID: id}).Delete(&OAuth2Token{}).Error
}

// GetTokenByAccess : アクセストークンからトークンを取得します
func (s *DefaultStore) GetTokenByAccess(access string) (*oauth2.Token, error) {
	ot := &OAuth2Token{}
	if err := s.db.Where(&OAuth2Token{AccessToken: access}).Take(ot).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, oauth2.ErrTokenNotFound
		}
		return nil, nil
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           ot.ID,
		ClientID:     ot.ClientID,
		UserID:       ot.UserID,
		RedirectURI:  ot.RedirectURI,
		AccessToken:  ot.AccessToken,
		RefreshToken: ot.RefreshToken,
		CreatedAt:    ot.CreatedAt,
		ExpiresIn:    ot.ExpiresIn,
		Scopes:       scope,
	}
	return token, nil
}

// DeleteTokenByAccess : アクセストークンからトークンを削除します
func (s *DefaultStore) DeleteTokenByAccess(access string) error {
	return s.db.Where(&OAuth2Token{AccessToken: access}).Delete(&OAuth2Token{}).Error
}

// GetTokenByRefresh : リフレッシュトークンからトークンを取得します
func (s *DefaultStore) GetTokenByRefresh(refresh string) (*oauth2.Token, error) {
	ot := &OAuth2Token{}
	if err := s.db.Where(&OAuth2Token{RefreshToken: refresh}).Take(ot).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, oauth2.ErrTokenNotFound
		}
		return nil, nil
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           ot.ID,
		ClientID:     ot.ClientID,
		UserID:       ot.UserID,
		RedirectURI:  ot.RedirectURI,
		AccessToken:  ot.AccessToken,
		RefreshToken: ot.RefreshToken,
		CreatedAt:    ot.CreatedAt,
		ExpiresIn:    ot.ExpiresIn,
		Scopes:       scope,
	}
	return token, nil
}

// DeleteTokenByRefresh : リフレッシュトークンからトークンを削除します
func (s *DefaultStore) DeleteTokenByRefresh(refresh string) error {
	return s.db.Where(&OAuth2Token{RefreshToken: refresh}).Delete(&OAuth2Token{}).Error
}

// GetTokensByUser : 指定したユーザーのトークンを全て取得します
func (s *DefaultStore) GetTokensByUser(userID uuid.UUID) ([]*oauth2.Token, error) {
	ts := make([]*OAuth2Token, 0)
	if err := s.db.Where(&OAuth2Token{UserID: userID}).Find(&ts).Error; err != nil {
		return nil, err
	}

	tokens := make([]*oauth2.Token, len(ts))
	for i, v := range ts {
		scope, err := oauth2.SplitAndValidateScope(v.Scopes)
		if err != nil {
			return nil, err
		}

		tokens[i] = &oauth2.Token{
			ID:           v.ID,
			ClientID:     v.ClientID,
			UserID:       v.UserID,
			RedirectURI:  v.RedirectURI,
			AccessToken:  v.AccessToken,
			RefreshToken: v.RefreshToken,
			CreatedAt:    v.CreatedAt,
			ExpiresIn:    v.ExpiresIn,
			Scopes:       scope,
		}
	}

	return tokens, nil
}

// DeleteTokenByUser : 指定したユーザーのトークンを全て削除します
func (s *DefaultStore) DeleteTokenByUser(userID uuid.UUID) error {
	return s.db.Where(&OAuth2Token{UserID: userID}).Delete(&OAuth2Token{}).Error
}

// DeleteTokenByClient : 指定したクライアントのトークンを全て削除します
func (s *DefaultStore) DeleteTokenByClient(clientID string) error {
	return s.db.Where(&OAuth2Token{ClientID: clientID}).Delete(&OAuth2Token{}).Error
}
