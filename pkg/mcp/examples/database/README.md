# Database Query MCP Server

A production-quality Model Context Protocol (MCP) server that provides secure database query capabilities with support for multiple database types.

## Features

### Core Capabilities
- **SQL Query Execution**: Execute parameterized queries with SQL injection prevention
- **Transaction Support**: Begin, commit, and rollback database transactions
- **Table Discovery**: List all tables in the database
- **Schema Exploration**: Get detailed column information for any table
- **Multi-Database Support**: Works with SQLite, PostgreSQL, and MySQL
- **Security Features**: 
  - Prepared statement support for SQL injection prevention
  - Read-only mode for restricted access
  - Query timeout protection
  - Connection pooling with proper limits

### Supported Databases
- **SQLite** (default) - Perfect for development and testing
- **PostgreSQL** - Full support for schemas and advanced features
- **MySQL** - Complete compatibility with MySQL/MariaDB

## Installation

```bash
# Install dependencies
go mod download

# Build the server
go build -o database-server
```

## Usage

### Basic Usage (SQLite in-memory)
```bash
./database-server
```

### PostgreSQL Connection
```bash
./database-server \
  -driver postgres \
  -conn "host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable"
```

### MySQL Connection
```bash
./database-server \
  -driver mysql \
  -conn "user:password@tcp(localhost:3306)/database"
```

### Command Line Options
- `-driver` - Database driver: `sqlite3`, `postgres`, or `mysql` (default: `sqlite3`)
- `-conn` - Database connection string (default: `file::memory:?cache=shared`)
- `-max-rows` - Maximum rows to return per query (default: `1000`)
- `-timeout` - Query execution timeout (default: `30s`)
- `-read-only` - Enable read-only mode to prevent write operations (default: `false`)

## MCP Tools

### 1. `query` - Execute SQL Queries
Execute SQL queries with parameter binding for security.

**Parameters:**
- `query` (required): SQL query to execute
- `params` (optional): Array of parameters for prepared statements
- `transactionId` (optional): Execute within a specific transaction

**Example:**
```json
{
  "tool": "query",
  "arguments": {
    "query": "SELECT * FROM users WHERE age > ? AND city = ?",
    "params": [21, "New York"]
  }
}
```

### 2. `beginTransaction` - Start a Transaction
Begin a new database transaction with optional read-only mode.

**Parameters:**
- `readOnly` (optional): Whether this is a read-only transaction (default: `false`)

**Returns:**
- `transactionId`: Unique identifier for the transaction

### 3. `commitTransaction` - Commit a Transaction
Commit changes made within a transaction.

**Parameters:**
- `transactionId` (required): Transaction ID to commit

### 4. `rollbackTransaction` - Rollback a Transaction
Rollback all changes made within a transaction.

**Parameters:**
- `transactionId` (required): Transaction ID to rollback

## MCP Resources

### 1. `database://tables` - List All Tables
Returns a JSON array of all tables in the database with their metadata.

**Response format:**
```json
[
  {
    "name": "users",
    "schema": "public",
    "type": "TABLE"
  },
  {
    "name": "orders_view",
    "schema": "public", 
    "type": "VIEW"
  }
]
```

### 2. `database://schema/{table}` - Get Table Schema
Returns detailed column information for a specific table.

**Response format:**
```json
[
  {
    "name": "id",
    "dataType": "integer",
    "isNullable": false,
    "defaultValue": "nextval('users_id_seq')",
    "isPrimaryKey": true
  },
  {
    "name": "email",
    "dataType": "varchar",
    "isNullable": false,
    "isPrimaryKey": false
  }
]
```

## Security Best Practices

1. **Always use parameterized queries**: The server enforces prepared statements to prevent SQL injection
2. **Enable read-only mode**: Use `-read-only` flag when write access is not needed
3. **Set appropriate timeouts**: Configure query timeouts to prevent long-running queries
4. **Limit result sets**: Use `-max-rows` to prevent memory exhaustion from large result sets
5. **Use secure connections**: For production, always use SSL/TLS connections:
   - PostgreSQL: Add `sslmode=require` to connection string
   - MySQL: Add `tls=true` to connection string

## Example Workflow

### 1. Simple Query
```bash
# Execute a basic SELECT query
{
  "tool": "query",
  "arguments": {
    "query": "SELECT COUNT(*) as total FROM users"
  }
}
```

### 2. Transaction Example
```bash
# Begin transaction
{
  "tool": "beginTransaction",
  "arguments": {}
}
# Returns: {"transactionId": "tx_1234567890"}

# Insert data
{
  "tool": "query", 
  "arguments": {
    "query": "INSERT INTO users (name, email) VALUES (?, ?)",
    "params": ["John Doe", "john@example.com"],
    "transactionId": "tx_1234567890"
  }
}

# Commit transaction
{
  "tool": "commitTransaction",
  "arguments": {
    "transactionId": "tx_1234567890"
  }
}
```

### 3. Schema Exploration
```bash
# List all tables
{
  "resource": "database://tables"
}

# Get schema for users table
{
  "resource": "database://schema/users"
}
```

## Error Handling

The server provides detailed error messages for common issues:
- **Invalid SQL syntax**: Returns the database-specific error message
- **Connection failures**: Clear connection error details
- **Timeout errors**: Query execution time exceeded the configured timeout
- **Transaction errors**: Invalid transaction ID or transaction already completed
- **Parameter binding errors**: Mismatch between query placeholders and provided parameters

## Performance Considerations

1. **Connection Pooling**: The server maintains a connection pool with:
   - Max open connections: 25
   - Max idle connections: 5
   - Connection lifetime: 5 minutes

2. **Query Optimization**: 
   - Results are streamed to minimize memory usage
   - Large result sets are automatically limited by `-max-rows`
   - Use proper indexes in your database for optimal query performance

3. **Transaction Management**:
   - Transactions are automatically cleaned up on server shutdown
   - Long-running transactions should be avoided
   - Use read-only transactions when possible for better concurrency

## Development and Testing

For development, the default SQLite in-memory database is perfect:

```bash
# Start the server with an in-memory database
./database-server

# The database starts empty, so create some test data:
{
  "tool": "query",
  "arguments": {
    "query": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)"
  }
}

{
  "tool": "query",
  "arguments": {
    "query": "INSERT INTO users (name, email) VALUES ('Test User', 'test@example.com')"
  }
}
```

## Limitations

1. **Stored Procedures**: Limited support varies by database type
2. **Binary Data**: Binary data is converted to base64 strings
3. **Large Objects**: LOBs are not fully supported
4. **Database-specific features**: Some advanced features may not be available across all database types

## License

This example is provided as-is for demonstration purposes.