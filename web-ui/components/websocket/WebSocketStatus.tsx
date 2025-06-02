/**
 * WebSocket Status Component
 * 
 * Displays real-time WebSocket connection status with reconnection controls
 * and detailed connection information for debugging and user awareness.
 */

'use client'

import { useState } from 'react'
import { useWebSocket, type ConnectionStatus } from '@/hooks/useWebSocket'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Wifi,
  WifiOff,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Clock,
  Activity,
  Settings,
  Eye,
  EyeOff
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface WebSocketStatusProps {
  url?: string
  repository?: string
  sessionId?: string
  onMessage?: (message: any) => void
  className?: string
}

export function WebSocketStatus({
  url = `ws://${typeof window !== 'undefined' ? window.location.host : 'localhost:9080'}/ws`,
  repository,
  sessionId,
  onMessage,
  className
}: WebSocketStatusProps) {
  const [showDetails, setShowDetails] = useState(false)
  const [messageHistory, setMessageHistory] = useState<any[]>([])

  const webSocket = useWebSocket({
    url,
    reconnectInterval: 3000,
    maxReconnectAttempts: 5,
    heartbeatInterval: 30000,
    onConnect: () => {
      console.log('WebSocket connected')
      
      // Subscribe to repository and session if provided
      if (repository || sessionId) {
        webSocket.sendMessage({
          type: 'subscribe',
          repository,
          session_id: sessionId
        })
      }
    },
    onDisconnect: () => {
      console.log('WebSocket disconnected')
    },
    onError: (error) => {
      console.error('WebSocket error:', error)
    },
    onMessage: (message) => {
      setMessageHistory(prev => [...prev.slice(-49), message]) // Keep last 50 messages
      onMessage?.(message)
    }
  })

  const getStatusColor = (status: ConnectionStatus): string => {
    switch (status) {
      case 'connected':
        return 'text-green-600 bg-green-50'
      case 'connecting':
      case 'reconnecting':
        return 'text-yellow-600 bg-yellow-50'
      case 'disconnected':
        return 'text-gray-600 bg-gray-50'
      case 'error':
        return 'text-red-600 bg-red-50'
      default:
        return 'text-gray-600 bg-gray-50'
    }
  }

  const getStatusIcon = (status: ConnectionStatus) => {
    switch (status) {
      case 'connected':
        return <CheckCircle className="h-3 w-3" />
      case 'connecting':
      case 'reconnecting':
        return <RefreshCw className="h-3 w-3 animate-spin" />
      case 'disconnected':
        return <WifiOff className="h-3 w-3" />
      case 'error':
        return <AlertTriangle className="h-3 w-3" />
      default:
        return <WifiOff className="h-3 w-3" />
    }
  }

  const getStatusText = (status: ConnectionStatus): string => {
    switch (status) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'reconnecting':
        return `Reconnecting... (${webSocket.reconnectAttempts}/5)`
      case 'disconnected':
        return 'Disconnected'
      case 'error':
        return 'Connection Error'
      default:
        return 'Unknown'
    }
  }

  const formatTimestamp = (timestamp: string): string => {
    return new Date(timestamp).toLocaleTimeString()
  }

  const renderMessageHistory = () => (
    <div className="space-y-2 max-h-64 overflow-y-auto">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium">Recent Messages</h4>
        <Badge variant="secondary" className="text-xs">
          {messageHistory.length}/50
        </Badge>
      </div>
      
      {messageHistory.length === 0 ? (
        <div className="text-xs text-muted-foreground text-center py-4">
          No messages received yet
        </div>
      ) : (
        <div className="space-y-1">
          {messageHistory.slice(-10).reverse().map((message, index) => (
            <div key={index} className="text-xs bg-muted p-2 rounded">
              <div className="flex items-center justify-between mb-1">
                <Badge variant="outline" className="text-xs">
                  {message.type}
                </Badge>
                <span className="text-muted-foreground">
                  {formatTimestamp(message.timestamp)}
                </span>
              </div>
              {message.action && (
                <div className="text-muted-foreground">
                  Action: {message.action}
                </div>
              )}
              {message.chunk_id && (
                <div className="text-muted-foreground truncate">
                  Chunk: {message.chunk_id}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )

  return (
    <div className={cn('flex items-center gap-2', className)}>
      {/* Status Indicator */}
      <Popover>
        <PopoverTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              'h-8 px-2 text-xs',
              getStatusColor(webSocket.connectionStatus)
            )}
          >
            {getStatusIcon(webSocket.connectionStatus)}
            <span className="ml-1 hidden sm:inline">
              {getStatusText(webSocket.connectionStatus)}
            </span>
          </Button>
        </PopoverTrigger>
        
        <PopoverContent className="w-80 p-0" align="end">
          <Card className="border-0 shadow-none">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2">
                <Activity className="h-4 w-4" />
                WebSocket Connection
              </CardTitle>
            </CardHeader>
            
            <CardContent className="space-y-4">
              {/* Connection Status */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Status:</span>
                  <Badge
                    variant="secondary"
                    className={cn('text-xs', getStatusColor(webSocket.connectionStatus))}
                  >
                    {getStatusIcon(webSocket.connectionStatus)}
                    <span className="ml-1">
                      {getStatusText(webSocket.connectionStatus)}
                    </span>
                  </Badge>
                </div>
                
                {webSocket.reconnectAttempts > 0 && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Attempts:</span>
                    <span className="text-sm">{webSocket.reconnectAttempts}/5</span>
                  </div>
                )}
                
                {webSocket.messageQueue.length > 0 && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Queued:</span>
                    <span className="text-sm">{webSocket.messageQueue.length} messages</span>
                  </div>
                )}
              </div>

              {/* Subscription Info */}
              {(repository || sessionId) && (
                <div className="space-y-2">
                  <h4 className="text-sm font-medium">Subscriptions</h4>
                  {repository && (
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-muted-foreground">Repository:</span>
                      <span className="text-xs truncate max-w-32">{repository}</span>
                    </div>
                  )}
                  {sessionId && (
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-muted-foreground">Session:</span>
                      <span className="text-xs truncate max-w-32">{sessionId}</span>
                    </div>
                  )}
                </div>
              )}

              {/* Last Message */}
              {webSocket.lastMessage && (
                <div className="space-y-2">
                  <h4 className="text-sm font-medium">Last Message</h4>
                  <div className="text-xs bg-muted p-2 rounded">
                    <div className="flex items-center justify-between">
                      <Badge variant="outline" className="text-xs">
                        {webSocket.lastMessage.type}
                      </Badge>
                      <span className="text-muted-foreground">
                        {formatTimestamp(webSocket.lastMessage.timestamp)}
                      </span>
                    </div>
                    {webSocket.lastMessage.action && (
                      <div className="mt-1 text-muted-foreground">
                        {webSocket.lastMessage.action}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Controls */}
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={webSocket.reconnect}
                  disabled={webSocket.connectionStatus === 'connecting'}
                  className="flex-1"
                >
                  <RefreshCw className="h-3 w-3 mr-1" />
                  Reconnect
                </Button>
                
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setShowDetails(!showDetails)}
                >
                  {showDetails ? (
                    <EyeOff className="h-3 w-3" />
                  ) : (
                    <Eye className="h-3 w-3" />
                  )}
                </Button>
              </div>

              {/* Message History */}
              {showDetails && renderMessageHistory()}
            </CardContent>
          </Card>
        </PopoverContent>
      </Popover>

      {/* Quick Actions */}
      {webSocket.connectionStatus === 'error' && (
        <Button
          size="sm"
          variant="outline"
          onClick={webSocket.reconnect}
          className="h-8 px-2"
        >
          <RefreshCw className="h-3 w-3" />
        </Button>
      )}
    </div>
  )
}