# API Gateway Examples

## Quick Start

### 1. Testing with JSONPlaceholder (No Auth Required)

```bash
# Start the server
API_GATEWAY_CONFIG=config.yaml go run main.go

# In Claude Desktop, you can now use:
# - jsonplaceholder_get_posts: Get all posts
# - jsonplaceholder_get_post: Get a specific post by ID
# - jsonplaceholder_create_post: Create a new post
```

### 2. Weather API Setup

1. Get a free API key from [OpenWeatherMap](https://openweathermap.org/api)
2. Set environment variable:
   ```bash
   export OPENWEATHER_API_KEY="your_api_key_here"
   ```
3. Use in Claude:
   - `weather_current`: Get current weather (e.g., city: "London")
   - `weather_forecast`: Get 5-day forecast

### 3. GitHub API Setup

1. Create a personal access token at [GitHub Settings](https://github.com/settings/tokens)
2. Set environment variable:
   ```bash
   export GITHUB_TOKEN="your_token_here"
   ```
3. Available tools:
   - `github_get_user`: Get user info
   - `github_list_repos`: List user's repositories
   - `github_search_repos`: Search all repositories

## Real-World Usage Examples

### Weather Dashboard

```yaml
# Custom weather config for a dashboard app
apis:
  weather_dashboard:
    base_url: "https://api.openweathermap.org/data/2.5"
    auth_type: "api_key"
    auth_config:
      key: "${OPENWEATHER_API_KEY}"
      param: "appid"
    rate_limit:
      requests_per_second: 50  # Higher limit for dashboard
      burst: 100
    cache_ttl: 2m  # Shorter cache for real-time data
    endpoints:
      - name: "multi_city"
        path: "/group"
        method: "GET"
        description: "Get weather for multiple cities at once"
        parameters:
          - name: "id"
            type: "string"
            required: true
            description: "Comma-separated city IDs"
            in: "query"
          - name: "units"
            type: "string"
            required: false
            description: "Temperature units"
            in: "query"
```

### Slack Integration

```yaml
apis:
  slack:
    base_url: "https://slack.com/api"
    auth_type: "bearer"
    auth_config:
      token: "${SLACK_BOT_TOKEN}"
    rate_limit:
      requests_per_second: 1  # Slack's tier 1 rate limit
      burst: 1
    cache_ttl: 0  # No caching for real-time messages
    endpoints:
      - name: "post_message"
        path: "/chat.postMessage"
        method: "POST"
        description: "Post a message to a Slack channel"
        parameters:
          - name: "channel"
            type: "string"
            required: true
            description: "Channel ID or name"
            in: "body"
          - name: "text"
            type: "string"
            required: true
            description: "Message text"
            in: "body"
          - name: "thread_ts"
            type: "string"
            required: false
            description: "Thread timestamp for replies"
            in: "body"
```

### Stripe Payment Processing

```yaml
apis:
  stripe:
    base_url: "https://api.stripe.com/v1"
    auth_type: "api_key"
    auth_config:
      key: "Bearer ${STRIPE_SECRET_KEY}"  # Stripe uses Bearer prefix
      header: "Authorization"
    rate_limit:
      requests_per_second: 25
      burst: 50
    cache_ttl: 0  # Never cache payment data
    headers:
      Stripe-Version: "2023-10-16"
    endpoints:
      - name: "create_customer"
        path: "/customers"
        method: "POST"
        description: "Create a new customer"
        parameters:
          - name: "email"
            type: "string"
            required: true
            description: "Customer email"
            in: "body"
          - name: "name"
            type: "string"
            required: false
            description: "Customer name"
            in: "body"
      
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
          - name: "customer"
            type: "string"
            required: false
            description: "Customer ID"
            in: "body"
```

### Multi-Environment Configuration

```yaml
# Production config with multiple APIs
apis:
  # Production API with strict rate limits
  production_api:
    base_url: "${PROD_API_URL}"
    auth_type: "oauth2"
    auth_config:
      access_token: "${PROD_ACCESS_TOKEN}"
    rate_limit:
      requests_per_second: 100
      burst: 200
    cache_ttl: 1m
    headers:
      X-Environment: "production"
      X-Client-Version: "1.0.0"
    endpoints:
      - name: "get_data"
        path: "/api/v1/data"
        method: "GET"
        description: "Get production data"
        parameters:
          - name: "filter"
            type: "string"
            required: false
            in: "query"

  # Staging API with relaxed limits
  staging_api:
    base_url: "${STAGING_API_URL}"
    auth_type: "api_key"
    auth_config:
      key: "${STAGING_API_KEY}"
      header: "X-API-Key"
    rate_limit:
      requests_per_second: 1000
      burst: 2000
    cache_ttl: 0  # No caching in staging
    headers:
      X-Environment: "staging"
```

### Analytics API Aggregator

```yaml
apis:
  google_analytics:
    base_url: "https://analyticsreporting.googleapis.com/v4"
    auth_type: "oauth2"
    auth_config:
      access_token: "${GA_ACCESS_TOKEN}"
    rate_limit:
      requests_per_second: 10
      burst: 20
    cache_ttl: 1h  # Cache analytics data
    endpoints:
      - name: "get_reports"
        path: "/reports:batchGet"
        method: "POST"
        description: "Get analytics reports"
        parameters:
          - name: "reportRequests"
            type: "array"
            required: true
            in: "body"

  mixpanel:
    base_url: "https://mixpanel.com/api/2.0"
    auth_type: "api_key"
    auth_config:
      key: "${MIXPANEL_SECRET}"
      header: "Authorization"
    rate_limit:
      requests_per_second: 60
      burst: 120
    cache_ttl: 30m
    endpoints:
      - name: "export_events"
        path: "/export"
        method: "GET"
        description: "Export raw event data"
        parameters:
          - name: "from_date"
            type: "string"
            required: true
            in: "query"
          - name: "to_date"
            type: "string"
            required: true
            in: "query"
```

## Advanced Patterns

### 1. Webhook Receiver

```yaml
apis:
  webhook_receiver:
    base_url: "${WEBHOOK_ENDPOINT}"
    auth_type: "api_key"
    auth_config:
      key: "${WEBHOOK_SECRET}"
      header: "X-Webhook-Secret"
    rate_limit:
      requests_per_second: 1000
      burst: 5000
    cache_ttl: 0
    endpoints:
      - name: "forward_webhook"
        path: "/webhook"
        method: "POST"
        description: "Forward webhook payload"
        parameters:
          - name: "event_type"
            type: "string"
            required: true
            in: "body"
          - name: "payload"
            type: "object"
            required: true
            in: "body"
```

### 2. GraphQL API

```yaml
apis:
  graphql:
    base_url: "https://api.example.com"
    auth_type: "bearer"
    auth_config:
      token: "${GRAPHQL_TOKEN}"
    rate_limit:
      requests_per_second: 100
      burst: 200
    cache_ttl: 5m
    headers:
      Content-Type: "application/json"
    endpoints:
      - name: "query"
        path: "/graphql"
        method: "POST"
        description: "Execute GraphQL query"
        parameters:
          - name: "query"
            type: "string"
            required: true
            description: "GraphQL query string"
            in: "body"
          - name: "variables"
            type: "object"
            required: false
            description: "Query variables"
            in: "body"
```

### 3. Batch Processing

```yaml
apis:
  batch_processor:
    base_url: "${BATCH_API_URL}"
    auth_type: "api_key"
    auth_config:
      key: "${BATCH_API_KEY}"
      param: "api_key"
    rate_limit:
      requests_per_second: 5  # Low rate for batch operations
      burst: 10
    cache_ttl: 0
    endpoints:
      - name: "submit_batch"
        path: "/batch/submit"
        method: "POST"
        description: "Submit batch job"
        parameters:
          - name: "items"
            type: "array"
            required: true
            description: "Array of items to process"
            in: "body"
          - name: "callback_url"
            type: "string"
            required: false
            description: "Webhook for completion"
            in: "body"
```

## Testing Your Configuration

### 1. Validate YAML

```bash
# Check if config is valid YAML
python -m yaml config.yaml
```

### 2. Test with curl

```bash
# Test an endpoint directly
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "https://api.github.com/users/octocat"
```

### 3. Environment Variable Substitution

The gateway supports environment variable substitution in config:

```yaml
auth_config:
  key: "${API_KEY}"           # From environment
  key: "${API_KEY:-default}"  # With default value
```

### 4. Debug Mode

Set environment variable for verbose logging:

```bash
export DEBUG=true
./api-gateway
```

## Best Practices

1. **Security**
   - Always use environment variables for secrets
   - Never commit API keys to version control
   - Use HTTPS endpoints only

2. **Performance**
   - Enable caching for read-only operations
   - Set appropriate rate limits
   - Use burst capacity wisely

3. **Reliability**
   - Handle errors gracefully
   - Implement retries for transient failures
   - Monitor API usage and limits

4. **Organization**
   - Group related endpoints under one API
   - Use consistent naming conventions
   - Document all parameters clearly