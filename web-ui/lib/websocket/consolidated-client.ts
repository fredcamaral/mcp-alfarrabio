/**
 * Consolidated WebSocket Client
 * 
 * Unified WebSocket implementation that combines the best features from both implementations
 * and provides a single source of truth for WebSocket connections.
 */

import { useState, useEffect, useCallback, useMemo } from 'react'
import { logger } from '@/lib/logger'
import { store } from '@/store/store'
import { 
  addMemories, 
  updateMemory, 
  removeMemory 
} from '@/store/slices/memoriesSlice'
import { 
  setWebSocketStatus, 
  updateWebSocketStats 
} from '@/store/slices/uiSlice'
import type { ConversationChunk } from '@/types/memory'

// WebSocket Message Types
export interface WebSocketMessage<T = unknown> {
  type: 'memory_created' | 'memory_updated' | 'memory_deleted' | 
        'pattern_detected' | 'system_status' | 'error' | 
        'ping' | 'pong' | 'subscribe' | 'unsubscribe'
  action?: string
  chunk_id?: string
  repository?: string
  session_id?: string
  content?: string
  summary?: string
  tags?: string[]
  timestamp: string
  data?: T
  id?: string
}

// Connection Status Types
export type ConnectionStatus = 
  | 'connecting' 
  | 'connected' 
  | 'disconnected' 
  | 'error' 
  | 'reconnecting'

// WebSocket Configuration
export interface WebSocketConfig {
  url: string
  reconnectInterval?: number
  maxReconnectAttempts?: number
  reconnectBackoffMultiplier?: number
  maxReconnectInterval?: number
  heartbeatInterval?: number
  messageQueueSize?: number
  protocols?: string[]
  enableCompression?: boolean
  enableHeartbeat?: boolean
}

// WebSocket Statistics
export interface WebSocketStats {
  connected: boolean
  connectionStatus: ConnectionStatus
  lastConnected?: Date
  lastDisconnected?: Date
  lastMessage?: Date
  messagesSent: number
  messagesReceived: number
  reconnectAttempts: number
  errors: number
  latency?: number
}

// Default Configuration
const DEFAULT_CONFIG: Required<Omit<WebSocketConfig, 'url' | 'protocols'>> = {
  reconnectInterval: 3000,
  maxReconnectAttempts: 5,
  reconnectBackoffMultiplier: 1.5,
  maxReconnectInterval: 30000,
  heartbeatInterval: 30000,
  messageQueueSize: 100,
  enableCompression: true,
  enableHeartbeat: true
}

/**
 * Consolidated WebSocket Client Class
 */
export class ConsolidatedWebSocketClient {
  private ws: WebSocket | null = null
  private config: Required<WebSocketConfig>
  private reconnectTimeout: NodeJS.Timeout | null = null
  private heartbeatInterval: NodeJS.Timeout | null = null
  private messageQueue: WebSocketMessage[] = []
  private stats: WebSocketStats
  private isManuallyDisconnected = false
  private reconnectCount = 0
  private pingTimeout: NodeJS.Timeout | null = null
  private lastPingTime = 0

  constructor(config: WebSocketConfig) {
    this.config = {
      ...DEFAULT_CONFIG,
      ...config,
      protocols: config.protocols || []
    }
    
    this.stats = {
      connected: false,
      connectionStatus: 'disconnected',
      messagesSent: 0,
      messagesReceived: 0,
      reconnectAttempts: 0,
      errors: 0
    }
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.CONNECTING || 
        this.ws?.readyState === WebSocket.OPEN) {
      logger.debug('WebSocket already connected or connecting')
      return
    }

    this.isManuallyDisconnected = false
    this.updateStatus('connecting')
    
    try {
      this.ws = new WebSocket(this.config.url, this.config.protocols)
      
      if (this.config.enableCompression) {
        // Note: Compression is negotiated automatically by the browser
      }
      
      this.setupEventHandlers()
    } catch (error) {
      logger.error('Failed to create WebSocket', { error })
      this.updateStatus('error')
      this.scheduleReconnect()
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.isManuallyDisconnected = true
    this.cleanup()
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
    
    this.updateStatus('disconnected')
  }

