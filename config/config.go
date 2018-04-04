package config

import (
	"log"
	"os"
)

var (
	// DatabaseUserName データベースのユーザー名
	DatabaseUserName = os.Getenv("MARIADB_USERNAME")
	// DatabasePassword データベースのパスワード
	DatabasePassword = os.Getenv("MARIADB_PASSWORD")
	// DatabaseHostName データベースのホスト名
	DatabaseHostName = os.Getenv("MARIADB_HOSTNAME")
	// DatabaseName データベース名
	DatabaseName = os.Getenv("MARIADB_DATABASE")

	// OSUserName OpenStack Swift準拠のオブジェクトストレージのユーザー名
	OSUserName = os.Getenv("OS_USERNAME")
	// OSPassword OpenStack Swift準拠のオブジェクトストレージのパスワード
	OSPassword = os.Getenv("OS_PASSWORD")
	// OSTenantName OpenStack Swift準拠のオブジェクトストレージのテナント名
	OSTenantName = os.Getenv("OS_TENANT_NAME")
	// OSTenantID OpenStack Swift準拠のオブジェクトストレージのテナントID
	OSTenantID = os.Getenv("OS_TENANT_ID")
	// OSContainer OpenStack Swift準拠のオブジェクトストレージのコンテナ名
	OSContainer = os.Getenv("OS_CONTAINER")
	// OSAuthURL OpenStack Swift準拠のオブジェクトストレージのAPIエンドポイント
	OSAuthURL = os.Getenv("OS_AUTH_URL")

	// TRAQOrigin traQサーバーのhttpオリジン
	TRAQOrigin = os.Getenv("TRAQ_ORIGIN")
	// Port traQサーバーのhttp公開ポート
	Port = os.Getenv("TRAQ_PORT")
	// LocalStorageDir ローカルストレージのディレクトリ
	LocalStorageDir = os.Getenv("TRAQ_LOCAL_STORAGE")

	// RS256PublicKeyFile OpenID Connect用RS256の公開鍵ファイル
	RS256PublicKeyFile = os.Getenv("TRAQ_RS256_PUBLIC_KEY")
	// RS256PrivateKeyFile OpenID Connect用のRS256の秘密鍵ファイル
	RS256PrivateKeyFile = os.Getenv("TRAQ_RS256_PRIVATE_KEY")

	// FirebaseServiceAccountJSONFile FirebaseのサービスアカウントJSONファイル
	FirebaseServiceAccountJSONFile = os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")

	// ImageMagickConverterExec ImageMagickの実行ファイル
	ImageMagickConverterExec = os.Getenv("IMAGEMAGICK_EXEC")
)

func init() {
	if len(DatabaseUserName) == 0 {
		DatabaseUserName = "root"
	}
	if len(DatabasePassword) == 0 {
		DatabasePassword = "password"
	}
	if len(DatabaseHostName) == 0 {
		DatabaseHostName = "127.0.0.1"
	}
	if len(DatabaseName) == 0 {
		DatabaseName = "traq"
	}

	if len(Port) == 0 {
		Port = "3000"
	}

	if len(RS256PublicKeyFile) > 0 && len(RS256PrivateKeyFile) == 0 {
		log.Fatal("env 'TRAQ_RS256_PUBLIC_KEY' is set, but 'TRAQ_RS256_PRIVATE_KEY' isn't")
	}
	if len(RS256PublicKeyFile) == 0 && len(RS256PrivateKeyFile) > 0 {
		log.Fatal("env 'TRAQ_RS256_PRIVATE_KEY' is set, but 'TRAQ_RS256_PUBLIC_KEY' isn't")
	}
}
