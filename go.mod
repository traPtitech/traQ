module github.com/traPtitech/traQ

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/firestore v1.1.1 // indirect
	firebase.google.com/go v3.13.0+incompatible
	github.com/NYTimes/gziphandler v1.1.1
	github.com/blendle/zapdriver v1.3.1
	github.com/bluele/gcache v0.0.0-20190518031135-bc40bd653833
	github.com/boz/go-throttle v0.0.0-20160922054636-fdc4eab740c1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/disintegration/imaging v1.6.2
	github.com/dyatlov/go-opengraph v0.0.0-20180429202543-816b6608b3c8
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fogleman/gg v1.1.0 // indirect
	github.com/gavv/httpexpect/v2 v2.2.0
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v3.4.0+incompatible
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/mock v1.5.0
	github.com/google/wire v0.5.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jakobvarmose/go-qidenticon v0.0.0-20170128000056-5c327fb4e74a
	github.com/jinzhu/gorm v1.9.16
	github.com/json-iterator/go v1.1.10
	github.com/labstack/echo/v4 v4.1.14
	github.com/leandro-lugaresi/hub v1.1.1
	github.com/lib/pq v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/ncw/swift v1.0.53
	github.com/olivere/elastic/v7 v7.0.23
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.7
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/api v0.43.0
	gopkg.in/gormigrate.v1 v1.6.0
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/square/go-jose.v2 v2.4.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/blendle/zapdriver v1.3.1 => github.com/wtks/zapdriver v1.3.1-patch.0

go 1.16
