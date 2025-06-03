# System Interaction Flow Diagrams

Service-to-service communication, transport protocols, and system architecture patterns.

## Multi-Protocol Transport Architecture

```mermaid
sequenceDiagram
    participant IDE as Claude Desktop/VS Code
    participant WC as Web Client
    participant GQL as GraphQL Client
    participant AG as API Gateway
    participant MCP as MCP Server
    participant DI as DI Container
    participant VS as Vector Store
    
    par stdio Protocol
        IDE->>AG: stdio JSON-RPC
        AG->>MCP: forward request
    and HTTP Protocol
        WC->>AG: HTTP POST /mcp
        AG->>MCP: JSON-RPC over HTTP
    and WebSocket Protocol
        WC->>AG: WebSocket /ws
        AG->>MCP: real-time JSON-RPC
    and GraphQL Protocol
        GQL->>AG: POST /graphql
        AG->>MCP: GraphQL to MCP tool mapping
    end
    
    MCP->>DI: access service dependencies
    DI-->>MCP: injected services
    MCP->>VS: vector operations
    VS-->>MCP: operation results
    MCP-->>AG: unified response format
    
    alt stdio Response
        AG-->>IDE: stdio JSON-RPC response
    else HTTP Response
        AG-->>WC: HTTP JSON response
    else WebSocket Response
        AG-->>WC: WebSocket message
    else GraphQL Response
        AG-->>GQL: GraphQL response
    end
    
    Note over IDE,VS: Multi-protocol convergence to unified backend
```

## Service Dependency Initialization

```mermaid
sequenceDiagram
    participant M as Main
    participant DI as DI Container
    parameter CS as Config Service
    participant VS as Vector Store
    participant ES as Embedding Service
    participant IS as Intelligence Services
    participant WS as Workflow Services
    parameter HS as Health Service
    
    M->>DI: initialize container
    DI->>CS: load configuration
    CS-->>DI: config loaded
    
    DI->>VS: initialize vector store
    VS->>VS: Qdrant client + wrappers
    VS-->>DI: vector store ready
    
    DI->>ES: initialize embedding service
    ES->>ES: OpenAI client + circuit breaker
    ES-->>DI: embedding service ready
    
    DI->>IS: initialize intelligence layer
    IS->>IS: learning, pattern, knowledge engines
    IS-->>DI: intelligence services ready
    
    DI->>WS: initialize workflow services
    WS->>WS: context suggester, todo tracker
    WS-->>DI: workflow services ready
    
    DI->>HS: initialize health monitoring
    HS-->>DI: health service ready
    
    DI-->>M: all services initialized
    
    Note over M,HS: Ordered dependency injection with health monitoring
```

## WebSocket Hub Management

```mermaid
sequenceDiagram
    participant C1 as Client 1
    participant C2 as Client 2
    participant WH as WebSocket Hub
    participant CM as Connection Manager
    participant BM as Broadcast Manager
    participant RM as Room Manager
    
    C1->>WH: WebSocket connection
    WH->>CM: register client
    CM->>RM: assign to repository room
    RM-->>CM: room assigned
    CM-->>WH: client registered
    WH-->>C1: connection established
    
    C2->>WH: WebSocket connection
    WH->>CM: register client
    CM->>RM: assign to repository room
    RM-->>CM: room assigned
    CM-->>WH: client registered
    WH-->>C2: connection established
    
    C1->>WH: memory operation
    WH->>MemoryServer: process operation
    MemoryServer-->>WH: operation result
    WH->>BM: broadcast memory update
    BM->>RM: get room clients
    RM-->>BM: clients in repository room
    BM->>C1: operation result
    BM->>C2: memory update notification
    
    Note over C1,RM: Repository-scoped real-time updates
```

## Circuit Breaker Coordination

```mermaid
sequenceDiagram
    participant A as Application
    participant CB1 as Vector Store CB
    participant CB2 as Embedding CB
    participant CB3 as External API CB
    participant HM as Health Monitor
    participant AM as Alert Manager
    participant FB as Fallback Manager
    
    A->>CB1: vector operation
    A->>CB2: embedding operation  
    A->>CB3: external API call
    
    par Circuit Monitoring
        CB1->>HM: report health status
        CB2->>HM: report health status
        CB3->>HM: report health status
    end
    
    HM->>HM: aggregate system health
    
    alt System Healthy
        HM-->>A: all operations proceed
    else Partial Degradation
        HM->>FB: activate fallback strategies
        FB->>CB1: enable fallback mode
        FB-->>A: degraded service mode
    else System Unhealthy
        HM->>AM: trigger alerts
        AM-->>Admin: system failure alert
        HM-->>A: emergency fallback mode
    end
    
    Note over A,FB: Coordinated circuit breaker management
```

## Event-Driven Memory Updates

