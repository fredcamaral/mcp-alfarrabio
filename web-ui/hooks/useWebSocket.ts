/**
 * WebSocket Hook with Reconnection and Error Handling
 * 
 * Provides robust WebSocket connection management with automatic reconnection,
 * exponential backoff, connection status tracking, and event handling.
 */

import { useState, useEffect, useRef, useCallback } from 'react'

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error' | 'reconnecting'

export interface WebSocketMessage {
  type: string
  action?: string
  chunk_id?: string
  repository?: string
  session_id?: string
  content?: string
  summary?: string
  tags?: string[]
  timestamp: string
  data?: any
}

export interface WebSocketOptions {
  url: string
  reconnectInterval?: number
  maxReconnectAttempts?: number
  reconnectBackoffMultiplier?: number
  maxReconnectInterval?: number
  heartbeatInterval?: number
  messageQueueSize?: number
  protocols?: string[]
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
  onMessage?: (message: WebSocketMessage) => void
}

export interface UseWebSocketReturn {
  connectionStatus: ConnectionStatus
  lastMessage: WebSocketMessage | null
  sendMessage: (message: any) => boolean
  reconnect: () => void
  disconnect: () => void
  isConnected: boolean
  reconnectAttempts: number
  messageQueue: any[]
}

export function useWebSocket(options: WebSocketOptions): UseWebSocketReturn {
  const {
    url,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5,
    reconnectBackoffMultiplier = 1.5,
    maxReconnectInterval = 30000,
    heartbeatInterval = 30000,
    messageQueueSize = 100,
    protocols,
    onConnect,
    onDisconnect,
    onError,
    onMessage
  } = options

  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected')
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const [reconnectAttempts, setReconnectAttempts] = useState(0)
  const [messageQueue, setMessageQueue] = useState<any[]>([])

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const heartbeatTimeoutRef = useRef<number | null>(null)
  const heartbeatIntervalRef = useRef<number | null>(null)
  const isManuallyClosedRef = useRef(false)

  const clearTimeouts = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current)
      heartbeatTimeoutRef.current = null
    }
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current)
      heartbeatIntervalRef.current = null
    }
  }, [])

  const calculateReconnectDelay = useCallback((attemptNumber: number): number => {
    const delay = reconnectInterval * Math.pow(reconnectBackoffMultiplier, attemptNumber)
    return Math.min(delay, maxReconnectInterval)
  }, [reconnectInterval, reconnectBackoffMultiplier, maxReconnectInterval])

  const sendHeartbeat = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'ping',
        timestamp: new Date().toISOString()
      }))
    }
  }, [])

  const startHeartbeat = useCallback(() => {
    clearTimeouts()
    
    if (heartbeatInterval > 0) {
      heartbeatIntervalRef.current = setInterval(sendHeartbeat, heartbeatInterval)
    }
  }, [heartbeatInterval, sendHeartbeat, clearTimeouts])

  const flushMessageQueue = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN && messageQueue.length > 0) {
      messageQueue.forEach(message => {
        try {
          wsRef.current?.send(JSON.stringify(message))
        } catch (error) {
          console.error('Error sending queued message:', error)
        }
      })
      setMessageQueue([])
    }
  }, [messageQueue])

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.CONNECTING || 
        wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setConnectionStatus('connecting')
    
    try {
      const ws = new WebSocket(url, protocols)
      wsRef.current = ws

      ws.onopen = (event) => {
        console.log('WebSocket connected')
        setConnectionStatus('connected')
        setReconnectAttempts(0)
        isManuallyClosedRef.current = false
        
        startHeartbeat()
        flushMessageQueue()
        onConnect?.()
      }

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          setLastMessage(message)
          
          // Handle heartbeat responses
          if (message.type === 'pong') {
            // Reset heartbeat timeout
            if (heartbeatTimeoutRef.current) {
              clearTimeout(heartbeatTimeoutRef.current)
            }
            return
          }
          
          onMessage?.(message)
        } catch (error) {
          console.error('Error parsing WebSocket message:', error)
        }
      }

      ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason)
        clearTimeouts()
        
        if (!isManuallyClosedRef.current) {
          setConnectionStatus('disconnected')
          onDisconnect?.()
          
          // Attempt reconnection if not manually closed
          if (reconnectAttempts < maxReconnectAttempts) {
            const delay = calculateReconnectDelay(reconnectAttempts)
            setConnectionStatus('reconnecting')
            
            reconnectTimeoutRef.current = setTimeout(() => {
              setReconnectAttempts(prev => prev + 1)
              connect()
            }, delay)
          } else {
            setConnectionStatus('error')
            console.error('Max reconnection attempts reached')
          }
        }
      }

      ws.onerror = (event) => {
        console.error('WebSocket error:', event)
        setConnectionStatus('error')
        onError?.(event)
      }

    } catch (error) {
      console.error('Error creating WebSocket connection:', error)
      setConnectionStatus('error')
    }
  }, [
    url, 
    protocols, 
    reconnectAttempts, 
    maxReconnectAttempts, 
    calculateReconnectDelay,
    startHeartbeat,
    flushMessageQueue,
    onConnect,
    onDisconnect,
    onError,
    onMessage,
    clearTimeouts
  ])

  const disconnect = useCallback(() => {
    isManuallyClosedRef.current = true
    clearTimeouts()
    
    if (wsRef.current) {
      wsRef.current.close(1000, 'Manual disconnect')
      wsRef.current = null
    }
    
    setConnectionStatus('disconnected')
    setReconnectAttempts(0)
  }, [clearTimeouts])

  const reconnect = useCallback(() => {
    disconnect()
    setReconnectAttempts(0)
    setTimeout(connect, 100)
  }, [disconnect, connect])

  const sendMessage = useCallback((message: any): boolean => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      try {
        wsRef.current.send(JSON.stringify(message))
        return true
      } catch (error) {
        console.error('Error sending WebSocket message:', error)
        return false
      }
    } else {
      // Queue message for later sending
      setMessageQueue(prev => {
        const newQueue = [...prev, message]
        // Limit queue size
        if (newQueue.length > messageQueueSize) {
          newQueue.shift()
        }
        return newQueue
      })
      return false
    }
  }, [messageQueueSize])

  // Auto-connect on mount
  useEffect(() => {
    connect()
    
    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  // Handle page visibility changes
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && 
          connectionStatus === 'disconnected' && 
          !isManuallyClosedRef.current) {
        reconnect()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [connectionStatus, reconnect])

  // Handle network online/offline events
  useEffect(() => {
    const handleOnline = () => {
      if (connectionStatus === 'disconnected' && !isManuallyClosedRef.current) {
        reconnect()
      }
    }

    const handleOffline = () => {
      setConnectionStatus('disconnected')
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    
    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [connectionStatus, reconnect])

  return {
    connectionStatus,
    lastMessage,
    sendMessage,
    reconnect,
    disconnect,
    isConnected: connectionStatus === 'connected',
    reconnectAttempts,
    messageQueue
  }
}