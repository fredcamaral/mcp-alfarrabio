module mcp-memory/pkg/mcp/examples/database

go 1.21

require (
	mcp-memory/pkg/mcp v0.0.0
	github.com/lib/pq v1.10.9
	github.com/go-sql-driver/mysql v1.8.1
	github.com/mattn/go-sqlite3 v1.14.22
)

replace mcp-memory/pkg/mcp => ../../