package oauth2

import (
	"math"
)

var (
	//AccessTokenExp : アクセストークンの有効時間(秒)
	AccessTokenExp = math.MaxInt32
	//AuthorizationCodeExp : 認可コードの有効時間(秒)
	AuthorizationCodeExp = 60 * 5
	//IsRefreshEnabled : リフレッシュトークンを発行するかどうか
	IsRefreshEnabled = false
)
