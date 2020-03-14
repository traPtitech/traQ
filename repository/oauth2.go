package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

type UpdateClientArgs struct {
	Name         null.String
	Description  null.String
	Confidential null.Bool
	DeveloperID  uuid.NullUUID
	Secret       null.String
	CallbackURL  null.String
	Scopes       model.AccessScopes
}

// OAuth2Repository OAuth2用リポジトリ
type OAuth2Repository interface {
	// GetClient 指定したIDのクライアントを取得します
	//
	// 成功した場合、クライアントとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetClient(id string) (*model.OAuth2Client, error)
	// GetClientsByUser 指定した登録者のクライアントを全て取得します
	//
	// 成功した場合、クライアントの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetClientsByUser(userID uuid.UUID) ([]*model.OAuth2Client, error)
	// SaveClient クライアントを保存します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	SaveClient(client *model.OAuth2Client) error
	// UpdateClient クライアント情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないクライアントの場合、ErrNotFoundを返します。
	// clientIDに空文字を指定するとErrNilIDを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	UpdateClient(clientID string, args UpdateClientArgs) error
	// DeleteClient 指定したクライアントを削除します
	//
	// 成功した、或いは既に存在しない場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteClient(id string) error
	// SaveAuthorize 認可データを保存します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	SaveAuthorize(data *model.OAuth2Authorize) error
	// GetAuthorize 指定したコードの認可データを取得します
	//
	// 成功した場合、認可データとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetAuthorize(code string) (*model.OAuth2Authorize, error)
	// DeleteAuthorize 指定したコードの認可データを削除します
	//
	// 成功した、或いは既に存在しない場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteAuthorize(code string) error
	// IssueToken トークンを発行します
	//
	// 成功した場合、トークンとnilを返します。
	// DBによるエラーを返すことがあります。
	IssueToken(client *model.OAuth2Client, userID uuid.UUID, redirectURI string, scope model.AccessScopes, expire int, refresh bool) (*model.OAuth2Token, error)
	// GetTokenByID 指定したIDのトークンを取得します
	//
	// 成功した場合、トークンとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetTokenByID(id uuid.UUID) (*model.OAuth2Token, error)
	// DeleteTokenByID 指定したIDのトークンを削除します
	//
	// 成功した、或いは既に存在しない場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteTokenByID(id uuid.UUID) error
	// GetTokenByAccess 指定したアクセストークンのトークンを取得します
	//
	// 成功した場合、トークンとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetTokenByAccess(access string) (*model.OAuth2Token, error)
	// DeleteTokenByAccess 指定したアクセストークンのトークンを削除します
	//
	// 成功した、或いは既に存在しない場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteTokenByAccess(access string) error
	// GetTokenByRefresh 指定したリフレッシュトークンのトークンを取得します
	//
	// 成功した場合、トークンとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetTokenByRefresh(refresh string) (*model.OAuth2Token, error)
	// DeleteTokenByRefresh 指定したリフレッシュトークンのトークンを削除します
	//
	// 成功した、或いは既に存在しない場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteTokenByRefresh(refresh string) error
	// GetTokensByUser 指定したユーザーのトークンを全て取得します
	//
	// 成功した場合、トークンの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetTokensByUser(userID uuid.UUID) ([]*model.OAuth2Token, error)
	// DeleteTokenByUser 指定したユーザーのトークンを全て削除します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteTokenByUser(userID uuid.UUID) error
	// DeleteTokenByClient 指定したクライアントのトークンを全て削除します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteTokenByClient(clientID string) error
}
