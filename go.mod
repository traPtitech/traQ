module github.com/traPtitech/traQ

require (
	cloud.google.com/go v0.92.3 // indirect
	cloud.google.com/go/firestore v1.5.0 // indirect
	cloud.google.com/go/profiler v0.1.0
	cloud.google.com/go/storage v1.16.0 // indirect
	firebase.google.com/go v3.13.0+incompatible
	github.com/NYTimes/gziphandler v1.1.1
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/blendle/zapdriver v1.3.1
	github.com/bluele/gcache v0.0.2
	github.com/boz/go-throttle v0.0.0-20160922054636-fdc4eab740c1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/disintegration/imaging v1.6.2
	github.com/dyatlov/go-opengraph v0.0.0-20210112100619-dae8665a5b09
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/gavv/httpexpect/v2 v2.3.1
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.0.0
	github.com/go-gormigrate/gormigrate/v2 v2.0.0
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/wire v0.5.0
	github.com/gorilla/websocket v1.4.2
	github.com/hajimehoshi/go-mp3 v0.3.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jakobvarmose/go-qidenticon v0.0.0-20170128000056-5c327fb4e74a
	github.com/json-iterator/go v1.1.11
	github.com/labstack/echo/v4 v4.5.0
	github.com/leandro-lugaresi/hub v1.1.1
	github.com/lthibault/jitterbug/v2 v2.2.2
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/motoki317/go-waveform v0.0.3
	github.com/ncw/swift v1.0.53
	github.com/olivere/elastic/v7 v7.0.27
	github.com/orcaman/writerseeker v0.0.0-20200621085525-1d3f536ff85e
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.29.0 // indirect
	github.com/prometheus/procfs v0.7.1 // indirect
	github.com/sapphi-red/midec v0.5.2
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/exp v0.0.0-20210715201039-d37aa40e8013
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/time v0.0.0-20210611083556-38a9dc6acbc6 // indirect
	google.golang.org/api v0.54.0
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/mysql v1.1.2
	gorm.io/gorm v1.21.13
)

replace github.com/blendle/zapdriver v1.3.1 => github.com/wtks/zapdriver v1.3.1-patch.0

go 1.16
