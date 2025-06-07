/**
 * WebSocket Provider
 * 
 * Initializes and manages the global WebSocket connection
 */

'use client'

import { useEffect } from 'react'
import { getWebSocketClient, useWebSocket } from '@/lib/websocket/consolidated-client'
import { logger } from '@/lib/logger'
import { config } from '@/lib/env-validation'

interface WebSocketProviderProps {
  children: React.ReactNode
}

export function WebSocketProvider({ children }: WebSocketProviderProps) {
  useEffect(() => {
    // Only initialize if WebSocket is enabled
    if (!config.features.websocket) {
      logger.info('WebSocket disabled in configuration')
      return
    }

    const client = getWebSocketClient()
    
    if (!client) {
      logger.warn('WebSocket client not available - check configuration')
      return
    }

    // The client auto-connects on first use, but we can ensure it's connected
    logger.info('WebSocket provider initialized')
    
    // Subscribe to the default repository on mount
    const defaultRepo = config.repository.default
    if (defaultRepo) {
      client.subscribe(defaultRepo)
      logger.info('Subscribed to default repository:', { repository: defaultRepo })
    }

    // Cleanup on unmount
    return () => {
      if (defaultRepo) {
        client.unsubscribe(defaultRepo)
      }
    }
  }, [])

  // This provider doesn't render anything special, just ensures WebSocket is initialized
  return <>{children}</>
}

/**
 * Hook to use WebSocket with automatic provider check
 */
export function useWebSocketWithProvider() {
  const wsData = useWebSocket()
  
  if (!config.features.websocket) {
    return {
      ...wsData,
      isConnected: false,
      status: 'disconnected' as const,
      error: 'WebSocket disabled'
    }
  }
  
  return wsData
}