```mermaid
sequenceDiagram
    participant MS as Memory Server
    participant EB as Event Bus
    participant LE as Learning Engine
    participant KG as Knowledge Graph
    participant CS as Context Suggester
    participant AN as Analytics
    participant WS as WebSocket Clients
    
    MS->>EB: publish memory.created event
    
    par Event Processing
        EB->>LE: memory.created
        LE->>LE: update learning models
        LE-->>EB: learning updated
    and
        EB->>KG: memory.created
        KG->>KG: update knowledge graph
        KG-->>EB: graph updated
    and
        EB->>CS: memory.created
        CS->>CS: update context suggestions
        CS-->>EB: context updated
    and
        EB->>AN: memory.created
        AN->>AN: update analytics
        AN-->>EB: analytics updated
    and
        EB->>WS: memory.created
        WS-->>Clients: real-time notifications
    end
    
    Note over MS,WS: Asynchronous event-driven updates
```

## Health Monitoring System

```mermaid
sequenceDiagram
    participant LB as Load Balancer
    participant HE as Health Endpoint
    participant HM as Health Manager
    participant VS as Vector Store
    participant ES as Embedding Service
    participant MM as Memory Manager
    participant MS as Metrics Service
    
    LB->>HE: GET /health
    HE->>HM: comprehensive health check
    
    par Component Health Checks
        HM->>VS: check Qdrant connectivity
        VS->>VS: ping Qdrant cluster
        VS-->>HM: connection status
    and
        HM->>ES: check OpenAI API
        ES->>ES: test embedding generation
        ES-->>HM: API status
    and
        HM->>MM: check memory usage
        MM->>MM: system resource analysis
        MM-->>HM: resource metrics
    and
        HM->>MS: check metrics collection
        MS-->>HM: metrics status
    end
    
    HM->>HM: aggregate health score
    
    alt All Components Healthy
        HM-->>HE: healthy status
        HE-->>LB: 200 OK
    else Some Components Degraded
        HM-->>HE: degraded status
        HE-->>LB: 200 OK (with warnings)
    else Critical Components Failed
        HM-->>HE: unhealthy status
        HE-->>LB: 503 Service Unavailable
    end
    
    Note over LB,MS: Comprehensive system health monitoring
```

## Graceful Shutdown Sequence

```mermaid
sequenceDiagram
    participant OS as Operating System
    participant MS as Main Server
    participant WH as WebSocket Hub
    participant CS as Connection Service
    participant VS as Vector Store
    participant BM as Backup Manager
    participant HS as Health Service
    
    OS->>MS: SIGTERM signal
    MS->>HS: stop health endpoints
    HS-->>MS: health stopped
    MS->>WH: close WebSocket connections
    WH->>WH: notify clients of shutdown
    WH-->>MS: connections closed
    MS->>CS: stop accepting new connections
    CS-->>MS: new connections stopped
    MS->>VS: flush pending operations
    VS->>VS: complete in-flight operations
    VS-->>MS: operations completed
    MS->>BM: trigger emergency backup
    BM->>BM: backup critical data
    BM-->>MS: backup completed
    MS->>MS: cleanup resources
    MS-->>OS: graceful shutdown complete
    
    Note over OS,HS: Orderly resource cleanup and data protection
```

## Load Balancing & Scaling

```mermaid
sequenceDiagram
    participant C as Clients
    participant LB as Load Balancer
    participant I1 as Instance 1
    participant I2 as Instance 2
    participant I3 as Instance 3
    participant VS as Shared Vector Store
    participant HM as Health Monitor
    
    C->>LB: memory operations
    LB->>HM: check instance health
    HM-->>LB: instance status
    
    alt Instance 1 Healthy
        LB->>I1: route request
        I1->>VS: vector operation
        VS-->>I1: operation result
        I1-->>LB: response
    else Instance 1 Unhealthy
        LB->>I2: route to backup instance
        I2->>VS: vector operation
        VS-->>I2: operation result
        I2-->>LB: response
    end
    
    LB-->>C: response
    
    loop Auto-scaling
        HM->>HM: monitor load metrics
        alt High Load
            HM->>I3: start additional instance
            I3-->>HM: instance ready
            HM->>LB: add instance to pool
        else Low Load
            HM->>LB: remove excess instance
            HM->>I3: graceful shutdown
        end
    end
    
    Note over C,HM: Dynamic scaling with health-aware routing
```

## Cross-Service Communication Pattern

```mermaid
sequenceDiagram
    participant MC as Memory Core
    participant IS as Intelligence Service
    participant WF as Workflow Service
    participant AS as Analytics Service
    participant NS as Notification Service
    participant AU as Audit Service
    
    MC->>IS: memory operation request
    IS->>WF: check workflow context
    WF-->>IS: workflow state
    IS->>AS: log analytics event
    AS-->>IS: analytics recorded
    IS->>MC: enhanced operation
    MC->>MC: execute memory operation
    MC->>NS: trigger notifications
    NS-->>Clients: real-time updates
    MC->>AU: audit operation
    AU-->>MC: audit logged
    MC-->>IS: operation complete
    IS-->>MC: enhanced result
    
    Note over MC,AU: Comprehensive service orchestration
```