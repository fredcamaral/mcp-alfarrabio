'use client'

import React, { useState, useEffect, useMemo, useCallback, memo } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectMemories,
  selectSelectedMemoryIds,
  selectMemoriesLoading,
  selectMemoriesError,
  setSelectedMemory,
  toggleMemorySelection,
  clearMemorySelection,
  selectAllMemories,
  setLoading,
  setError,
  setMemories
} from '@/store/slices/memoriesSlice'
import { useListChunks, useSearchMemories, useGraphQLError } from '@/lib/graphql/hooks'
import { cn, formatDate, getMemoryTypeIcon, getConfidenceColor, formatConfidence } from '@/lib/utils'
import { ConversationChunk } from '@/types/memory'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Search,
  MoreVertical,
  Star,
  Share,
  Download,
  Trash2,
  CheckSquare,
  Square,
  RefreshCw,
  AlertCircle
} from 'lucide-react'
import { 
  useDebounce, 
  useVirtualList, 
  useIntersectionObserver,
  useComponentPerformance 
} from '@/hooks/usePerformanceOptimizations'

interface MemoryListProps {
  className?: string
}

// Memoized memory card component
const MemoryCard = memo(({ 
  memory, 
  isSelected, 
  onMemoryClick, 
  onMemorySelect 
}: {
  memory: ConversationChunk
  isSelected: boolean
  onMemoryClick: (memory: ConversationChunk) => void
  onMemorySelect: (memoryId: string, event: React.MouseEvent) => void
}) => {
  const confidence = memory.metadata?.confidence?.score || 0
  const tags = memory.metadata?.tags || []
  
  const handleClick = useCallback(() => {
    onMemoryClick(memory)
  }, [memory, onMemoryClick])

  const handleSelect = useCallback((event: React.MouseEvent) => {
    onMemorySelect(memory.id, event)
  }, [memory.id, onMemorySelect])

  return (
    <Card
      className={cn(
        "cursor-pointer transition-all duration-200 hover:shadow-md",
        isSelected && "ring-2 ring-primary ring-offset-2"
      )}
      onClick={handleClick}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center space-x-3 flex-1">
            <button
              onClick={handleSelect}
              className="flex-shrink-0"
            >
              {isSelected ? (
                <CheckSquare className="h-4 w-4 text-primary" />
              ) : (
                <Square className="h-4 w-4 text-muted-foreground hover:text-foreground" />
              )}
            </button>

            <div className="flex items-center space-x-2 flex-1 min-w-0">
              <span className="text-lg">
                {getMemoryTypeIcon(memory.type)}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center space-x-2">
                  <Badge variant="outline" className="text-xs">
                    {memory.type.replace('_', ' ')}
                  </Badge>
                  {confidence > 0 && (
                    <Badge
                      variant="secondary"
                      className={cn("text-xs", getConfidenceColor(confidence))}
                    >
                      {formatConfidence(confidence)}
                    </Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground mt-1">
                  {formatDate(memory.timestamp)}
                </p>
              </div>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem>
                <Star className="mr-2 h-4 w-4" />
                Add to favorites
              </DropdownMenuItem>
              <DropdownMenuItem>
                <Share className="mr-2 h-4 w-4" />
                Share
              </DropdownMenuItem>
              <DropdownMenuItem>
                <Download className="mr-2 h-4 w-4" />
                Export
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <p className="text-sm text-foreground line-clamp-3 mb-3">
          {memory.content}
        </p>

        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {tags.slice(0, 3).map((tag: string) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
            {tags.length > 3 && (
              <Badge variant="secondary" className="text-xs">
                +{tags.length - 3} more
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
})

MemoryCard.displayName = 'MemoryCard'

// Memoized loading skeleton
const LoadingSkeleton = memo(() => (
  <div className="grid grid-cols-1 gap-4">
    {Array.from({ length: 6 }).map((_, i) => (
      <Card key={i} className="animate-pulse">
        <CardHeader>
          <div className="h-4 bg-muted rounded w-3/4" />
          <div className="h-3 bg-muted rounded w-1/2" />
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="h-3 bg-muted rounded" />
            <div className="h-3 bg-muted rounded w-2/3" />
          </div>
        </CardContent>
      </Card>
    ))}
  </div>
))

LoadingSkeleton.displayName = 'LoadingSkeleton'

// Memoized empty state
const EmptyState = memo(({ searchQuery }: { searchQuery: string }) => (
  <div className="text-center py-12">
    <div className="text-6xl mb-4">🧠</div>
    <h3 className="text-lg font-medium text-foreground mb-2">
      {searchQuery ? 'No memories found' : 'No memories yet'}
    </h3>
    <p className="text-sm text-muted-foreground">
      {searchQuery
        ? `No memories match your search for "${searchQuery}"`
        : 'Start a conversation to create your first memory'
      }
    </p>
  </div>
))

EmptyState.displayName = 'EmptyState'

// Virtual list item component
const VirtualListItem = memo(({ 
  memory, 
  index, 
  isSelected, 
  onMemoryClick, 
  onMemorySelect 
}: {
  memory: ConversationChunk
  index: number
  isSelected: boolean
  onMemoryClick: (memory: ConversationChunk) => void
  onMemorySelect: (memoryId: string, event: React.MouseEvent) => void
}) => {
  const [ref, isVisible] = useIntersectionObserver({
    threshold: 0.1,
    rootMargin: '100px',
  })

  return (
    <div
      ref={ref}
      style={{
        position: 'absolute',
        top: index * 180, // Approximate height of each card
        left: 0,
        right: 0,
        height: 180,
      }}
    >
      {isVisible && (
        <MemoryCard
          memory={memory}
          isSelected={isSelected}
          onMemoryClick={onMemoryClick}
          onMemorySelect={onMemorySelect}
        />
      )}
    </div>
  )
})

VirtualListItem.displayName = 'VirtualListItem'

export function MemoryList({ className }: MemoryListProps) {
  const { trackUpdate } = useComponentPerformance('MemoryList')
  const dispatch = useAppDispatch()
  const memories = useAppSelector(selectMemories)
  const selectedMemoryIds = useAppSelector(selectSelectedMemoryIds)
  const isLoading = useAppSelector(selectMemoriesLoading)
  const error = useAppSelector(selectMemoriesError)
  const { handleError } = useGraphQLError()

  // Local state for filtering and search
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [selectedType, setSelectedType] = useState<string>('all')
  const [repository] = useState(process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory')
  const [enableVirtualization, setEnableVirtualization] = useState(false)

  // Debounce search query for performance
  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  // GraphQL queries
  const {
    loading: chunksLoading,
    refetch: refetchChunks
  } = useListChunks(repository, 100, 0, {
    onCompleted: (data: { listChunks: ConversationChunk[] }) => {
      if (data?.listChunks) {
        dispatch(setMemories(data.listChunks))
        trackUpdate('chunks-loaded')
      }
    },
    onError: (error: Error) => {
      const errorMessage = handleError(error)
      dispatch(setError(errorMessage))
      trackUpdate('chunks-error')
    }
  })

  // Search query (only when there's a search term)
  const {
    loading: searchLoading
  } = useSearchMemories(
    {
      query: debouncedSearchQuery,
      repository,
      types: selectedType !== 'all' ? [selectedType] : undefined,
      tags: selectedTags.length > 0 ? selectedTags : undefined,
      limit: 50,
      minRelevanceScore: 0.7
    },
    {
      skip: !debouncedSearchQuery.trim(),
      onCompleted: (data: { search: { chunks: Array<{ chunk: ConversationChunk }> } }) => {
        if (data?.search?.chunks) {
          const chunks = data.search.chunks.map(scoredChunk => scoredChunk.chunk)
          dispatch(setMemories(chunks))
          trackUpdate('search-completed')
        }
      },
      onError: (error: Error) => {
        const errorMessage = handleError(error)
        dispatch(setError(errorMessage))
        trackUpdate('search-error')
      }
    }
  )

  // Memoized filtering
  const filteredMemories = useMemo(() => {
    if (debouncedSearchQuery) return memories

    return memories.filter(memory => {
      const tags = memory.metadata?.tags || []
      const matchesType = selectedType === 'all' || memory.type === selectedType
      const matchesTags = selectedTags.length === 0 || selectedTags.every((tag: string) => tags.includes(tag))
      return matchesType && matchesTags
    })
  }, [memories, selectedType, selectedTags, debouncedSearchQuery])

  // Memoized tags and types
  const { allTags, allTypes } = useMemo(() => {
    const tags = Array.from(new Set(memories.flatMap(m => m.metadata?.tags || [])))
    const types = Array.from(new Set(memories.map(m => m.type)))
    return { allTags: tags, allTypes: types }
  }, [memories])

  // Enable virtualization for large lists
  useEffect(() => {
    setEnableVirtualization(filteredMemories.length > 50)
  }, [filteredMemories.length])

  // Virtual list configuration
  const {
    visibleItems,
    totalHeight,
    onScroll
  } = useVirtualList({
    items: filteredMemories,
    itemHeight: 180,
    containerHeight: 600,
    overscan: 5,
  })

  // Update loading state
  useEffect(() => {
    const loading = debouncedSearchQuery ? searchLoading : chunksLoading
    dispatch(setLoading(loading))
  }, [chunksLoading, searchLoading, debouncedSearchQuery, dispatch])

  // Optimized handlers with useCallback
  const handleSearch = useCallback((query: string) => {
    setSearchQuery(query)
    trackUpdate('search-input')
  }, [trackUpdate])

  const handleMemoryClick = useCallback((memory: ConversationChunk) => {
    dispatch(setSelectedMemory(memory))
    trackUpdate('memory-selected')
  }, [dispatch, trackUpdate])

  const handleMemorySelect = useCallback((memoryId: string, event: React.MouseEvent) => {
    event.stopPropagation()
    dispatch(toggleMemorySelection(memoryId))
    trackUpdate('memory-toggle')
  }, [dispatch, trackUpdate])

  const handleSelectAll = useCallback(() => {
    if (selectedMemoryIds.length === filteredMemories.length) {
      dispatch(clearMemorySelection())
    } else {
      dispatch(selectAllMemories())
    }
    trackUpdate('select-all')
  }, [selectedMemoryIds.length, filteredMemories.length, dispatch, trackUpdate])

  const handleRefresh = useCallback(() => {
    if (debouncedSearchQuery) {
      setSearchQuery(debouncedSearchQuery + ' ')
      setSearchQuery(debouncedSearchQuery.trim())
    } else {
      refetchChunks()
    }
    trackUpdate('refresh')
  }, [debouncedSearchQuery, refetchChunks, trackUpdate])

  const handleTypeChange = useCallback((value: string) => {
    setSelectedType(value)
    trackUpdate('type-filter')
  }, [trackUpdate])

  const handleTagsChange = useCallback((value: string) => {
    setSelectedTags(value === 'all' ? [] : value.split(','))
    trackUpdate('tags-filter')
  }, [trackUpdate])

  if (error) {
    return (
      <div className={cn("flex items-center justify-center h-64", className)}>
        <div className="text-center space-y-4">
          <AlertCircle className="h-12 w-12 text-destructive mx-auto" />
          <div>
            <h3 className="text-lg font-medium">Failed to load memories</h3>
            <p className="text-sm text-muted-foreground mt-1">{error}</p>
            <Button onClick={handleRefresh} className="mt-4">
              <RefreshCw className="mr-2 h-4 w-4" />
              Try Again
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Filters and Controls */}
      <div className="space-y-4">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search memories..."
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            className="pl-10"
          />
        </div>

        {/* Filters */}
        <div className="flex items-center space-x-4">
          <Select value={selectedType} onValueChange={handleTypeChange}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder="All types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              {allTypes.map((type) => (
                <SelectItem key={type} value={type}>
                  {type.replace('_', ' ')}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select 
            value={selectedTags.length > 0 ? selectedTags.join(',') : 'all'} 
            onValueChange={handleTagsChange}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="All tags" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All tags</SelectItem>
              {allTags.map((tag: string) => (
                <SelectItem key={tag} value={tag}>
                  {tag}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="flex-1" />

          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isLoading}
            >
              <RefreshCw className={cn("h-4 w-4 mr-2", isLoading && "animate-spin")} />
              Refresh
            </Button>

            {selectedMemoryIds.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleSelectAll}
              >
                {selectedMemoryIds.length === filteredMemories.length ? 'Deselect All' : 'Select All'}
              </Button>
            )}
          </div>
        </div>

        {/* Results summary */}
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <div>
            {isLoading ? (
              <span>Loading memories...</span>
            ) : (
              <span>
                {filteredMemories.length} {filteredMemories.length === 1 ? 'memory' : 'memories'}
                {debouncedSearchQuery && ` matching "${debouncedSearchQuery}"`}
                {enableVirtualization && ' (virtualized)'}
              </span>
            )}
          </div>
          <div>
            {selectedMemoryIds.length > 0 && `${selectedMemoryIds.length} selected`}
          </div>
        </div>
      </div>

      {/* Memory List */}
      {isLoading ? (
        <LoadingSkeleton />
      ) : filteredMemories.length === 0 ? (
        <EmptyState searchQuery={debouncedSearchQuery} />
      ) : enableVirtualization ? (
        <div 
          className="relative overflow-auto"
          style={{ height: 600 }}
          onScroll={onScroll}
        >
          <div style={{ height: totalHeight, position: 'relative' }}>
            {visibleItems.map(({ item: memory, index }) => (
              <VirtualListItem
                key={memory.id}
                memory={memory}
                index={index}
                isSelected={selectedMemoryIds.includes(memory.id)}
                onMemoryClick={handleMemoryClick}
                onMemorySelect={handleMemorySelect}
              />
            ))}
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {filteredMemories.map((memory) => (
            <MemoryCard
              key={memory.id}
              memory={memory}
              isSelected={selectedMemoryIds.includes(memory.id)}
              onMemoryClick={handleMemoryClick}
              onMemorySelect={handleMemorySelect}
            />
          ))}
        </div>
      )}
    </div>
  )
}