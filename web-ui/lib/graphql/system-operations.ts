/**
 * GraphQL System Operations
 * 
 * Queries for health, status, and system monitoring
 */

import { gql } from '@apollo/client'

// Types
export interface HealthStatus {
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

export interface SystemStatus {
  status: 'operational' | 'degraded' | 'down'
  timestamp: string
  uptime: number
  backend: {
    status: string
    version?: string
    [key: string]: unknown
  }
  performance: {
    avgResponseTime: number
    requestsPerMinute: number
    errorRate: number
    activeConnections?: number
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
    memory?: {
      rss: string
      heapTotal: string
      heapUsed: string
      external: string
    }
    node?: {
      version: string
      platform: string
      arch: string
    }
    cpu?: {
      usage: number
      cores: number
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

// Queries
export const HEALTH_CHECK = gql`
  query HealthCheck {
    health {
      status
      timestamp
      checks {
        database {
          status
          latency
          error
        }
        vectorDb {
          status
          latency
          error
        }
        cache {
          status
          latency
          error
        }
      }
    }
  }
`

export const SYSTEM_STATUS = gql`
  query SystemStatus {
    systemStatus {
      status
      timestamp
      uptime
      backend {
        status
        version
      }
      performance {
        avgResponseTime
        requestsPerMinute
        errorRate
        activeConnections
      }
      errors {
        total
        byType
        recent {
          timestamp
          error
          count
        }
      }
      system {
        memory {
          rss
          heapTotal
          heapUsed
          external
        }
        node {
          version
          platform
          arch
        }
        cpu {
          usage
          cores
        }
      }
      features {
        csrf
        websocket
        graphql
        monitoring
        errorBoundaries
      }
    }
  }
`

// Export/Import Operations
export interface ExportResult {
  success: boolean
  format: string
  size: number
  downloadUrl: string
  expiresAt: string
}

export interface ImportResult {
  success: boolean
  importedCount: number
  errors?: string[]
}

export const EXPORT_MEMORIES = gql`
  query ExportMemories($repository: String!, $format: String!) {
    exportMemories(repository: $repository, format: $format) {
      success
      format
      size
      downloadUrl
      expiresAt
    }
  }
`

export const IMPORT_MEMORIES = gql`
  mutation ImportMemories($data: String!, $repository: String!, $sessionId: String!) {
    importMemories(data: $data, repository: $repository, sessionId: $sessionId) {
      success
      importedCount
      errors
    }
  }
`

// Session Management
export interface Session {
  id: string
  repository: string
  startedAt: string
  lastActivity: string
  chunkCount: number
  metadata?: Record<string, unknown>
}

export interface SessionResult {
  success: boolean
  duration: number
  chunkCount: number
  summary?: string
}

export const CREATE_SESSION = gql`
  mutation CreateSession($sessionId: String!, $repository: String!) {
    createSession(sessionId: $sessionId, repository: $repository) {
      id
      repository
      startedAt
      lastActivity
      chunkCount
    }
  }
`

export const END_SESSION = gql`
  mutation EndSession($sessionId: String!, $repository: String!) {
    endSession(sessionId: $sessionId, repository: $repository) {
      success
      duration
      chunkCount
      summary
    }
  }
`