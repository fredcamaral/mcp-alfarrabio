# API Gateway MCP Server

An MCP server that acts as an API gateway, proxying requests to external APIs with authentication, rate limiting, and caching.

## Features

- 🔐 **Multiple Authentication Methods**: API keys, Bearer tokens, OAuth2
- 🚦 **Rate Limiting**: Per-API rate limits with configurable burst
- 💾 **Response Caching**: TTL-based caching to reduce API calls
- 🔧 **Flexible Configuration**: YAML-based configuration for easy management
- 🌐 **Multi-API Support**: Proxy to multiple APIs from a single server
- 📝 **Dynamic Tool Registration**: Each API endpoint becomes an MCP tool

## Installation

```bash
go mod download
go build -o api-gateway
```

## Configuration

Create a `config.yaml` file (or set `API_GATEWAY_CONFIG` environment variable):

### Basic Example

```yaml
apis:
  weather:
    base_url: "https://api.openweathermap.org/data/2.5"
    auth_type: "api_key"
    auth_config:
      key: "YOUR_API_KEY"
      param: "appid"
    rate_limit:
      requests_per_second: 10
      burst: 20
    cache_ttl: 5m
    endpoints:
      - name: "current"
        path: "/weather"
        method: "GET"
        description: "Get current weather for a city"
        parameters:
          - name: "q"
            type: "string"
            required: true
            description: "City name"
            in: "query"
          - name: "units"
            type: "string"
            required: false
            description: "Temperature units (metric, imperial)"
            in: "query"
```

### GitHub API Example

```yaml
apis:
  github:
    base_url: "https://api.github.com"
    auth_type: "bearer"
    auth_config:
      token: "YOUR_GITHUB_TOKEN"
    rate_limit:
      requests_per_second: 30
      burst: 60
    cache_ttl: 1m
    headers:
      Accept: "application/vnd.github.v3+json"
    endpoints:
      - name: "get_user"
        path: "/users/{username}"
        method: "GET"
        description: "Get GitHub user information"
        parameters:
          - name: "username"
            type: "string"
            required: true
            description: "GitHub username"
            in: "path"
      
      - name: "list_repos"
        path: "/users/{username}/repos"
        method: "GET"
        description: "List user repositories"
        parameters:
          - name: "username"
            type: "string"
            required: true
            description: "GitHub username"
            in: "path"
          - name: "sort"
            type: "string"
            required: false
            description: "Sort by: created, updated, pushed, full_name"
            in: "query"
          - name: "per_page"
            type: "number"
            required: false
            description: "Results per page (max 100)"
            in: "query"
      
      - name: "create_issue"
        path: "/repos/{owner}/{repo}/issues"
        method: "POST"
        description: "Create a new issue"
        parameters:
          - name: "owner"
            type: "string"
            required: true
            description: "Repository owner"
            in: "path"
          - name: "repo"
            type: "string"
            required: true
            description: "Repository name"
            in: "path"
          - name: "title"
            type: "string"
            required: true
            description: "Issue title"
            in: "body"
          - name: "body"
            type: "string"
            required: false
            description: "Issue body"
            in: "body"
          - name: "labels"
            type: "array"
            required: false
            description: "Labels to apply"
            in: "body"
```

### Multiple APIs Example

