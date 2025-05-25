module mcp-memory/pkg/mcp/examples/database

go 1.21

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.22
	mcp-memory/pkg/mcp v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	golang.org/x/net v0.20.0 // indirect
)

replace mcp-memory/pkg/mcp => ../../
