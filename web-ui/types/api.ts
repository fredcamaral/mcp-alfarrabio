/**
 * API Type Definitions
 * 
 * Provides type-safe interfaces for API requests and responses
 */

// Base API Response
export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
  message?: string
  details?: unknown
}

// Error Response
export interface ApiError {
  error: string
  message?: string
  code?: string
  details?: Record<string, unknown>
  statusCode?: number
}

// Pagination
export interface PaginationParams {
  limit?: number
  offset?: number
  cursor?: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  limit: number
  offset: number
  hasMore: boolean
  nextCursor?: string
}

// Memory API Types
export interface MemoryCreateRequest {
  content: string
  type: string
  repository: string
  sessionId?: string
  tags?: string[]
  metadata?: Record<string, unknown>
  priority?: 'low' | 'medium' | 'high'
  isPublic?: boolean
}

export interface MemoryUpdateRequest {
  id: string
  content?: string
  type?: string
  tags?: string[]
  metadata?: Record<string, unknown>
  priority?: 'low' | 'medium' | 'high'
}

export interface MemorySearchRequest extends PaginationParams {
  query: string
  repository?: string
  types?: string[]
  tags?: string[]
  sessionId?: string
  minRelevanceScore?: number
  timeRange?: {
    start?: string
    end?: string
  }
}

// Repository API Types
export interface RepositoryCreateRequest {
  url: string
  description?: string
  metadata?: Record<string, unknown>
}

export interface RepositoryUpdateRequest {
  id: string
  description?: string
  status?: 'ACTIVE' | 'INACTIVE' | 'SYNCING' | 'ERROR'
  metadata?: Record<string, unknown>
}

// Pattern API Types
export interface PatternSearchRequest extends PaginationParams {
  repository: string
  types?: string[]
  minConfidence?: number
}

// Config API Types
export interface ConfigUpdateRequest {
  host?: string
  port?: number
  protocol?: 'http' | 'https'
  transportProtocols?: {
    http?: boolean
    websocket?: boolean
    grpc?: boolean
  }
  vectorDb?: {
    provider?: 'qdrant' | 'chroma'
    host?: string
    port?: number
    collection?: string
    dimension?: number
  }
  openai?: {
    apiKey?: string
    model?: string
    maxTokens?: number
    temperature?: number
    timeout?: number
  }
  cacheEnabled?: boolean
  realtimeEnabled?: boolean
  analyticsEnabled?: boolean
  debugMode?: boolean
  authEnabled?: boolean
}

// Backup API Types
export interface BackupCreateRequest {
  repository?: string
  name?: string
  includeVectors?: boolean
  format?: 'json' | 'msgpack' | 'csv'
  compress?: boolean
}

export interface BackupRestoreRequest {
  file: string
  overwrite?: boolean
  validateIntegrity?: boolean
}

// Status API Types
export interface SystemStatus {
  status: 'operational' | 'degraded' | 'down'
  timestamp: string
  uptime: number
  backend: {
    status: string
    [key: string]: unknown
  }
  performance: {
    avgResponseTime: number
    requestsPerMinute: number
    errorRate: number
    [key: string]: unknown
  }
  errors: {
    total: number
    byType: Record<string, number>
    recent: Array<{
      timestamp: string
      error: string
      count: number
    }>
  }
  system: {
    memory: {
      rss: string
      heapTotal: string
      heapUsed: string
      external: string
    } | null
    node: {
      version: string
      platform: string
      arch: string
    }
  }
  features: {
    csrf: boolean
    websocket: boolean
    graphql: boolean
    monitoring: boolean
    errorBoundaries: boolean
  }
}

// Health Check Types
export interface HealthCheckResponse {
  status: 'healthy' | 'unhealthy' | 'degraded'
  timestamp: string
  checks: {
    database: HealthCheckResult
    vectorDb: HealthCheckResult
    cache: HealthCheckResult
    [key: string]: HealthCheckResult
  }
}

export interface HealthCheckResult {
  status: 'healthy' | 'unhealthy' | 'degraded'
  latency?: number
  error?: string
  details?: Record<string, unknown>
}

// WebSocket Message Types
export interface WSMessage<T = unknown> {
  type: string
  action?: string
  data?: T
  timestamp: string
  id?: string
}

// Form Data Types
export interface FormSubmitHandler {
  (url: string, data: FormData | Record<string, unknown>): Promise<Response>
}

// Logger Context Types
export interface LogContext {
  component?: string
  action?: string
  userId?: string
  sessionId?: string
  error?: unknown
  [key: string]: unknown
}

// Performance Metrics Types
export interface PerformanceMetric {
  name: string
  value: number
  unit: string
  timestamp: string
  tags?: Record<string, string>
}

// Error Reporter Types
export interface ErrorReport {
  error: Error | unknown
  context?: LogContext
  severity?: 'low' | 'medium' | 'high' | 'critical'
  timestamp: string
  stackTrace?: string
  userAgent?: string
  url?: string
}