module mcp-memory/pkg/mcp/examples/api-gateway

go 1.21

require (
	golang.org/x/time v0.5.0
	gopkg.in/yaml.v3 v3.0.1
	mcp-memory/pkg/mcp v0.0.0
)

require (
	github.com/gorilla/websocket v1.5.1 // indirect
	golang.org/x/net v0.20.0 // indirect
)

replace mcp-memory/pkg/mcp => ../../
