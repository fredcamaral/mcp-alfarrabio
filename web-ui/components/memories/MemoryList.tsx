'use client'

import { useState } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import { 
  selectAllMemories, 
  selectSelectedMemoryIds,
  selectMemoriesLoading,
  selectViewMode,
  toggleMemorySelection,
  setSelectedMemory,
  setViewMode 
} from '@/store/slices/memoriesSlice'
import { cn, formatDate, getMemoryTypeIcon, truncateText, getConfidenceColor, formatConfidence } from '@/lib/utils'
import { ConversationChunk } from '@/types/memory'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  List,
  Grid3X3,
  Timeline,
  MoreHorizontal,
  Eye,
  Link,
  Trash2,
  Star,
  GitBranch,
  Clock,
  User
} from 'lucide-react'

interface MemoryListProps {
  className?: string
}

export function MemoryList({ className }: MemoryListProps) {
  const dispatch = useAppDispatch()
  const memories = useAppSelector(selectAllMemories)
  const selectedIds = useAppSelector(selectSelectedMemoryIds)
  const isLoading = useAppSelector(selectMemoriesLoading)
  const viewMode = useAppSelector(selectViewMode)

  const handleMemoryClick = (memory: ConversationChunk) => {
    dispatch(setSelectedMemory(memory))
  }

  const handleMemorySelect = (memoryId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    dispatch(toggleMemorySelection(memoryId))
  }

  const renderViewModeToggle = () => (
    <div className="flex items-center space-x-1 bg-muted rounded-lg p-1">
      <Button
        variant={viewMode === 'list' ? 'secondary' : 'ghost'}
        size="sm"
        onClick={() => dispatch(setViewMode('list'))}
        className="h-8 w-8 p-0"
      >
        <List className="h-4 w-4" />
      </Button>
      <Button
        variant={viewMode === 'graph' ? 'secondary' : 'ghost'}
        size="sm"
        onClick={() => dispatch(setViewMode('graph'))}
        className="h-8 w-8 p-0"
      >
        <Grid3X3 className="h-4 w-4" />
      </Button>
      <Button
        variant={viewMode === 'timeline' ? 'secondary' : 'ghost'}
        size="sm"
        onClick={() => dispatch(setViewMode('timeline'))}
        className="h-8 w-8 p-0"
      >
        <Timeline className="h-4 w-4" />
      </Button>
    </div>
  )

  const renderMemoryCard = (memory: ConversationChunk) => {
    const isSelected = selectedIds.has(memory.id)
    
    return (
      <Card 
        key={memory.id}
        className={cn(
          "cursor-pointer transition-all hover:shadow-md",
          isSelected && "ring-2 ring-primary"
        )}
        onClick={() => handleMemoryClick(memory)}
      >
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between">
            <div className="flex items-start space-x-3 flex-1 min-w-0">
              <Checkbox
                checked={isSelected}
                onCheckedChange={(checked) => handleMemorySelect(memory.id, {} as React.MouseEvent)}
                onClick={(e) => e.stopPropagation()}
                className="mt-1"
              />
              
              <div className="flex-1 min-w-0">
                <div className="flex items-center space-x-2 mb-2">
                  <span className="text-lg">
                    {getMemoryTypeIcon(memory.type)}
                  </span>
                  <Badge 
                    variant="outline" 
                    className={cn("text-xs", getConfidenceColor(memory.confidence || 0))}
                  >
                    {memory.type.replace('_', ' ')}
                  </Badge>
                  {memory.confidence && (
                    <Badge variant="secondary" className="text-xs">
                      {formatConfidence(memory.confidence)}
                    </Badge>
                  )}
                </div>
                
                <h3 className="font-medium text-foreground leading-tight mb-1">
                  {truncateText(memory.content, 100)}
                </h3>
                
                <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                  <div className="flex items-center space-x-1">
                    <Clock className="h-3 w-3" />
                    <span>{formatDate(memory.timestamp)}</span>
                  </div>
                  
                  {memory.repository && (
                    <div className="flex items-center space-x-1">
                      <GitBranch className="h-3 w-3" />
                      <span>{memory.repository}</span>
                    </div>
                  )}
                  
                  {memory.session_id && (
                    <div className="flex items-center space-x-1">
                      <User className="h-3 w-3" />
                      <span>{memory.session_id.slice(0, 8)}...</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
            
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button 
                  variant="ghost" 
                  size="sm" 
                  className="h-8 w-8 p-0"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem>
                  <Eye className="mr-2 h-4 w-4" />
                  View Details
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <Link className="mr-2 h-4 w-4" />
                  Show Relationships
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <Star className="mr-2 h-4 w-4" />
                  Add to Favorites
                </DropdownMenuItem>
                <DropdownMenuItem className="text-destructive">
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardHeader>
        
        <CardContent className="pt-0">
          {memory.tags && memory.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mb-3">
              {memory.tags.slice(0, 3).map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">
                  {tag}
                </Badge>
              ))}
              {memory.tags.length > 3 && (
                <Badge variant="secondary" className="text-xs">
                  +{memory.tags.length - 3} more
                </Badge>
              )}
            </div>
          )}
          
          <p className="text-sm text-muted-foreground line-clamp-2">
            {truncateText(memory.content, 200)}
          </p>
        </CardContent>
      </Card>
    )
  }

  const renderListView = () => (
    <div className="space-y-4">
      {memories.map(renderMemoryCard)}
    </div>
  )

  const renderGridView = () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {memories.map(renderMemoryCard)}
    </div>
  )

  const renderTimelineView = () => (
    <div className="space-y-6">
      {memories.map((memory, index) => (
        <div key={memory.id} className="flex">
          <div className="flex-shrink-0 w-24 text-right pr-4">
            <span className="text-xs text-muted-foreground">
              {formatDate(memory.timestamp)}
            </span>
          </div>
          <div className="flex-shrink-0 flex flex-col items-center">
            <div className="w-3 h-3 bg-primary rounded-full" />
            {index < memories.length - 1 && (
              <div className="w-0.5 h-16 bg-border mt-2" />
            )}
          </div>
          <div className="flex-1 pl-4">
            {renderMemoryCard(memory)}
          </div>
        </div>
      ))}
    </div>
  )

  const renderLoading = () => (
    <div className="space-y-4">
      {Array.from({ length: 5 }).map((_, i) => (
        <Card key={i}>
          <CardHeader>
            <div className="flex items-start space-x-3">
              <Skeleton className="h-4 w-4" />
              <div className="space-y-2 flex-1">
                <Skeleton className="h-4 w-3/4" />
                <div className="flex space-x-2">
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-3 w-16" />
                </div>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      ))}
    </div>
  )

  if (isLoading) {
    return (
      <div className={cn("space-y-6", className)}>
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-32" />
          <Skeleton className="h-8 w-24" />
        </div>
        {renderLoading()}
      </div>
    )
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Header with view toggle */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-foreground">
            {memories.length} Memories
          </h2>
          <p className="text-sm text-muted-foreground">
            {selectedIds.size > 0 && `${selectedIds.size} selected`}
          </p>
        </div>
        {renderViewModeToggle()}
      </div>

      {/* Memory list */}
      {memories.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-6xl mb-4">🧠</div>
          <h3 className="text-lg font-medium text-foreground mb-2">
            No memories found
          </h3>
          <p className="text-muted-foreground">
            Start a conversation to create your first memory
          </p>
        </div>
      ) : (
        <>
          {viewMode === 'list' && renderListView()}
          {viewMode === 'graph' && renderGridView()}
          {viewMode === 'timeline' && renderTimelineView()}
        </>
      )}
    </div>
  )
}