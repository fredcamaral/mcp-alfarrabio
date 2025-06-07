/**
 * WebSocket Status Component
 * 
 * Displays real-time WebSocket connection status with reconnection controls
 * and detailed connection information for debugging and user awareness.
 */

'use client'

import { useWebSocket } from '@/lib/websocket/consolidated-client'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ErrorBoundary } from '@/components/error/ErrorBoundary'
import { logger } from '@/lib/logger'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  CheckCircle2,
  AlertCircle,
  WifiOff,
  RefreshCw,
  Activity,
  Clock,
  MessageSquare,
  Send
} from 'lucide-react'

interface WebSocketStatusProps {
  className?: string
}

export function WebSocketStatus({ className }: WebSocketStatusProps) {
  return (
    <ErrorBoundary
      onError={(error) => logger.error('WebSocket Status component error:', error)}
      enableRetry={true}
      fallback={
        <div className={cn("h-8 w-8 rounded-full bg-muted flex items-center justify-center", className)}>
          <WifiOff className="h-3 w-3 text-muted-foreground" />
        </div>
      }
    >
      <WebSocketStatusInner className={className} />
    </ErrorBoundary>
  )
}

function WebSocketStatusInner({ className }: WebSocketStatusProps) {
  const { status, lastMessage, connect, disconnect, isConnected } = useWebSocket()

  const getStatusIcon = () => {
    switch (status) {
      case 'connected':
        return <CheckCircle2 className="h-3 w-3" />
      case 'connecting':
        return <RefreshCw className="h-3 w-3 animate-spin" />
      case 'error':
        return <AlertCircle className="h-3 w-3" />
      default:
        return <WifiOff className="h-3 w-3" />
    }
  }

  const getStatusColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-success'
      case 'connecting':
        return 'bg-warning'
      case 'error':
        return 'bg-destructive'
      default:
        return 'bg-muted-foreground'
    }
  }

  const getStatusText = () => {
    switch (status) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'error':
        return 'Connection Error'
      default:
        return 'Disconnected'
    }
  }


  const formatLastActivity = (timestamp: number) => {
    const diff = Date.now() - timestamp
    const seconds = Math.floor(diff / 1000)

    if (seconds < 60) {
      return 'Just now'
    } else if (seconds < 3600) {
      return `${Math.floor(seconds / 60)}m ago`
    } else {
      return `${Math.floor(seconds / 3600)}h ago`
    }
  }

  const getMessageTypeColor = (type: string) => {
    switch (type) {
      case 'memory_created':
        return 'text-success'
      case 'memory_updated':
        return 'text-info'
      case 'memory_deleted':
        return 'text-destructive'
      case 'pattern_detected':
        return 'text-purple'
      case 'error':
        return 'text-destructive'
      default:
        return 'text-muted-foreground'
    }
  }

  return (
    <TooltipProvider>
      <Popover>
        <PopoverTrigger asChild>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className={cn(
                  "relative h-8 w-8 p-0 rounded-full",
                  className
                )}
              >
                <div className="relative">
                  {getStatusIcon()}
                  <div
                    className={cn(
                      "absolute -top-1 -right-1 h-2 w-2 rounded-full",
                      getStatusColor()
                    )}
                  />
                </div>
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>WebSocket: {getStatusText()}</p>
            </TooltipContent>
          </Tooltip>
        </PopoverTrigger>

        <PopoverContent className="w-80" align="end">
          <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
              <h4 className="font-medium">WebSocket Connection</h4>
              <Badge
                variant={isConnected ? "default" : "secondary"}
                className={cn(
                  "text-xs",
                  isConnected ? "bg-success" : "bg-muted-foreground"
                )}
              >
                {getStatusText()}
              </Badge>
            </div>

            {/* Connection Stats */}
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="flex items-center space-x-2">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="font-medium">Uptime</p>
                  <p className="text-muted-foreground">N/A</p>
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <Activity className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="font-medium">Last Activity</p>
                  <p className="text-muted-foreground">{lastMessage ? formatLastActivity(new Date(lastMessage.timestamp).getTime()) : 'N/A'}</p>
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <MessageSquare className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="font-medium">Received</p>
                  <p className="text-muted-foreground">N/A</p>
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <Send className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="font-medium">Sent</p>
                  <p className="text-muted-foreground">N/A</p>
                </div>
              </div>
            </div>

            {/* Reconnection Info */}
            {/* Reconnection info removed since not available from hook */}

            {/* Last Message */}
            {lastMessage && (
              <div className="space-y-2">
                <h5 className="font-medium text-sm">Last Message</h5>
                <div className="p-2 bg-muted rounded-md text-xs">
                  <div className="flex items-center justify-between mb-1">
                    <span className={cn("font-medium", getMessageTypeColor(lastMessage.type))}>
                      {lastMessage.type.replace('_', ' ').toUpperCase()}
                    </span>
                    <span className="text-muted-foreground">
                      {new Date(lastMessage.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  {(lastMessage.data != null) && (
                    <p className="text-muted-foreground">
                      {String(typeof lastMessage.data === 'string'
                        ? lastMessage.data
                        : JSON.stringify(lastMessage.data).slice(0, 100) + '...'
                      )}
                    </p>
                  )}
                </div>
              </div>
            )}

            {/* Connection Actions */}
            <div className="flex space-x-2">
              {isConnected ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={disconnect}
                  className="flex-1"
                >
                  Disconnect
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={connect}
                  className="flex-1"
                >
                  {status === 'connecting' ? (
                    <>
                      <RefreshCw className="mr-2 h-3 w-3 animate-spin" />
                      Connecting...
                    </>
                  ) : (
                    'Connect'
                  )}
                </Button>
              )}
            </div>

            {/* Connection URL */}
            <div className="text-xs text-muted-foreground">
              <p>Endpoint: {process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:9080/ws'}</p>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  )
}