package impl

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
)

// DefaultStore : OAuth2用のデフォルトのデータストア(mysql)
type DefaultStore struct{}

// GetClient : クライアントIDからクライアントを取得します
func (*DefaultStore) GetClient(id string) (*oauth2.Client, error) {
	oc, err := model.GetOAuth2ClientByClientID(id)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, oauth2.ErrClientNotFound
		default:
			return nil, err
		}
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
		CreatorID:    uuid.FromStringOrNil(oc.CreatorID),
		Secret:       oc.Secret,
		RedirectURI:  oc.RedirectURI,
		Scopes:       scope,
	}
	return client, nil
}

// GetClientsByUser : 指定した登録者のクライアントを全て取得します
func (*DefaultStore) GetClientsByUser(userID uuid.UUID) ([]*oauth2.Client, error) {
	cs, err := model.GetOAuth2ClientsByUser(userID.String())
	if err != nil {
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
			CreatorID:    uuid.FromStringOrNil(v.CreatorID),
			Secret:       v.Secret,
			RedirectURI:  v.RedirectURI,
			Scopes:       scope,
		}
	}

	return clients, nil
}

// SaveClient : クライアントを保存します
func (*DefaultStore) SaveClient(client *oauth2.Client) error {
	oc := &model.OAuth2Client{
		ID:           client.ID,
		Name:         client.Name,
		Description:  client.Description,
		Confidential: client.Confidential,
		CreatorID:    client.CreatorID.String(),
		Secret:       client.Secret,
		RedirectURI:  client.RedirectURI,
		Scopes:       client.Scopes.String(),
	}
	return oc.Create()
}

// UpdateClient : クライアント情報を更新します
func (*DefaultStore) UpdateClient(client *oauth2.Client) error {
	oc := &model.OAuth2Client{
		ID:           client.ID,
		Name:         client.Name,
		Description:  client.Description,
		Confidential: client.Confidential,
		CreatorID:    client.CreatorID.String(),
		Secret:       client.Secret,
		RedirectURI:  client.RedirectURI,
		Scopes:       client.Scopes.String(),
	}
	return oc.Update()
}

// DeleteClient : クライアントを削除します
func (*DefaultStore) DeleteClient(id string) error {
	oc, err := model.GetOAuth2ClientByClientID(id)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil
		default:
			return err
		}
	}

	return oc.Delete()
}

// SaveAuthorize : 認可データを保存します
func (*DefaultStore) SaveAuthorize(data *oauth2.AuthorizeData) error {
	oa := &model.OAuth2Authorize{
		Code:                data.Code,
		ClientID:            data.ClientID,
		UserID:              data.UserID.String(),
		ExpiresIn:           data.ExpiresIn,
		RedirectURI:         data.RedirectURI,
		Scopes:              data.Scopes.String(),
		OriginalScopes:      data.OriginalScopes.String(),
		CodeChallenge:       data.CodeChallenge,
		CodeChallengeMethod: data.CodeChallengeMethod,
		Nonce:               data.Nonce,
		CreatedAt:           data.CreatedAt,
	}
	return oa.Create()
}

// GetAuthorize : 認可コードから認可データを取得します
func (*DefaultStore) GetAuthorize(code string) (*oauth2.AuthorizeData, error) {
	oa, err := model.GetOAuth2Authorize(code)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, oauth2.ErrAuthorizeNotFound
		default:
			return nil, err
		}
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
		UserID:              uuid.FromStringOrNil(oa.UserID),
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
func (*DefaultStore) DeleteAuthorize(code string) error {
	oa, err := model.GetOAuth2Authorize(code)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil
		default:
			return err
		}
	}

	return oa.Delete()
}

