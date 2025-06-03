# API Transport Flow Diagrams

Multi-protocol API interactions supporting stdio, HTTP, WebSocket, and SSE transports.

## HTTP JSON-RPC Flow

```mermaid
sequenceDiagram
    participant C as HTTP Client
    participant CM as CORS Middleware
    participant S as MCP Server
    participant H as HTTP Handler
    participant T as Tool Handler
    participant VS as Vector Store
    
    C->>CM: POST /mcp (JSON-RPC request)
    CM->>CM: validate CORS headers
    CM-->>C: preflight response (if OPTIONS)
    C->>H: JSON-RPC payload
    H->>H: parse JSON-RPC message
    H->>S: extract tool call
    S->>T: route to tool handler
    T->>VS: perform memory operation
    VS-->>T: operation result
    T-->>S: tool response
    S-->>H: JSON-RPC response
    H-->>C: HTTP 200 with result
    
    Note over C,VS: HTTP transport with CORS support
```

## WebSocket Real-Time Flow

```mermaid
sequenceDiagram
    participant C as WebSocket Client
    participant WH as WebSocket Hub
    participant S as MCP Server
    participant VS as Vector Store
    participant N as Notifier
    
    C->>WH: WebSocket connection /ws
    WH->>WH: register client connection
    WH-->>C: connection established
    
    C->>WH: JSON-RPC memory operation
    WH->>S: forward tool request
    S->>VS: vector store operation
    VS-->>S: operation complete
    S->>N: trigger notification
    N->>WH: broadcast to clients
    WH->>C: real-time update
    S-->>WH: operation response
    WH-->>C: JSON-RPC response
    
    Note over C,N: Bidirectional real-time communication
```

## Server-Sent Events (SSE) Flow

```mermaid
sequenceDiagram
    participant C as SSE Client
    participant SH as SSE Handler
    participant HB as Heartbeat
    participant S as MCP Server
    participant ES as Event Stream
    
    C->>SH: GET /sse (EventSource)
    SH->>SH: establish SSE connection
    SH->>HB: start heartbeat timer
    SH-->>C: connected event
    
    loop Heartbeat
        HB->>SH: heartbeat tick
        SH-->>C: heartbeat event
    end
    
    C->>SH: HTTP POST /mcp (operation)
    SH->>S: process tool request
    S-->>SH: operation result
    SH->>ES: create event stream
    ES-->>C: data event with result
    
    Note over C,ES: Server-sent events with heartbeat
```

## stdio MCP Protocol Flow

```mermaid
sequenceDiagram
    participant IDE as Claude Desktop/VS Code
    participant P as MCP Proxy
    participant S as MCP Server
    participant DI as DI Container
    participant VS as Vector Store
    
    IDE->>P: stdio JSON-RPC request
    P->>P: proxy message formatting
    P->>S: forward to server
    S->>DI: access service dependencies
    DI-->>S: injected services
    S->>VS: memory operation
    VS-->>S: operation result
    S-->>P: JSON-RPC response
    P-->>IDE: stdio response
    
    Note over IDE,VS: Direct IDE integration via stdio
```

## GraphQL API Flow

```mermaid
sequenceDiagram
    participant C as GraphQL Client
    participant GS as GraphQL Server
    participant R as Resolver
    participant S as MCP Server
    participant VS as Vector Store
    participant FC as Field Cache
    
    C->>GS: POST /graphql (query)
    GS->>GS: parse GraphQL query
    GS->>R: resolve fields
    R->>FC: check field cache
    FC-->>R: cache miss
    R->>S: call memory operations
    S->>VS: vector store queries
    VS-->>S: query results
    S-->>R: resolved data
    R->>FC: cache field result
    R-->>GS: GraphQL response
    GS-->>C: JSON response
    
    Note over C,FC: Complex queries with field-level caching
```

## Health Check Endpoints Flow

```mermaid
sequenceDiagram
    participant LB as Load Balancer
    participant H as Health Handler
    participant HM as Health Manager
    participant VS as Vector Store
    participant ES as Embedding Service
    participant M as Memory Stats
    
    LB->>H: GET /health
    H->>HM: check system health
    
    par Component Checks
        HM->>VS: Qdrant connectivity
        VS-->>HM: healthy/unhealthy
    and
        HM->>ES: OpenAI API status
        ES-->>HM: API accessible
    and
        HM->>M: memory usage stats
        M-->>HM: resource metrics
    end
    
    HM->>HM: aggregate health status
    HM-->>H: health report
    H-->>LB: 200 OK or 503 Service Unavailable
    
    Note over LB,M: Comprehensive health monitoring
```

## Multi-Protocol Error Handling

```mermaid
sequenceDiagram
    participant C as Client
    participant TP as Transport Protocol
    participant EH as Error Handler
    participant AL as Audit Logger
    participant EM as Error Mapper
    
    C->>TP: malformed request
    TP->>TP: request validation fails
    TP->>EH: handle protocol error
    EH->>EM: map to standard error
    EM-->>EH: standardized error format
    EH->>AL: audit error occurrence
    
    alt HTTP Transport
        EH-->>C: HTTP 400/500 with JSON error
    else WebSocket Transport
        EH-->>C: WebSocket error frame
    else SSE Transport
        EH-->>C: error event stream
    else stdio Transport
        EH-->>C: JSON-RPC error response
    end
    
    Note over C,AL: Protocol-specific error responses
```

## Rate Limiting & Throttling Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant RL as Rate Limiter
    participant TH as Throttler
    participant S as MCP Server
    participant M as Metrics
    
    C->>RL: API request
    RL->>RL: check client rate limit
    RL->>TH: apply throttling rules
    
    alt Within Rate Limit
        TH-->>S: forward request
        S-->>TH: process normally
        TH-->>RL: successful response
        RL->>M: record metrics
        RL-->>C: 200 OK with response
    else Rate Limit Exceeded
        TH-->>RL: throttle request
        RL->>M: record rate limit hit
        RL-->>C: 429 Too Many Requests
    end
    
    Note over C,M: Request throttling with metrics
```

## Authentication Flow (API Key)

```mermaid
sequenceDiagram
    participant C as Client
    participant AM as Auth Middleware
    parameter AC as Access Control
    participant S as MCP Server
    parameter U as User Store
    
    C->>AM: request with X-API-Key header
    AM->>AC: validate API key
    AC->>U: lookup user by key
    U-->>AC: user permissions
    AC->>AC: check repository access
    
    alt Valid Key & Permissions
        AC-->>AM: authorization granted
        AM->>S: forward with user context
        S-->>AM: operation result
        AM-->>C: authorized response
    else Invalid Key or No Access
        AC-->>AM: authorization denied
        AM-->>C: 401 Unauthorized
    end
    
    Note over C,U: Token-based authentication with repository scoping
```