```yaml
apis:
  # OpenAI API
  openai:
    base_url: "https://api.openai.com/v1"
    auth_type: "bearer"
    auth_config:
      token: "YOUR_OPENAI_API_KEY"
    rate_limit:
      requests_per_second: 50
      burst: 100
    cache_ttl: 0  # No caching for AI responses
    headers:
      Content-Type: "application/json"
    endpoints:
      - name: "chat_completion"
        path: "/chat/completions"
        method: "POST"
        description: "Create a chat completion"
        parameters:
          - name: "model"
            type: "string"
            required: true
            description: "Model to use (e.g., gpt-4)"
            in: "body"
          - name: "messages"
            type: "array"
            required: true
            description: "Array of message objects"
            in: "body"
          - name: "temperature"
            type: "number"
            required: false
            description: "Sampling temperature (0-2)"
            in: "body"
  
  # NewsAPI
  news:
    base_url: "https://newsapi.org/v2"
    auth_type: "api_key"
    auth_config:
      key: "YOUR_NEWS_API_KEY"
      header: "X-Api-Key"
    rate_limit:
      requests_per_second: 5
      burst: 10
    cache_ttl: 15m
    endpoints:
      - name: "top_headlines"
        path: "/top-headlines"
        method: "GET"
        description: "Get top news headlines"
        parameters:
          - name: "country"
            type: "string"
            required: false
            description: "2-letter country code"
            in: "query"
          - name: "category"
            type: "string"
            required: false
            description: "Category (business, entertainment, health, science, sports, technology)"
            in: "query"
          - name: "q"
            type: "string"
            required: false
            description: "Keywords to search for"
            in: "query"
  
  # Stripe API
  stripe:
    base_url: "https://api.stripe.com/v1"
    auth_type: "api_key"
    auth_config:
      key: "YOUR_STRIPE_SECRET_KEY"
      header: "Authorization"
    rate_limit:
      requests_per_second: 25
      burst: 50
    cache_ttl: 30s
    endpoints:
      - name: "list_customers"
        path: "/customers"
        method: "GET"
        description: "List all customers"
        parameters:
          - name: "limit"
            type: "number"
            required: false
            description: "Number of customers to return"
            in: "query"
      
      - name: "create_payment_intent"
        path: "/payment_intents"
        method: "POST"
        description: "Create a payment intent"
        parameters:
          - name: "amount"
            type: "number"
            required: true
            description: "Amount in cents"
            in: "body"
          - name: "currency"
            type: "string"
            required: true
            description: "Three-letter currency code"
            in: "body"
```

## Usage with Claude Desktop

Add to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "api-gateway": {
      "command": "/path/to/api-gateway",
      "env": {
        "API_GATEWAY_CONFIG": "/path/to/config.yaml"
      }
    }
  }
}
```

## Authentication Types

### API Key

```yaml
auth_type: "api_key"
auth_config:
  key: "YOUR_API_KEY"
  # Option 1: As query parameter
  param: "api_key"
  # Option 2: As header
  header: "X-API-Key"
```

### Bearer Token

```yaml
auth_type: "bearer"
auth_config:
  token: "YOUR_TOKEN"
```

### OAuth2

```yaml
auth_type: "oauth2"
auth_config:
  access_token: "YOUR_ACCESS_TOKEN"
  # Note: Token refresh not yet implemented
```

## Rate Limiting

Configure per-API rate limits:

```yaml
rate_limit:
  requests_per_second: 10  # Average rate
  burst: 20               # Maximum burst size
```

## Caching

Configure response caching per API:

```yaml
cache_ttl: 5m  # Cache responses for 5 minutes
# Supported units: s, m, h
# Set to 0 to disable caching
```

## Environment Variables

- `API_GATEWAY_CONFIG`: Path to configuration file (default: `config.yaml`)

## Security Considerations

1. **Never commit API keys**: Use environment variables or secure secret management
2. **Use HTTPS**: Always use HTTPS endpoints for secure communication
3. **Validate inputs**: The gateway validates required parameters
4. **Rate limiting**: Protects against API quota exhaustion

## Advanced Configuration

### Custom Headers

Add custom headers to all requests for an API:

```yaml
headers:
  User-Agent: "MCP-API-Gateway/1.0"
  Accept: "application/json"
```

### Complex Parameters

Support for different parameter types:

```yaml
parameters:
  - name: "tags"
    type: "array"
    in: "body"
  - name: "metadata"
    type: "object"
    in: "body"
  - name: "active"
    type: "boolean"
    in: "query"
```

## Tool Naming Convention

Tools are registered with the format: `{api_name}_{endpoint_name}`

Examples:
- `github_get_user`
- `weather_current`
- `stripe_create_payment_intent`

## Error Handling

The gateway provides detailed error messages:
- Rate limit exceeded
- Authentication failures
- API errors with status codes
- Network timeouts (30s default)

## Performance Tips

1. **Enable caching** for read-only endpoints
2. **Set appropriate rate limits** to avoid hitting API quotas
3. **Use burst capacity** for handling traffic spikes
4. **Monitor cache hit rates** for optimization

## Extending the Gateway

To add support for new authentication methods or features:

1. Modify the `APIConfig` struct
2. Update the `addAuthentication` method
3. Add new configuration examples

## Troubleshooting

1. **Check logs** for detailed error messages
2. **Verify API credentials** are correct
3. **Test endpoints** with curl first
4. **Monitor rate limits** in API dashboards