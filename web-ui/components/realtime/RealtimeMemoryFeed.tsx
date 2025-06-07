'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Bell, Activity, Zap } from 'lucide-react'
import { useRealtimeChunks, useRealtimePatterns } from '@/lib/graphql/hooks'
import { formatDistanceToNow } from 'date-fns'
import { logger } from '@/lib/logger'

interface RealtimeMemoryFeedProps {
  repository: string
  sessionId?: string
  maxItems?: number
}

export function RealtimeMemoryFeed({ 
  repository, 
  sessionId, 
  maxItems = 20 
}: RealtimeMemoryFeedProps) {
  const [notifications, setNotifications] = useState<string[]>([])
  
  // Use real-time hooks
  const { 
    data: chunks, 
    loading: chunksLoading, 
    hasNewChunk 
  } = useRealtimeChunks(repository, sessionId, maxItems)
  
  const { 
    data: patterns, 
    loading: patternsLoading, 
    hasNewPattern,
    latestPattern 
  } = useRealtimePatterns(repository)
  
  // Handle new chunk notifications
  useEffect(() => {
    if (hasNewChunk) {
      const notification = `New memory chunk added`
      setNotifications(prev => [notification, ...prev].slice(0, 10))
      
      logger.info('New chunk received via subscription', {
        component: 'RealtimeMemoryFeed',
        repository
      })
    }
  }, [hasNewChunk, repository])
  
  // Handle new pattern notifications
  useEffect(() => {
    if (hasNewPattern && latestPattern) {
      const notification = `New pattern detected: ${latestPattern.type}`
      setNotifications(prev => [notification, ...prev].slice(0, 10))
      
      logger.info('New pattern detected via subscription', {
        component: 'RealtimeMemoryFeed',
        repository,
        patternType: latestPattern.type
      })
    }
  }, [hasNewPattern, latestPattern, repository])
  
  const isLoading = chunksLoading || patternsLoading
  
  return (
    <div className="space-y-4">
      {/* Real-time Status */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Real-time Status
            </CardTitle>
            <div className="flex items-center gap-2">
              {hasNewChunk && (
                <Badge variant="default" className="animate-pulse">
                  <Zap className="h-3 w-3 mr-1" />
                  New Memory
                </Badge>
              )}
              {hasNewPattern && (
                <Badge variant="secondary" className="animate-pulse">
                  <Bell className="h-3 w-3 mr-1" />
                  Pattern Detected
                </Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-sm text-muted-foreground">
            {notifications.length > 0 ? (
              <div className="space-y-1">
                {notifications.slice(0, 3).map((notif, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <span className="w-2 h-2 bg-success rounded-full animate-pulse" />
                    {notif}
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 bg-muted rounded-full" />
                Waiting for real-time updates...
              </div>
            )}
          </div>
        </CardContent>
      </Card>
      
      {/* Recent Memories */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent Memories</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-20 bg-muted rounded animate-pulse" />
              ))}
            </div>
          ) : chunks?.listChunks && chunks.listChunks.length > 0 ? (
            <ScrollArea className="h-[400px]">
              <div className="space-y-3">
                {chunks.listChunks.slice(0, maxItems).map((chunk) => (
                  <Card key={chunk.id} className="p-3">
                    <div className="flex items-start justify-between mb-2">
                      <Badge variant="outline">{chunk.type}</Badge>
                      <span className="text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(chunk.timestamp), { addSuffix: true })}
                      </span>
                    </div>
                    <p className="text-sm line-clamp-2">{chunk.content}</p>
                    {chunk.metadata?.tags && chunk.metadata.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {chunk.metadata.tags.slice(0, 3).map((tag: string) => (
                          <Badge key={tag} variant="secondary" className="text-xs">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </Card>
                ))}
              </div>
            </ScrollArea>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              No memories found for this repository
            </div>
          )}
        </CardContent>
      </Card>
      
      {/* Pattern Activity */}
      {patterns?.getPatterns && patterns.getPatterns.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Active Patterns</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {patterns.getPatterns.slice(0, 5).map((pattern, idx) => (
                <div key={idx} className="flex items-center justify-between p-2 rounded-lg bg-muted/50">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${
                      hasNewPattern && latestPattern?.type === pattern.type 
                        ? 'bg-success animate-pulse' 
                        : 'bg-muted-foreground'
                    }`} />
                    <span className="text-sm font-medium">{pattern.type}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {pattern.count} occurrences
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {Math.round(pattern.confidence * 100)}% confidence
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}