  /**
   * Send a message through WebSocket
   */
  send(message: Omit<WebSocketMessage, 'timestamp'>): boolean {
    const fullMessage: WebSocketMessage = {
      ...message,
      timestamp: new Date().toISOString()
    }

    if (this.ws?.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(JSON.stringify(fullMessage))
        this.stats.messagesSent++
        this.updateStats()
        return true
      } catch (error) {
        logger.error('Failed to send WebSocket message', { error })
        this.stats.errors++
        return false
      }
    } else {
      // Queue message if not connected
      this.queueMessage(fullMessage)
      return false
    }
  }

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus {
    return this.stats.connectionStatus
  }

  /**
   * Get connection statistics
   */
  getStats(): Readonly<WebSocketStats> {
    return { ...this.stats }
  }

  /**
   * Subscribe to a repository or session
   */
  subscribe(repository?: string, sessionId?: string): void {
    this.send({
      type: 'subscribe',
      repository,
      session_id: sessionId
    })
  }

  /**
   * Unsubscribe from a repository or session
   */
  unsubscribe(repository?: string, sessionId?: string): void {
    this.send({
      type: 'unsubscribe',
      repository,
      session_id: sessionId
    })
  }

  private setupEventHandlers(): void {
    if (!this.ws) return

    this.ws.onopen = this.handleOpen.bind(this)
    this.ws.onmessage = this.handleMessage.bind(this)
    this.ws.onclose = this.handleClose.bind(this)
    this.ws.onerror = this.handleError.bind(this)
  }

  private handleOpen(): void {
    logger.info('WebSocket connected', { url: this.config.url })
    
    this.reconnectCount = 0
    this.stats.lastConnected = new Date()
    this.updateStatus('connected')
    
    // Start heartbeat
    if (this.config.enableHeartbeat) {
      this.startHeartbeat()
    }
    
    // Flush message queue
    this.flushMessageQueue()
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)
      
      this.stats.messagesReceived++
      this.stats.lastMessage = new Date()
      this.updateStats()
      
      // Handle different message types
      switch (message.type) {
        case 'pong':
          this.handlePong()
          break
          
        case 'memory_created':
          this.handleMemoryCreated(message)
          break
          
        case 'memory_updated':
          this.handleMemoryUpdated(message)
          break
          
        case 'memory_deleted':
          this.handleMemoryDeleted(message)
          break
          
        case 'pattern_detected':
          this.handlePatternDetected(message)
          break
          
        case 'system_status':
          this.handleSystemStatus(message)
          break
          
        case 'error':
          this.handleErrorMessage(message)
          break
          
        default:
          logger.debug('Unhandled WebSocket message type', { type: message.type })
      }
    } catch (error) {
      logger.error('Failed to parse WebSocket message', { error, data: event.data })
      this.stats.errors++
    }
  }

  private handleClose(event: CloseEvent): void {
    logger.info('WebSocket closed', { 
      code: event.code, 
      reason: event.reason,
      wasClean: event.wasClean 
    })
    
    this.cleanup()
    this.stats.lastDisconnected = new Date()
    
    if (!this.isManuallyDisconnected) {
      this.updateStatus('disconnected')
      this.scheduleReconnect()
    }
  }

  private handleError(event: Event): void {
    logger.error('WebSocket error', { error: event })
    this.stats.errors++
    this.updateStatus('error')
  }

  private handlePong(): void {
    if (this.lastPingTime > 0) {
      this.stats.latency = Date.now() - this.lastPingTime
      this.updateStats()
    }
    
    // Clear ping timeout
    if (this.pingTimeout) {
      clearTimeout(this.pingTimeout)
      this.pingTimeout = null
    }
  }

  private handleMemoryCreated(message: WebSocketMessage): void {
    if (message.chunk_id && message.content) {
      const chunk: ConversationChunk = {
        id: message.chunk_id,
        content: message.content,
        session_id: message.session_id || '',
        timestamp: message.timestamp,
        type: 'discussion',
        summary: message.summary,
        metadata: {
          repository: message.repository,
          tags: message.tags
        }
      }
      
      store.dispatch(addMemories([chunk]))
    }
  }

  private handleMemoryUpdated(message: WebSocketMessage): void {
    if (message.chunk_id && message.data) {
      // Assuming message.data is a ConversationChunk with the updated data
      store.dispatch(updateMemory(message.data as ConversationChunk))
    }
  }

  private handleMemoryDeleted(message: WebSocketMessage): void {
    if (message.chunk_id) {
      store.dispatch(removeMemory(message.chunk_id))
    }
  }

  private handlePatternDetected(message: WebSocketMessage): void {
    logger.info('Pattern detected', { 
      patternType: typeof message.data,
      repository: message.repository 
    })
    // Could dispatch a notification or update patterns state
  }

  private handleSystemStatus(message: WebSocketMessage): void {
    logger.debug('System status update', { statusType: typeof message.data })
    // Could update system status in Redux store
  }

  private handleErrorMessage(message: WebSocketMessage): void {
    const error = new Error(message.content || 'WebSocket error')
    logger.error('WebSocket error message', error, { 
      errorType: message.content,
      dataType: typeof message.data 
    })
    this.stats.errors++
  }

  private startHeartbeat(): void {
    this.stopHeartbeat()
    
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.lastPingTime = Date.now()
        this.send({ type: 'ping' })
        
        // Set timeout for pong response
        this.pingTimeout = setTimeout(() => {
          logger.warn('Ping timeout - no pong received')
          this.ws?.close(4000, 'Ping timeout')
        }, 5000)
      }
    }, this.config.heartbeatInterval)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
    
    if (this.pingTimeout) {
      clearTimeout(this.pingTimeout)
      this.pingTimeout = null
    }
  }

  private scheduleReconnect(): void {
    if (this.isManuallyDisconnected || 
        this.reconnectCount >= this.config.maxReconnectAttempts) {
      if (this.reconnectCount >= this.config.maxReconnectAttempts) {
        logger.error('Max reconnection attempts reached')
        this.updateStatus('error')
      }
      return
    }
    
    const delay = this.calculateReconnectDelay()
    
    logger.info(`Scheduling reconnect in ${delay}ms`, { 
      attempt: this.reconnectCount + 1,
      maxAttempts: this.config.maxReconnectAttempts 
    })
    
    this.updateStatus('reconnecting')
    
    this.reconnectTimeout = setTimeout(() => {
      this.reconnectCount++
      this.stats.reconnectAttempts++
      this.connect()
    }, delay)
  }

  private calculateReconnectDelay(): number {
    const baseDelay = this.config.reconnectInterval
    const multiplier = Math.pow(
      this.config.reconnectBackoffMultiplier, 
      this.reconnectCount
    )
    const delay = baseDelay * multiplier
    
    return Math.min(delay, this.config.maxReconnectInterval)
  }

  private queueMessage(message: WebSocketMessage): void {
    this.messageQueue.push(message)
    
    // Limit queue size
    if (this.messageQueue.length > this.config.messageQueueSize) {
      this.messageQueue.shift()
    }
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0 && 
           this.ws?.readyState === WebSocket.OPEN) {
      const message = this.messageQueue.shift()
      if (message) {
        this.send(message)
      }
    }
  }

  private cleanup(): void {
    this.stopHeartbeat()
    
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }
  }

  private updateStatus(status: ConnectionStatus): void {
    this.stats.connectionStatus = status
    this.stats.connected = status === 'connected'
    
    // Map to store-compatible status
    const storeStatus = status === 'reconnecting' ? 'connecting' : status as 'connecting' | 'connected' | 'disconnected' | 'error'
    store.dispatch(setWebSocketStatus(storeStatus))
    this.updateStats()
  }

  private updateStats(): void {
    store.dispatch(updateWebSocketStats({
      messagesReceived: this.stats.messagesReceived,
      messagesSent: this.stats.messagesSent,
      reconnectAttempts: this.stats.reconnectAttempts
    }))
  }
}

