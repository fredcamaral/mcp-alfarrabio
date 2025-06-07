/**
 * API Client for MCP Memory Server
 * 
 * Provides a comprehensive client for interacting with the Go backend API endpoints.
 * Handles authentication, error handling, request/response transformation, and typing.
 */

import { CSRFManager } from './csrf-client'
import { handleError } from './error-handling'
import { logger } from './logger'

// Use relative URLs in browser to leverage Next.js proxy, direct backend URLs in server
const BASE_URL = typeof window !== 'undefined' ? '' : (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080')

// Request/Response interceptor types
export interface RequestInterceptor {
  onRequest?: (config: RequestInit & { url: string }) => RequestInit & { url: string }
  onRequestError?: (error: unknown) => Promise<never>
}

export interface ResponseInterceptor {
  onResponse?: (response: Response) => Response | Promise<Response>
  onResponseError?: (error: unknown) => Promise<never>
}

// Interceptor manager
class InterceptorManager {
  private requestInterceptors: RequestInterceptor[] = []
  private responseInterceptors: ResponseInterceptor[] = []

  addRequestInterceptor(interceptor: RequestInterceptor): () => void {
    this.requestInterceptors.push(interceptor)
    return () => {
      const index = this.requestInterceptors.indexOf(interceptor)
      if (index > -1) {
        this.requestInterceptors.splice(index, 1)
      }
    }
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): () => void {
    this.responseInterceptors.push(interceptor)
    return () => {
      const index = this.responseInterceptors.indexOf(interceptor)
      if (index > -1) {
        this.responseInterceptors.splice(index, 1)
      }
    }
  }

  async processRequest(config: RequestInit & { url: string }): Promise<RequestInit & { url: string }> {
    let processedConfig = config

    for (const interceptor of this.requestInterceptors) {
      try {
        if (interceptor.onRequest) {
          processedConfig = interceptor.onRequest(processedConfig)
        }
      } catch (error) {
        if (interceptor.onRequestError) {
          await interceptor.onRequestError(error)
        }
        throw error
      }
    }

    return processedConfig
  }

  async processResponse(response: Response): Promise<Response> {
    let processedResponse = response

    for (const interceptor of this.responseInterceptors) {
      try {
        if (interceptor.onResponse) {
          processedResponse = await Promise.resolve(interceptor.onResponse(processedResponse))
        }
      } catch (error) {
        if (interceptor.onResponseError) {
          await interceptor.onResponseError(error)
        }
        throw error
      }
    }

    return processedResponse
  }

  async processResponseError(error: unknown): Promise<never> {
    for (const interceptor of this.responseInterceptors) {
      if (interceptor.onResponseError) {
        try {
          await interceptor.onResponseError(error)
        } catch {
          // Continue with original error if interceptor fails
          break
        }
      }
    }
    throw error
  }
}

// Re-export types from the main type definitions
import type { ConversationChunk } from '@/types/memory'
export type { ConversationChunk } from '@/types/memory'

export interface SearchRequest {
  query: string
  repository?: string
  limit?: number
  offset?: number
  confidence_threshold?: number
  type_filter?: string[]
  date_from?: string
  date_to?: string
  tags?: string[]
  session_id?: string
}

export interface SearchResponse {
  chunks: ConversationChunk[]
  total: number
  query: string
  query_time: number
  confidence_scores: number[]
}

export interface MemoryAnalysisRequest {
  operation: string
  options: {
    repository: string
    session_id?: string
    [key: string]: unknown
  }
}

export interface MemoryAnalysisResponse {
  operation: string
  result: unknown
  status: string
  repository: string
  session_id?: string
}

export interface BackupMetadata {
  version: string
  created_at: string
  chunk_count: number
  size: number
  repository?: string
  metadata?: {
    backup_file?: string
    compression?: string
    format?: string
  }
}

export interface APIErrorResponse {
  error: string
  code?: string
  details?: unknown
  status?: number
}

// API Client class
export class MemoryAPIClient {
  private baseURL: string
  private defaultHeaders: Record<string, string>
  private interceptors: InterceptorManager

  constructor(baseURL: string = BASE_URL) {
    this.baseURL = baseURL
    this.defaultHeaders = {
      'Content-Type': 'application/json',
    }
    this.interceptors = new InterceptorManager()
    this.setupDefaultInterceptors()
  }

  /**
   * Add request interceptor
   */
  addRequestInterceptor(interceptor: RequestInterceptor): () => void {
    return this.interceptors.addRequestInterceptor(interceptor)
  }

  /**
   * Add response interceptor
   */
  addResponseInterceptor(interceptor: ResponseInterceptor): () => void {
    return this.interceptors.addResponseInterceptor(interceptor)
  }

  /**
   * Setup default interceptors for logging, error handling, etc.
   */
  private setupDefaultInterceptors(): void {
    // Request logging interceptor
    this.addRequestInterceptor({
      onRequest: (config) => {
        if (process.env.NODE_ENV === 'development') {
          logger.debug(`API Request: ${config.method || 'GET'} ${config.url}`, {
            component: 'APIClient',
            action: 'request',
            method: config.method || 'GET',
            url: config.url
          })
        }
        return config
      },
      onRequestError: async (error) => {
        logger.error('Request interceptor error', error, {
          component: 'APIClient',
          action: 'request-interceptor'
        })
        throw error
      }
    })

    // Response logging and error handling interceptor
    this.addResponseInterceptor({
      onResponse: (response) => {
        if (process.env.NODE_ENV === 'development') {
          logger.debug(`API Response: ${response.status} ${response.url}`, {
            component: 'APIClient',
            action: 'response',
            status: response.status,
            url: response.url
          })
        }
        return response
      },
      onResponseError: async (error) => {
        logger.error('Response interceptor error', error, {
          component: 'APIClient',
          action: 'response-interceptor'
        })
        handleError(error, { source: 'api_response_interceptor' })
        throw error
      }
    })

  }

  private async makeRequest<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {

    const url = `${this.baseURL}${endpoint}`

    // Add CSRF headers for protected requests
    const headers = {
      ...this.defaultHeaders,
      ...CSRFManager.getHeaders(),
      ...(options.headers as Record<string, string> || {})
    }

    let config: RequestInit & { url: string } = {
      ...options,
      headers,
      credentials: 'include',
      url
    }

    try {
      // Process request through interceptors
      config = await this.interceptors.processRequest(config)

      const response = await fetch(config.url, config)

      // Process response through interceptors
      const processedResponse = await this.interceptors.processResponse(response)

      if (!processedResponse.ok) {
        const errorData = await processedResponse.json().catch(() => ({}))
        const apiError = new APIError(
          errorData.error || `HTTP ${processedResponse.status}: ${processedResponse.statusText}`,
          errorData.code,
          errorData.details,
          processedResponse.status
        )
        throw apiError
      }

      // Handle empty responses
      const contentType = processedResponse.headers.get('content-type')
      if (contentType && contentType.includes('application/json')) {
        return await processedResponse.json()
      } else {
        return {} as T
      }
    } catch (error) {

      // Process error through interceptors
      try {
        await this.interceptors.processResponseError(error)
      } catch (interceptorError) {
        // Use interceptor error if available, otherwise original error
        error = interceptorError
      }

      if (error instanceof APIError) {
        handleError(error, {
          source: 'api_client',
          endpoint,
          method: options.method || 'GET'
        })
        throw error
      }

      const networkError = new APIError(
        error instanceof Error ? error.message : 'Network error occurred',
        'NETWORK_ERROR'
      )
      handleError(networkError, {
        source: 'api_client',
        endpoint,
        method: options.method || 'GET',
        originalError: error
      })
      throw networkError
    }
  }


  // Memory Operations
  async storeChunk(chunk: Partial<ConversationChunk>): Promise<{ chunk_id: string }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_create',
        params: {
          operation: 'store_chunk',
          options: {
            content: chunk.content,
            type: chunk.type || 'discussion',
            session_id: chunk.session_id,
            repository: chunk.metadata?.repository,
            tags: chunk.metadata?.tags,
            metadata: chunk.metadata
          }
        },
        id: Date.now()
      })
    })
  }

  async searchMemories(request: SearchRequest): Promise<SearchResponse> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_read',
        params: {
          operation: 'search',
          options: {
            query: request.query,
            repository: request.repository,
            limit: request.limit || 50,
            offset: request.offset || 0,
            confidence_threshold: request.confidence_threshold,
            type_filter: request.type_filter,
            date_from: request.date_from,
            date_to: request.date_to,
            tags: request.tags,
            session_id: request.session_id
          }
        },
        id: Date.now()
      })
    })
  }

  async getMemoryContext(repository: string): Promise<SearchResponse> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_read',
        params: {
          operation: 'get_context',
          options: {
            repository
          }
        },
        id: Date.now()
      })
    })
  }

  async findSimilarMemories(problem: string, repository: string): Promise<ConversationChunk[]> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_read',
        params: {
          operation: 'find_similar',
          options: {
            problem,
            repository
          }
        },
        id: Date.now()
      })
    })
  }

  async getMemoryRelationships(chunkId: string, repository: string): Promise<{ relationships: Array<{ id: string; source: string; target: string; type: string; confidence: number }> }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_read',
        params: {
          operation: 'get_relationships',
          options: {
            chunk_id: chunkId,
            repository
          }
        },
        id: Date.now()
      })
    })
  }

  async analyzeMemories(request: MemoryAnalysisRequest): Promise<MemoryAnalysisResponse> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_analyze',
        params: request,
        id: Date.now()
      })
    })
  }

  // Backup Operations
  async listBackups(): Promise<BackupMetadata[]> {
    const response = await this.makeRequest<{ backups: BackupMetadata[] }>('/api/backup')
    return response.backups
  }

  async createBackup(name?: string, repository?: string): Promise<BackupMetadata> {
    return this.makeRequest('/api/backup', {
      method: 'POST',
      body: JSON.stringify({ name, repository })
    })
  }

  async restoreBackup(backupFile: string, overwrite: boolean = false): Promise<{ message: string }> {
    return this.makeRequest('/api/backup', {
      method: 'PUT',
      body: JSON.stringify({ backupFile, overwrite })
    })
  }

  async cleanupBackups(): Promise<{ message: string }> {
    return this.makeRequest('/api/backup', {
      method: 'PATCH'
    })
  }

  // Health and Status
  async getHealth(): Promise<{ status: string; timestamp: string }> {
    return this.makeRequest('/health')
  }

  async getStatus(): Promise<{ status: string; uptime: number; version: string; health: Record<string, unknown> }> {
    return this.makeRequest('/api/status')
  }

  // Session Management
  async createSession(sessionId: string, repository: string): Promise<{ session_id: string }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_tasks',
        params: {
          operation: 'session_create',
          options: {
            session_id: sessionId,
            repository
          }
        },
        id: Date.now()
      })
    })
  }

  async endSession(sessionId: string, repository: string): Promise<{ message: string }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_tasks',
        params: {
          operation: 'session_end',
          options: {
            session_id: sessionId,
            repository
          }
        },
        id: Date.now()
      })
    })
  }

  // Export and Import
  async exportMemories(repository: string, format: string = 'json'): Promise<{ data: string; format: string; chunk_count: number }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_transfer',
        params: {
          operation: 'export_project',
          options: {
            repository,
            format
          }
        },
        id: Date.now()
      })
    })
  }

  async importMemories(data: string, repository: string, sessionId: string): Promise<{ message: string }> {
    return this.makeRequest('/api/mcp', {
      method: 'POST',
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'memory_transfer',
        params: {
          operation: 'import_context',
          options: {
            data,
            repository,
            session_id: sessionId
          }
        },
        id: Date.now()
      })
    })
  }

  // Validation
  async validateChunk(chunk: ConversationChunk): Promise<{ valid: boolean; errors?: string[] }> {
    // Simple client-side validation for now
    const errors: string[] = []

    if (!chunk.content || chunk.content.trim().length === 0) {
      errors.push('Content is required')
    }

    if (!chunk.type) {
      errors.push('Type is required')
    }

    return { valid: errors.length === 0, errors: errors.length > 0 ? errors : undefined }
  }

  // Connection test
  async testConnection(): Promise<boolean> {
    try {
      await this.getHealth()
      return true
    } catch {
      return false
    }
  }
}

// Custom API Error class
class APIError extends Error {
  constructor(
    message: string,
    public code?: string,
    public details?: unknown,
    public status?: number
  ) {
    super(message)
    this.name = 'APIError'
  }
}

// Export singleton instance
export const apiClient = new MemoryAPIClient()
export default apiClient