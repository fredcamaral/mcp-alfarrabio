# Authentication & Authorization Flow Diagrams

Security flows for multi-tenant access control and API key authentication.

## API Key Authentication Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant AM as Auth Middleware
    participant AC as Access Control Manager
    participant TS as Token Store
    participant US as User Store
    
    C->>AM: request with X-API-Key header
    AM->>AM: extract API key from header
    AM->>AC: validate token
    AC->>TS: lookup token in store
    TS-->>AC: token metadata (user_id, expiry)
    AC->>AC: check token expiration
    AC->>US: get user permissions
    US-->>AC: user access levels
    AC-->>AM: authentication result + user context
    
    alt Valid Token
        AM->>AM: set user context in request
        AM-->>C: proceed to handler
    else Invalid/Expired Token
        AM-->>C: 401 Unauthorized
    end
    
    Note over C,US: Token-based authentication with metadata
```

## Repository Access Control Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Request Handler
    participant AC as Access Control
    participant RP as Repository Permissions
    participant AL as Audit Logger
    
    C->>H: memory operation with repository parameter
    H->>H: extract repository from request
    H->>AC: check repository access
    AC->>RP: get user permissions for repository
    RP-->>AC: access level (none/read/write/admin)
    
    alt Admin Access
        AC-->>H: full access granted
        H->>AL: audit admin operation
    else Write Access
        AC->>AC: check operation type
        alt Write Operation
            AC-->>H: operation allowed
        else Admin-Only Operation
            AC-->>H: 403 Forbidden
        end
    else Read Access
        AC->>AC: check if read operation
        alt Read Operation
            AC-->>H: operation allowed
        else Write/Admin Operation
            AC-->>H: 403 Forbidden
        end
    else No Access
        AC-->>H: 403 Forbidden
    end
    
    H->>AL: audit access decision
    
    Note over C,AL: Hierarchical permission model with audit
```

## Multi-Tenant Isolation Flow

```mermaid
sequenceDiagram
    participant C1 as Client 1 (Repo A)
    participant C2 as Client 2 (Repo B)
    participant AC as Access Control
    participant VS as Vector Store
    participant QF as Query Filter
    participant AU as Audit
    
    par Client 1 Operations
        C1->>AC: search memories (repository=repo-a)
        AC->>AC: validate access to repo-a
        AC->>QF: apply repository filter
        QF->>VS: query with repo-a filter
        VS-->>QF: repo-a results only
        QF-->>AC: filtered results
        AC->>AU: audit repo-a access
        AC-->>C1: repo-a memories only
    and Client 2 Operations
        C2->>AC: search memories (repository=repo-b)
        AC->>AC: validate access to repo-b
        AC->>QF: apply repository filter
        QF->>VS: query with repo-b filter
        VS-->>QF: repo-b results only
        QF-->>AC: filtered results
        AC->>AU: audit repo-b access
        AC-->>C2: repo-b memories only
    end
    
    Note over C1,AU: Complete data isolation between tenants
```

## Token Generation & Management

```mermaid
sequenceDiagram
    participant A as Admin
    participant AM as Admin Interface
    participant TM as Token Manager
    participant TG as Token Generator
    participant TS as Token Store
    participant EN as Encryptor
    
    A->>AM: create API key for user
    AM->>TM: request token generation
    TM->>TG: generate secure token
    TG->>TG: create random token (32 bytes)
    TG-->>TM: raw token
    TM->>EN: encrypt token for storage
    EN-->>TM: encrypted token
    TM->>TS: store token with metadata
    TS-->>TM: storage confirmation
    TM-->>AM: token created successfully
    AM-->>A: API key (shown once)
    
    Note over A,EN: Secure token generation with encryption
```

## Permission Escalation Check

```mermaid
sequenceDiagram
    participant U as User
    participant H as Handler
    participant AC as Access Control
    participant PE as Permission Engine
    participant AL as Audit Logger
    participant SM as Security Monitor
    
    U->>H: admin operation request
    H->>AC: check current permissions
    AC->>PE: validate permission escalation
    PE->>PE: analyze operation vs current level
    
    alt Valid Escalation
        PE-->>AC: escalation allowed
        AC->>AL: audit permission escalation
        AC-->>H: proceed with elevated access
    else Unauthorized Escalation
        PE-->>AC: escalation denied
        AC->>SM: security violation alert
        SM->>SM: flag potential attack
        AC->>AL: audit violation attempt
        AC-->>H: 403 Forbidden
    end
    
    Note over U,SM: Security violation detection and alerting
```

## Access Control Matrix Validation

```mermaid
sequenceDiagram
    participant R as Request
    participant ACM as Access Control Matrix
    participant OP as Operation Parser
    participant PM as Permission Matcher
    participant CC as Compliance Checker
    
    R->>ACM: operation request
    ACM->>OP: parse operation type
    OP-->>ACM: operation metadata
    ACM->>PM: match against permissions
    PM->>PM: check user + repository + operation
    
    alt Permission Matrix Match
        PM-->>ACM: access granted
        ACM->>CC: compliance validation
        CC-->>ACM: compliant operation
        ACM-->>R: authorized
    else No Matrix Match
        PM-->>ACM: access denied
        ACM->>CC: log compliance violation
        ACM-->>R: 403 Forbidden
    end
    
    Note over R,CC: Matrix-based access control with compliance
```

## Session Management Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant SM as Session Manager
    participant SS as Session Store
    participant TM as Timeout Manager
    participant AL as Activity Logger
    
    C->>SM: authenticate with API key
    SM->>SS: create session record
    SS-->>SM: session ID
    SM->>TM: set session timeout
    SM->>AL: log session start
    SM-->>C: session established
    
    loop Active Session
        C->>SM: memory operation
        SM->>AL: log activity
        SM->>TM: reset timeout
        SM-->>C: operation result
    end
    
    TM->>SM: session timeout
    SM->>SS: invalidate session
    SM->>AL: log session end
    
    Note over C,AL: Session lifecycle with activity tracking
```

## Cross-Repository Access Flow

```mermaid
sequenceDiagram
    participant U as User
    participant H as Handler
    participant XRA as Cross-Repo Access
    participant RM as Repository Manager
    participant PM as Permission Merger
    participant GS as Global Search
    
    U->>H: cross-repository search request
    H->>XRA: validate cross-repo operation
    XRA->>RM: get user's repositories
    RM-->>XRA: accessible repository list
    XRA->>PM: merge repository permissions
    PM-->>XRA: effective permissions
    
    alt Has Cross-Repo Access
        XRA->>GS: search across allowed repos
        GS->>GS: apply repository filters
        GS-->>XRA: aggregated results
        XRA-->>H: cross-repo search results
    else Insufficient Permissions
        XRA-->>H: 403 Forbidden
    end
    
    Note over U,GS: Secure cross-repository data access
```

## Audit Trail Flow

```mermaid
sequenceDiagram
    participant O as Operation
    participant AL as Audit Logger
    participant AE as Audit Encoder
    participant AS as Audit Store
    participant AM as Audit Monitor
    participant N as Notification System
    
    O->>AL: security event occurred
    AL->>AE: encode audit entry
    AE->>AE: format: timestamp, user, action, resource
    AE-->>AL: structured audit log
    AL->>AS: persist audit entry
    AS-->>AL: storage confirmation
    AL->>AM: analyze for anomalies
    
    alt Anomaly Detected
        AM->>N: security alert
        N-->>Admin: notification sent
    else Normal Activity
        AM-->>AL: routine activity logged
    end
    
    Note over O,N: Comprehensive audit trail with anomaly detection
```