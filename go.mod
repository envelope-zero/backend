module github.com/envelope-zero/backend

go 1.18

require (
	github.com/gin-gonic/gin v1.7.7
	github.com/glebarez/sqlite v1.4.1
	github.com/shopspring/decimal v1.3.1
	gorm.io/driver/postgres v1.3.4
	gorm.io/gorm v1.23.4
)

require github.com/gin-contrib/requestid v0.0.3

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gin-contrib/logger v0.2.2
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/glebarez/go-sqlite v1.15.1 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.10.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.11.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.2.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.10.0 // indirect
	github.com/jackc/pgx/v4 v4.15.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/rs/zerolog v1.26.1
	github.com/stretchr/testify v1.7.1
	github.com/ugorji/go/codec v1.2.7 // indirect
	github.com/wei840222/gorm-zerolog v0.0.0-20210303025759-235c42bb33fa
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	modernc.org/libc v1.14.12 // indirect
	modernc.org/mathutil v1.4.1 // indirect
	modernc.org/memory v1.0.7 // indirect
	modernc.org/sqlite v1.15.2 // indirect
)

replace github.com/envelope-zero/backend/internal/controllers => ./internal/controllers
