package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseServer struct {
	db       *sql.DB
	dbType   string
	txMutex  sync.RWMutex
	activeTx map[string]*sql.Tx
	config   DatabaseConfig
}

type DatabaseConfig struct {
	Driver          string
	ConnectionString string
	MaxRows         int
	QueryTimeout    time.Duration
	ReadOnly        bool
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	RowCount int            `json:"rowCount"`
}

type TableInfo struct {
	Name    string `json:"name"`
	Schema  string `json:"schema,omitempty"`
	Type    string `json:"type"`
}

type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	IsNullable   bool   `json:"isNullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	IsPrimaryKey bool   `json:"isPrimaryKey"`
}

// toolHandler implements protocol.ToolHandler
type toolHandler struct {
	handler func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func (t *toolHandler) Handle(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return t.handler(ctx, params)
}

// resourceHandler implements server.ResourceHandler
type resourceHandler struct {
	handler func(ctx context.Context, uri string) ([]protocol.Content, error)
}

func (r *resourceHandler) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
	return r.handler(ctx, uri)
}

func NewDatabaseServer(config DatabaseConfig) (*DatabaseServer, error) {
	db, err := sql.Open(config.Driver, config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DatabaseServer{
		db:       db,
		dbType:   config.Driver,
		activeTx: make(map[string]*sql.Tx),
		config:   config,
	}, nil
}

func (s *DatabaseServer) Close() error {
	s.txMutex.Lock()
	defer s.txMutex.Unlock()

	// Rollback all active transactions
	for id, tx := range s.activeTx {
		_ = tx.Rollback()
		delete(s.activeTx, id)
	}

	return s.db.Close()
}

func (s *DatabaseServer) executeQuery(ctx context.Context, query string, args []interface{}, txID string) (*QueryResult, error) {
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	// Security check for read-only mode
	if s.config.ReadOnly && s.isWriteQuery(query) {
		return nil, fmt.Errorf("write operations are not allowed in read-only mode")
	}

	var rows *sql.Rows
	var err error

	if txID != "" {
		s.txMutex.RLock()
		tx, exists := s.activeTx[txID]
		s.txMutex.RUnlock()
		if !exists {
			return nil, fmt.Errorf("transaction %s not found", txID)
		}
		rows, err = tx.QueryContext(ctx, query, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Prepare result
	result := &QueryResult{
		Columns: columns,
		Rows:    [][]interface{}{},
	}

	// Scan rows
	rowCount := 0
	for rows.Next() && rowCount < s.config.MaxRows {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to proper types
		row := make([]interface{}, len(values))
		for i, v := range values {
			row[i] = s.convertValue(v)
		}

		result.Rows = append(result.Rows, row)
		rowCount++
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	result.RowCount = rowCount
	return result, nil
}

func (s *DatabaseServer) convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	case nil:
		return nil
	default:
		return val
	}
}

func (s *DatabaseServer) isWriteQuery(query string) bool {
	query = strings.TrimSpace(strings.ToUpper(query))
	writeKeywords := []string{"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE"}
	
	for _, keyword := range writeKeywords {
		if strings.HasPrefix(query, keyword) {
			return true
		}
	}
	return false
}

func (s *DatabaseServer) getTables(ctx context.Context, schema string) ([]TableInfo, error) {
	var query string
	var args []interface{}

	switch s.dbType {
	case "postgres":
		query = `
			SELECT table_schema, table_name, table_type
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		`
		if schema != "" {
			query += " AND table_schema = $1"
			args = append(args, schema)
		}
		query += " ORDER BY table_schema, table_name"

	case "mysql":
		query = `
			SELECT table_schema, table_name, table_type
			FROM information_schema.tables
			WHERE table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		`
		if schema != "" {
			query += " AND table_schema = ?"
			args = append(args, schema)
		}
		query += " ORDER BY table_schema, table_name"

	case "sqlite", "sqlite3":
		query = `
			SELECT name, type
			FROM sqlite_master
			WHERE type IN ('table', 'view')
			AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`
	default:
		return nil, fmt.Errorf("unsupported database type: %s", s.dbType)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		
		if s.dbType == "sqlite" || s.dbType == "sqlite3" {
			if err := rows.Scan(&table.Name, &table.Type); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&table.Schema, &table.Name, &table.Type); err != nil {
				return nil, err
			}
		}
		
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func (s *DatabaseServer) getTableSchema(ctx context.Context, tableName string, schema string) ([]ColumnInfo, error) {
	var query string
	var args []interface{}

	switch s.dbType {
	case "postgres":
		query = `
			SELECT 
				c.column_name,
				c.data_type,
				c.is_nullable = 'YES',
				c.column_default,
				COALESCE(
					(SELECT true FROM information_schema.table_constraints tc
					 JOIN information_schema.key_column_usage kcu 
					 ON tc.constraint_name = kcu.constraint_name
					 WHERE tc.table_schema = c.table_schema
					 AND tc.table_name = c.table_name
					 AND kcu.column_name = c.column_name
					 AND tc.constraint_type = 'PRIMARY KEY'), false
				) as is_primary_key
			FROM information_schema.columns c
			WHERE c.table_name = $1
		`
		args = append(args, tableName)
		if schema != "" {
			query += " AND c.table_schema = $2"
			args = append(args, schema)
		}
		query += " ORDER BY c.ordinal_position"

	case "mysql":
		query = `
			SELECT 
				column_name,
				data_type,
				is_nullable = 'YES',
				column_default,
				column_key = 'PRI'
			FROM information_schema.columns
			WHERE table_name = ?
		`
		args = append(args, tableName)
		if schema != "" {
			query += " AND table_schema = ?"
			args = append(args, schema)
		}
		query += " ORDER BY ordinal_position"

	case "sqlite", "sqlite3":
		// SQLite requires pragma
		query = fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", s.dbType)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	
	if s.dbType == "sqlite" || s.dbType == "sqlite3" {
		// SQLite pragma returns: cid, name, type, notnull, dflt_value, pk
		for rows.Next() {
			var cid int
			var col ColumnInfo
			var notnull int
			var pk int
			var dfltValue sql.NullString
			
			if err := rows.Scan(&cid, &col.Name, &col.DataType, &notnull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			
			col.IsNullable = notnull == 0
			col.IsPrimaryKey = pk > 0
			if dfltValue.Valid {
				col.DefaultValue = dfltValue.String
			}
			
			columns = append(columns, col)
		}
	} else {
		for rows.Next() {
			var col ColumnInfo
			var defaultValue sql.NullString
			
			if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &defaultValue, &col.IsPrimaryKey); err != nil {
				return nil, err
			}
			
			if defaultValue.Valid {
				col.DefaultValue = defaultValue.String
			}
			
			columns = append(columns, col)
		}
	}

	return columns, rows.Err()
}

func (s *DatabaseServer) beginTransaction(ctx context.Context, readOnly bool) (string, error) {
	opts := &sql.TxOptions{
		ReadOnly: readOnly,
	}
	
	tx, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Generate transaction ID
	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	
	s.txMutex.Lock()
	s.activeTx[txID] = tx
	s.txMutex.Unlock()

	return txID, nil
}

func (s *DatabaseServer) commitTransaction(txID string) error {
	s.txMutex.Lock()
	tx, exists := s.activeTx[txID]
	if exists {
		delete(s.activeTx, txID)
	}
	s.txMutex.Unlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", txID)
	}

	return tx.Commit()
}

func (s *DatabaseServer) rollbackTransaction(txID string) error {
	s.txMutex.Lock()
	tx, exists := s.activeTx[txID]
	if exists {
		delete(s.activeTx, txID)
	}
	s.txMutex.Unlock()

	if !exists {
		return fmt.Errorf("transaction %s not found", txID)
	}

	return tx.Rollback()
}

func main() {
	var (
		driver     = flag.String("driver", "sqlite3", "Database driver (sqlite3, postgres, mysql)")
		connString = flag.String("conn", "file::memory:?cache=shared", "Database connection string")
		maxRows    = flag.Int("max-rows", 1000, "Maximum rows to return per query")
		timeout    = flag.Duration("timeout", 30*time.Second, "Query timeout")
		readOnly   = flag.Bool("read-only", false, "Enable read-only mode")
	)
	flag.Parse()

	config := DatabaseConfig{
		Driver:           *driver,
		ConnectionString: *connString,
		MaxRows:          *maxRows,
		QueryTimeout:     *timeout,
		ReadOnly:         *readOnly,
	}

	dbServer, err := NewDatabaseServer(config)
	if err != nil {
		log.Fatalf("Failed to create database server: %v", err)
	}
	defer dbServer.Close()

	// Create MCP server
	mcpServer := server.NewServer("database-query", "1.0.0")

	// Register tools
	mcpServer.AddTool(protocol.Tool{
		Name:        "query",
		Description: "Execute a SQL query with parameter binding",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "SQL query to execute"
				},
				"params": {
					"type": "array",
					"description": "Query parameters for prepared statements",
					"items": {}
				},
				"transactionId": {
					"type": "string",
					"description": "Optional transaction ID to execute query within"
				}
			},
			"required": ["query"]
		}`),
	}, &toolHandler{
		handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			query, ok := params["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query parameter is required")
			}
			
			var queryParams []interface{}
			if p, ok := params["params"]; ok {
				if paramsArray, ok := p.([]interface{}); ok {
					queryParams = paramsArray
				}
			}
			
			txID := ""
			if tid, ok := params["transactionId"].(string); ok {
				txID = tid
			}

			return dbServer.executeQuery(ctx, query, queryParams, txID)
		},
	})

	mcpServer.AddTool(protocol.Tool{
		Name:        "beginTransaction",
		Description: "Begin a new database transaction",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"readOnly": {
					"type": "boolean",
					"description": "Whether this is a read-only transaction",
					"default": false
				}
			}
		}`),
	}, &toolHandler{
		handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			readOnly := false
			if ro, ok := params["readOnly"].(bool); ok {
				readOnly = ro
			}

			txID, err := dbServer.beginTransaction(ctx, readOnly)
			if err != nil {
				return nil, err
			}

			return map[string]string{"transactionId": txID}, nil
		},
	})

	mcpServer.AddTool(protocol.Tool{
		Name:        "commitTransaction",
		Description: "Commit a database transaction",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"transactionId": {
					"type": "string",
					"description": "Transaction ID to commit"
				}
			},
			"required": ["transactionId"]
		}`),
	}, &toolHandler{
		handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			txID, ok := params["transactionId"].(string)
			if !ok {
				return nil, fmt.Errorf("transactionId parameter is required")
			}

			if err := dbServer.commitTransaction(txID); err != nil {
				return nil, err
			}

			return map[string]string{"status": "committed"}, nil
		},
	})

	mcpServer.AddTool(protocol.Tool{
		Name:        "rollbackTransaction",
		Description: "Rollback a database transaction",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"transactionId": {
					"type": "string",
					"description": "Transaction ID to rollback"
				}
			},
			"required": ["transactionId"]
		}`),
	}, &toolHandler{
		handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			txID, ok := params["transactionId"].(string)
			if !ok {
				return nil, fmt.Errorf("transactionId parameter is required")
			}

			if err := dbServer.rollbackTransaction(txID); err != nil {
				return nil, err
			}

			return map[string]string{"status": "rolled back"}, nil
		},
	})

	// Register resources
	mcpServer.AddResource(protocol.Resource{
		Name:        "tables",
		Description: "List all tables in the database",
		URI:         "database://tables",
		MimeType:    "application/json",
	}, &resourceHandler{
		handler: func(ctx context.Context, uri string) ([]protocol.Content, error) {
			tables, err := dbServer.getTables(ctx, "")
			if err != nil {
				return nil, err
			}
			
			jsonBytes, err := json.Marshal(tables)
			if err != nil {
				return nil, err
			}
			
			return []protocol.Content{{
				Type: "text",
				Text: string(jsonBytes),
			}}, nil
		},
	})

	mcpServer.AddResource(protocol.Resource{
		Name:        "schema/{table}",
		Description: "Get schema information for a specific table",
		URI:         "database://schema/{table}",
		MimeType:    "application/json",
	}, &resourceHandler{
		handler: func(ctx context.Context, uri string) ([]protocol.Content, error) {
			// Parse table name from URI (simplified version)
			// In production, you'd properly parse the URI pattern
			parts := strings.Split(uri, "/")
			if len(parts) < 2 {
				return nil, fmt.Errorf("invalid URI format")
			}
			tableName := parts[len(parts)-1]
			
			columns, err := dbServer.getTableSchema(ctx, tableName, "")
			if err != nil {
				return nil, err
			}
			
			jsonBytes, err := json.Marshal(columns)
			if err != nil {
				return nil, err
			}
			
			return []protocol.Content{{
				Type: "text",
				Text: string(jsonBytes),
			}}, nil
		},
	})

	// Create stdio transport
	stdio := transport.NewStdioTransport(os.Stdin, os.Stdout)
	
	// Set transport
	mcpServer.SetTransport(stdio)
	
	// Start server
	log.Println("Database MCP server starting...")
	if err := mcpServer.Start(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}