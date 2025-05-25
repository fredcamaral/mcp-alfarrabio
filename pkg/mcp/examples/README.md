# MCP-Go Examples

This directory contains runnable examples demonstrating various features and use cases of the MCP-Go library.

## Examples

### [Calculator](calculator/)
A simple calculator MCP server that demonstrates:
- Basic tool registration
- Parameter validation with JSON Schema
- Error handling
- Mathematical operations

### [File Browser](file-browser/)
A file system browser that shows:
- Multiple tool coordination
- Resource management
- Security best practices
- Advanced schema validation

### [Database Query](database/)
Database integration example featuring:
- SQL query execution
- Connection pooling
- Transaction management
- Security considerations

### [API Gateway](api-gateway/)
HTTP API integration demonstrating:
- External API calls
- Authentication handling
- Rate limiting
- Response transformation

### [AI Assistant](ai-assistant/)
A comprehensive AI assistant implementation demonstrating:
- Multi-tool integration (web search, code execution, file management)
- Advanced tool chaining with data flow
- Context preservation and memory management
- Self-analysis and learning capabilities
- Intelligent suggestion generation

## Running the Examples

Each example can be run standalone:

```bash
cd calculator
go run main.go
```

Or built and installed:

```bash
go build -o calculator
./calculator
```

## Testing with Claude Desktop

To test an example with Claude Desktop:

1. Build the example:
   ```bash
   cd calculator
   go build -o calculator
   ```

2. Add to your Claude Desktop configuration (`claude_desktop_config.json`):
   ```json
   {
     "mcpServers": {
       "calculator": {
         "command": "/path/to/examples/calculator/calculator"
       }
     }
   }
   ```

3. Restart Claude Desktop and the server will be available!

## Creating Your Own Examples

When creating new examples:

1. Create a new directory for your example
2. Include a `README.md` explaining what the example demonstrates
3. Keep the code focused on demonstrating specific features
4. Include error handling and comments
5. Add configuration examples if needed

## Learning Path

We recommend exploring the examples in this order:

1. **Calculator** - Start here to understand the basics
2. **File Browser** - Learn about resources and security
3. **Database Query** - Understand external integrations
4. **API Gateway** - See HTTP and authentication patterns
5. **AI Assistant** - Explore advanced patterns

Each example builds on concepts from the previous ones, providing a structured learning experience.

## Contributing

If you have an interesting use case, we welcome example contributions! Please ensure your example:

- Demonstrates a unique use case or pattern
- Is well-documented with clear comments
- Follows Go best practices
- Includes a README explaining the example
- Has been tested with an MCP client

Happy coding! 🚀