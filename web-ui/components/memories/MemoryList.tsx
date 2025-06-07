'use client'

import { useEffect } from 'react'
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
  setMemories,
  removeMemory
} from '@/store/slices/memoriesSlice'
import {
  selectQuery,
  selectRepository,
  selectSelectedTypes,
  selectTimeRange,
  selectTags,
  selectMinRelevance,
  selectHasActiveFilters,
  setAvailableTags
} from '@/store/slices/filtersSlice'
import { addNotification } from '@/store/slices/uiSlice'
import { useListChunks, useSearchMemories, useGraphQLError, useDeleteChunk } from '@/lib/graphql/hooks'
import { cn, formatDate, getMemoryTypeIcon, getConfidenceColor, formatConfidence } from '@/lib/utils'
import { ConversationChunk } from '@/types/memory'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
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

interface MemoryListProps {
  className?: string
}

export function MemoryList({ className }: MemoryListProps) {
  const dispatch = useAppDispatch()
  const memories = useAppSelector(selectMemories)
  const selectedMemoryIds = useAppSelector(selectSelectedMemoryIds)
  const isLoading = useAppSelector(selectMemoriesLoading)
  const error = useAppSelector(selectMemoriesError)
  const { handleError } = useGraphQLError()
  
  // Delete mutation
  const [deleteChunk, { loading: deletingChunk }] = useDeleteChunk()

  // Filter state from Redux
  const searchQuery = useAppSelector(selectQuery)
  const repository = useAppSelector(selectRepository) || process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory'
  const selectedTypes = useAppSelector(selectSelectedTypes)
  const timeRange = useAppSelector(selectTimeRange)
  const selectedTags = useAppSelector(selectTags)
  const minRelevance = useAppSelector(selectMinRelevance)
  const hasActiveFilters = useAppSelector(selectHasActiveFilters)

  // Convert time range to date filter
  const getTimeRangeDate = () => {
    const now = new Date()
    switch (timeRange) {
      case 'recent':
        return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000) // 7 days
      case 'week':
        return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000) // 7 days
      case 'month':
        return new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000) // 30 days
      default:
        return null
    }
  }

  // GraphQL queries
  const {
    loading: chunksLoading,
    refetch: refetchChunks
  } = useListChunks(repository, 100, 0, {
    onCompleted: (data: { listChunks: ConversationChunk[] }) => {
      if (data?.listChunks) {
        dispatch(setMemories(data.listChunks))
        
        // Update available tags based on actual data
        const allTags = Array.from(new Set(data.listChunks.flatMap(m => m.metadata?.tags || [])))
        dispatch(setAvailableTags(allTags))
      }
    },
    onError: (error: Error) => {
      const errorMessage = handleError(error)
      dispatch(setError(errorMessage))
    }
  })

  // Search query (only when there's a search term or filters)
  const {
    loading: searchLoading,
    refetch: refetchSearch
  } = useSearchMemories(
    {
      query: searchQuery,
      repository,
      types: selectedTypes.length > 0 ? selectedTypes : undefined,
      tags: selectedTags.length > 0 ? selectedTags : undefined,
      limit: 50,
      minRelevanceScore: minRelevance
    },
    {
      skip: !hasActiveFilters,
      onCompleted: (data: { search: { chunks: Array<{ chunk: ConversationChunk }> } }) => {
        if (data?.search?.chunks) {
          // Extract chunks from scored results
          const chunks = data.search.chunks.map(scoredChunk => scoredChunk.chunk)
          dispatch(setMemories(chunks))
        }
      },
      onError: (error: Error) => {
        const errorMessage = handleError(error)
        dispatch(setError(errorMessage))
      }
    }
  )

  // Update loading state
  useEffect(() => {
    const loading = hasActiveFilters ? searchLoading : chunksLoading
    dispatch(setLoading(loading))
  }, [chunksLoading, searchLoading, hasActiveFilters, dispatch])

  // Filter memories based on time range (for non-search results)
  const filteredMemories = hasActiveFilters ? memories : memories.filter(memory => {
    const timeRangeDate = getTimeRangeDate()
    if (timeRangeDate && new Date(memory.timestamp) < timeRangeDate) {
      return false
    }
    return true
  })

  const handleMemoryClick = (memory: ConversationChunk) => {
    dispatch(setSelectedMemory(memory))
  }

  const handleMemorySelect = (memoryId: string, event: React.MouseEvent) => {
    event.stopPropagation()
    dispatch(toggleMemorySelection(memoryId))
  }

  const handleSelectAll = () => {
    if (selectedMemoryIds.length === filteredMemories.length) {
      dispatch(clearMemorySelection())
    } else {
      dispatch(selectAllMemories())
    }
  }

  const handleRefresh = () => {
    if (hasActiveFilters) {
      refetchSearch()
    } else {
      refetchChunks()
    }
  }

  const handleDeleteMemory = async (memoryId: string, event: React.MouseEvent) => {
    event.stopPropagation()
    
    try {
      await deleteChunk({ variables: { id: memoryId } })
      
      // Remove from Redux store
      dispatch(removeMemory(memoryId))
      
      // Show success notification
      dispatch(addNotification({
        type: 'success',
        title: 'Memory Deleted',
        message: 'The memory has been successfully deleted',
        duration: 3000
      }))
    } catch (error) {
      const errorMessage = handleError(error)
      dispatch(addNotification({
        type: 'error',
        title: 'Failed to Delete Memory',
        message: errorMessage,
        duration: 5000
      }))
    }
  }

  const renderMemoryCard = (memory: ConversationChunk) => {
    const isSelected = selectedMemoryIds.includes(memory.id)
    const confidence = memory.metadata?.confidence?.score || 0
    const tags = memory.metadata?.tags || []

    return (
      <Card
        key={memory.id}
        className={cn(
          "cursor-pointer transition-all duration-200 hover:shadow-md",
          isSelected && "ring-2 ring-primary ring-offset-2"
        )}
        onClick={() => handleMemoryClick(memory)}
      >
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between">
            <div className="flex items-center space-x-3 flex-1">
              <button
                onClick={(e) => handleMemorySelect(memory.id, e)}
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
                <DropdownMenuItem 
                  className="text-destructive"
                  onClick={(e) => handleDeleteMemory(memory.id, e)}
                  disabled={deletingChunk}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {deletingChunk ? 'Deleting...' : 'Delete'}
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
  }

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
      {/* Controls */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          {/* Results summary */}
          <div className="text-sm text-muted-foreground">
            {isLoading ? (
              <span>Loading memories...</span>
            ) : (
              <span>
                {filteredMemories.length} {filteredMemories.length === 1 ? 'memory' : 'memories'}
                {searchQuery && ` matching "${searchQuery}"`}
              </span>
            )}
          </div>

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
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleSelectAll}
                >
                  {selectedMemoryIds.length === filteredMemories.length ? 'Deselect All' : 'Select All'}
                </Button>
                <span className="text-sm text-muted-foreground">
                  {selectedMemoryIds.length} selected
                </span>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Memory List */}
      {isLoading ? (
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
      ) : filteredMemories.length === 0 ? (
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
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {filteredMemories.map(renderMemoryCard)}
        </div>
      )}
    </div>
  )
}