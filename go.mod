module github.com/traPtitech/traQ

require (
	cloud.google.com/go v0.50.0
	cloud.google.com/go/firestore v1.0.0 // indirect
	firebase.google.com/go v3.11.0+incompatible
	github.com/NYTimes/gziphandler v1.1.1
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/blendle/zapdriver v1.3.1
	github.com/denisenkom/go-mssqldb v0.0.0-20190806190131-db2462fef53b // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/disintegration/imaging v1.6.2
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fogleman/gg v1.1.0 // indirect
	github.com/gavv/httpexpect/v2 v2.0.2
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/pprof v0.0.0-20190723021845-34ac40c74b70 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jakobvarmose/go-qidenticon v0.0.0-20170128000056-5c327fb4e74a
	github.com/jinzhu/gorm v1.9.11
	github.com/json-iterator/go v1.1.8
	github.com/labstack/echo/v4 v4.1.11
	github.com/leandro-lugaresi/hub v1.1.0
	github.com/lib/pq v1.2.0 // indirect
	github.com/ncw/swift v1.0.49
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/prometheus/client_golang v1.3.0
	github.com/skip2/go-qrcode v0.0.0-20190110000554-dc11ecdae0a9
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7
	golang.org/x/exp v0.0.0-20191030013958-a1ab85dbe136
	golang.org/x/net v0.0.0-20190918130420-a8b05e9114ab // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191128015809-6d18c012aee9 // indirect
	google.golang.org/api v0.15.0
	google.golang.org/appengine v1.6.2 // indirect
	google.golang.org/grpc v1.23.1 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/go-playground/webhooks.v5 v5.13.0
	gopkg.in/gormigrate.v1 v1.6.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/yaml.v2 v2.2.7
)

replace github.com/blendle/zapdriver v1.3.1 => github.com/wtks/zapdriver v1.3.1-patch.0

go 1.13
