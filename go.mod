module github.com/vexxhost/migratekit

go 1.23.0

toolchain go1.24.6

// Replace with local SHA module for cross-module access
replace github.com/vexxhost/migratekit-sha => ./source/current/sha

// Replace with local Volume Daemon module for cross-module access
replace github.com/vexxhost/migratekit-volume-daemon => ./source/current/volume-daemon

require (
	github.com/vexxhost/migratekit-sha v0.0.0-00010101000000-000000000000
	github.com/vexxhost/migratekit-volume-daemon v0.0.0-00010101000000-000000000000
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/apache/cloudstack-go v2.4.1+incompatible
	github.com/fsnotify/fsnotify v1.9.0
	github.com/gin-gonic/gin v1.10.1
	github.com/go-sql-driver/mysql v1.8.1
	github.com/google/uuid v1.6.0
	github.com/gophercloud/gophercloud/v2 v2.8.0
	github.com/gorilla/mux v1.8.1
	github.com/gosimple/slug v1.15.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.10.0
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.6
	github.com/thediveo/enumflag/v2 v2.0.7
	github.com/vmware/govmomi v0.52.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.30.2
	libguestfs.org/libnbd v1.20.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp v0.0.0-20250103183323-7d7fa50e5329 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