// Singleton instance
let wsClient: ConsolidatedWebSocketClient | null = null

/**
 * Get or create WebSocket client instance
 */
export function getWebSocketClient(): ConsolidatedWebSocketClient | null {
  if (!wsClient && typeof window !== 'undefined') {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL
    const wsEnabled = process.env.NEXT_PUBLIC_ENABLE_WEBSOCKET === 'true'
    
    if (wsUrl && wsEnabled) {
      wsClient = new ConsolidatedWebSocketClient({
        url: wsUrl,
        protocols: ['v1.mcp.memory']
      })
    }
  }
  
  return wsClient
}

/**
 * React hook for WebSocket functionality
 */
export function useWebSocket() {
  const [isConnected, setIsConnected] = useState(false)
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const [lastMessage] = useState<WebSocketMessage | null>(null)
  
  const client = useMemo(() => getWebSocketClient(), [])
  
  useEffect(() => {
    if (!client) return
    
    // Auto-connect on mount
    client.connect()
    
    // Update local state from client stats
    const updateInterval = setInterval(() => {
      const stats = client.getStats()
      setIsConnected(stats.connected)
      setStatus(stats.connectionStatus)
    }, 1000)
    
    return () => {
      clearInterval(updateInterval)
      // Don't disconnect on unmount - let the singleton manage connection
    }
  }, [client])
  
  const sendMessage = useCallback((message: Omit<WebSocketMessage, 'timestamp'>) => {
    return client?.send(message) ?? false
  }, [client])
  
  const connect = useCallback(() => {
    client?.connect()
  }, [client])
  
  const disconnect = useCallback(() => {
    client?.disconnect()
  }, [client])
  
  const subscribe = useCallback((repository?: string, sessionId?: string) => {
    client?.subscribe(repository, sessionId)
  }, [client])
  
  const unsubscribe = useCallback((repository?: string, sessionId?: string) => {
    client?.unsubscribe(repository, sessionId)
  }, [client])
  
  const getStats = useCallback(() => {
    return client?.getStats() ?? null
  }, [client])
  
  return {
    isConnected,
    status,
    lastMessage,
    sendMessage,
    connect,
    disconnect,
    subscribe,
    unsubscribe,
    getStats,
    client
  }
}

// For backwards compatibility
export default ConsolidatedWebSocketClient