// SaveToken : トークンを保存します
func (*DefaultStore) SaveToken(token *oauth2.Token) error {
	ot := &model.OAuth2Token{
		ID:           token.ID.String(),
		ClientID:     token.ClientID,
		UserID:       token.UserID.String(),
		RedirectURI:  token.RedirectURI,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Scopes:       token.Scopes.String(),
		ExpiresIn:    token.ExpiresIn,
		CreatedAt:    token.CreatedAt,
	}
	return ot.Create()
}

// GetTokenByID : トークンIDからトークンを取得します
func (*DefaultStore) GetTokenByID(id uuid.UUID) (*oauth2.Token, error) {
	ot, err := model.GetOAUth2TokenByID(id.String())
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, oauth2.ErrTokenNotFound
		default:
			return nil, nil
		}
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           uuid.FromStringOrNil(ot.ID),
		ClientID:     ot.ClientID,
		UserID:       uuid.FromStringOrNil(ot.UserID),
		RedirectURI:  ot.RedirectURI,
		AccessToken:  ot.AccessToken,
		RefreshToken: ot.RefreshToken,
		CreatedAt:    ot.CreatedAt,
		ExpiresIn:    ot.ExpiresIn,
		Scopes:       scope,
	}
	return token, nil

}

// GetTokenByAccess : アクセストークンからトークンを取得します
func (*DefaultStore) GetTokenByAccess(access string) (*oauth2.Token, error) {
	ot, err := model.GetOAuth2TokenByAccess(access)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, oauth2.ErrTokenNotFound
		default:
			return nil, nil
		}
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           uuid.FromStringOrNil(ot.ID),
		ClientID:     ot.ClientID,
		UserID:       uuid.FromStringOrNil(ot.UserID),
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
func (*DefaultStore) DeleteTokenByAccess(access string) error {
	ot, err := model.GetOAuth2TokenByAccess(access)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil
		default:
			return err
		}
	}

	return ot.Delete()
}

// GetTokenByRefresh : リフレッシュトークンからトークンを取得します
func (*DefaultStore) GetTokenByRefresh(refresh string) (*oauth2.Token, error) {
	ot, err := model.GetOAuth2TokenByRefresh(refresh)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, oauth2.ErrTokenNotFound
		default:
			return nil, nil
		}
	}

	scope, err := oauth2.SplitAndValidateScope(ot.Scopes)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		ID:           uuid.FromStringOrNil(ot.ID),
		ClientID:     ot.ClientID,
		UserID:       uuid.FromStringOrNil(ot.UserID),
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
func (*DefaultStore) DeleteTokenByRefresh(refresh string) error {
	ot, err := model.GetOAuth2TokenByRefresh(refresh)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil
		default:
			return err
		}
	}

	return ot.Delete()
}

// GetTokensByUser : 指定したユーザーのトークンを全て取得します
func (*DefaultStore) GetTokensByUser(userID uuid.UUID) ([]*oauth2.Token, error) {
	ts, err := model.GetOAuth2TokenByUser(userID.String())
	if err != nil {
		return nil, err
	}

	tokens := make([]*oauth2.Token, len(ts))
	for i, v := range ts {
		scope, err := oauth2.SplitAndValidateScope(v.Scopes)
		if err != nil {
			return nil, err
		}

		tokens[i] = &oauth2.Token{
			ID:           uuid.FromStringOrNil(v.ID),
			ClientID:     v.ClientID,
			UserID:       uuid.FromStringOrNil(v.UserID),
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
func (*DefaultStore) DeleteTokenByUser(userID uuid.UUID) error {
	ts, err := model.GetOAuth2TokenByUser(userID.String())
	if err != nil {
		return err
	}

	for _, v := range ts {
		if err := v.Delete(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTokenByClient : 指定したクライアントのトークンを全て削除します
func (*DefaultStore) DeleteTokenByClient(clientID string) error {
	ts, err := model.GetOAuth2TokenByClient(clientID)
	if err != nil {
		return err
	}

	for _, v := range ts {
		if err := v.Delete(); err != nil {
			return err
		}
	}
	return nil
}
