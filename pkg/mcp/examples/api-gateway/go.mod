module mcp-memory/pkg/mcp/examples/api-gateway

go 1.23.0

require (
	golang.org/x/time v0.5.0
	gopkg.in/yaml.v3 v3.0.1
	mcp-memory/pkg/mcp v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
)

replace mcp-memory/pkg/mcp => ../../
