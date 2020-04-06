module github.com/traPtitech/traQ

require (
	cloud.google.com/go v0.56.0
	cloud.google.com/go/firestore v1.1.1 // indirect
	firebase.google.com/go v3.12.0+incompatible
	github.com/NYTimes/gziphandler v1.1.1
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/blendle/zapdriver v1.3.1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/disintegration/imaging v1.6.2
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fogleman/gg v1.1.0 // indirect
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jakobvarmose/go-qidenticon v0.0.0-20170128000056-5c327fb4e74a
	github.com/jinzhu/gorm v1.9.12
	github.com/json-iterator/go v1.1.9
	github.com/labstack/echo/v4 v4.1.14
	github.com/leandro-lugaresi/hub v1.1.0
	github.com/lib/pq v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/ncw/swift v1.0.50
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.7
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	google.golang.org/api v0.21.0
	gopkg.in/go-playground/webhooks.v5 v5.13.0
	gopkg.in/gormigrate.v1 v1.6.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/square/go-jose.v2 v2.4.1 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/blendle/zapdriver v1.3.1 => github.com/wtks/zapdriver v1.3.1-patch.0

go 1.13
