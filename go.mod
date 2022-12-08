module github.com/envelope-zero/backend

go 1.19

require (
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-contrib/logger v0.2.5
	github.com/gin-contrib/requestid v0.0.6
	github.com/gin-gonic/gin v1.8.1
	github.com/glebarez/go-sqlite v1.20.0
	github.com/glebarez/sqlite v1.5.0
	github.com/google/uuid v1.3.0
	github.com/rs/zerolog v1.28.0
	github.com/shopspring/decimal v1.3.1
	github.com/stretchr/testify v1.8.1
	github.com/swaggo/files v0.0.0-20221208230613-cf1eeac86b11
	github.com/swaggo/gin-swagger v1.5.3
	github.com/swaggo/swag v1.8.7
	github.com/wei840222/gorm-zerolog v0.0.0-20210303025759-235c42bb33fa
	golang.org/x/text v0.5.0
	gorm.io/gorm v1.24.2
)

require github.com/gin-contrib/pprof v1.4.0

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.0 // indirect
	github.com/goccy/go-json v0.9.8 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/exp v0.0.0-20221208152030-732eee02a75a
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/tools v0.2.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.21.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.4.0 // indirect
	modernc.org/sqlite v1.20.0 // indirect
)

replace github.com/envelope-zero/backend/pkg/controllers => ./pkg/controllers

replace github.com/envelope-zero/backend/api => ./api
