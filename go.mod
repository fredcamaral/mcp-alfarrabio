module mcp-memory

go 1.23.0

toolchain go1.24.3

require (
	github.com/amikos-tech/chroma-go v0.2.3
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/sashabaranov/go-openai v1.40.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.38.0
	mcp-memory/pkg/mcp v0.0.0-00010101000000-000000000000
)

replace mcp-memory/pkg/mcp => ./pkg/mcp

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/yalue/onnxruntime_go v1.19.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
