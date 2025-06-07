'use client'

import { useRef, useEffect } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectQuery,
  selectRepository,
  selectSelectedTypes,
  selectTimeRange,
  selectTags,
  selectMinRelevance,
  selectHasActiveFilters,
  selectAvailableRepositories,
  selectAvailableTags,
  setQuery,
  setRepository,
  toggleType,
  setTimeRange,
  setMinRelevance,
  addTag,
  removeTag,
  resetToDefaults,
  addToSearchHistory
} from '@/store/slices/filtersSlice'
import { selectGlobalSearchFocused, setGlobalSearchFocused } from '@/store/slices/uiSlice'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ChunkType } from '@/types/memory'
import {
  Search,
  X
} from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface MemorySearchProps {
  className?: string
}

export function MemorySearch({ className }: MemorySearchProps) {
  const dispatch = useAppDispatch()
  const searchInputRef = useRef<HTMLInputElement>(null)
  
  // Redux state
  const query = useAppSelector(selectQuery)
  const globalSearchFocused = useAppSelector(selectGlobalSearchFocused)
  const repository = useAppSelector(selectRepository)
  const selectedTypes = useAppSelector(selectSelectedTypes)
  const timeRange = useAppSelector(selectTimeRange)
  const tags = useAppSelector(selectTags)
  const minRelevance = useAppSelector(selectMinRelevance)
  const hasActiveFilters = useAppSelector(selectHasActiveFilters)
  const availableRepositories = useAppSelector(selectAvailableRepositories)
  const availableTags = useAppSelector(selectAvailableTags)

  // Available types (from memory types)
  const availableTypes: ChunkType[] = [
    'problem', 'solution', 'architecture_decision', 'session_summary',
    'code_change', 'discussion', 'analysis', 'verification', 'question'
  ]

  // Use actual repositories if available, otherwise use defaults
  const repositories = availableRepositories.length > 0 ? availableRepositories : [
    process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory',
    'github.com/company/ai-assistant-core', 
    'github.com/company/legacy-system'
  ]

  // Use actual tags if available, otherwise use defaults
  const tagOptions = availableTags.length > 0 ? availableTags : [
    'authentication', 'database', 'react', 'typescript', 'api',
    'security', 'performance', 'bug-fix', 'architecture', 'testing'
  ]

  // Handle focus request from sidebar navigation
  useEffect(() => {
    if (globalSearchFocused && searchInputRef.current) {
      searchInputRef.current.focus()
      searchInputRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // Reset the focus flag
      dispatch(setGlobalSearchFocused(false))
    }
  }, [globalSearchFocused, dispatch])

  const handleQueryChange = (value: string) => {
    dispatch(setQuery(value))
  }

  const handleSearch = () => {
    if (query.trim()) {
      dispatch(addToSearchHistory(query))
    }
  }

  const clearFilters = () => {
    dispatch(resetToDefaults())
  }

  const toggleTag = (tag: string) => {
    if (tags.includes(tag)) {
      dispatch(removeTag(tag))
    } else {
      dispatch(addTag(tag))
    }
  }

  const removeType = (type: ChunkType) => {
    dispatch(toggleType(type))
  }

  const removeTagFromFilter = (tag: string) => {
    dispatch(removeTag(tag))
  }

  return (
    <div className={`space-y-4 ${className || ''}`}>
      {/* Main search input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          ref={searchInputRef}
          placeholder="Search memories, patterns, or insights..."
          value={query}
          onChange={(e) => handleQueryChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleSearch()
            }
          }}
          className="pl-10 pr-4"
        />
        {query && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => dispatch(setQuery(''))}
            className="absolute right-2 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
          >
            <X className="h-3 w-3" />
          </Button>
        )}
      </div>

      {/* Active filters display */}
      {hasActiveFilters && (
        <div className="flex flex-wrap gap-2 items-center">
          <span className="text-sm text-muted-foreground">Active filters:</span>

          {repository && (
            <Badge variant="secondary" className="gap-1">
              Repository: {repository.split('/').pop()}
              <X
                className="h-3 w-3 cursor-pointer"
                onClick={() => dispatch(setRepository(undefined))}
              />
            </Badge>
          )}

          {selectedTypes.map((type: ChunkType) => (
            <Badge key={type} variant="secondary" className="gap-1">
              {type.replace('_', ' ')}
              <X
                className="h-3 w-3 cursor-pointer"
                onClick={() => removeType(type)}
              />
            </Badge>
          ))}

          {tags.map(tag => (
            <Badge key={tag} variant="secondary" className="gap-1">
              #{tag}
              <X
                className="h-3 w-3 cursor-pointer"
                onClick={() => removeTagFromFilter(tag)}
              />
            </Badge>
          ))}

          {timeRange !== 'all' && (
            <Badge variant="secondary" className="gap-1">
              {timeRange}
              <X
                className="h-3 w-3 cursor-pointer"
                onClick={() => dispatch(setTimeRange('all'))}
              />
            </Badge>
          )}

          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="text-xs"
          >
            Clear all
          </Button>
        </div>
      )}

      {/* Advanced filters */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Repository filter */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Repository</Label>
          <Select
            value={repository || 'all'}
            onValueChange={(value) => dispatch(setRepository(value === 'all' ? undefined : value))}
          >
            <SelectTrigger>
              <SelectValue placeholder="All repositories" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All repositories</SelectItem>
              {repositories.map(repo => (
                <SelectItem key={repo} value={repo}>
                  {repo.split('/').pop()}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Time range filter */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Time Range</Label>
          <Select
            value={timeRange}
            onValueChange={(value) => dispatch(setTimeRange(value as 'recent' | 'week' | 'month' | 'all'))}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="recent">Recent (7 days)</SelectItem>
              <SelectItem value="week">This week</SelectItem>
              <SelectItem value="month">This month</SelectItem>
              <SelectItem value="all">All time</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Relevance filter */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">
            Min Relevance: {Math.round(minRelevance * 100)}%
          </Label>
          <input
            type="range"
            min="0"
            max="1"
            step="0.1"
            value={minRelevance}
            onChange={(e) => dispatch(setMinRelevance(parseFloat(e.target.value)))}
            className="w-full"
          />
        </div>

        {/* Quick actions */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Quick Actions</Label>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                dispatch(setTimeRange('recent'))
                if (!selectedTypes.includes('problem')) {
                  dispatch(toggleType('problem'))
                }
              }}
            >
              Recent Issues
            </Button>
          </div>
        </div>
      </div>

      <Separator />

      {/* Type filters */}
      <div className="space-y-2">
        <Label className="text-sm font-medium">Memory Types</Label>
        <div className="flex flex-wrap gap-2">
          {availableTypes.map(type => (
            <Badge
              key={type}
              variant={selectedTypes.includes(type) ? "default" : "outline"}
              className="cursor-pointer"
              onClick={() => dispatch(toggleType(type))}
            >
              {type.replace('_', ' ')}
            </Badge>
          ))}
        </div>
      </div>

      {/* Tag filters */}
      <div className="space-y-2">
        <Label className="text-sm font-medium">Tags</Label>
        <div className="flex flex-wrap gap-2">
          {tagOptions.map(tag => (
            <Badge
              key={tag}
              variant={tags.includes(tag) ? "default" : "outline"}
              className="cursor-pointer"
              onClick={() => toggleTag(tag)}
            >
              #{tag}
            </Badge>
          ))}
        </div>
      </div>

      {/* Search tips */}
      <div className="text-xs text-muted-foreground space-y-1">
        <p>💡 <strong>Search tips:</strong></p>
        <p>• Use quotes for exact phrases: &quot;authentication error&quot;</p>
        <p>• Combine filters for precise results</p>
        <p>• Higher relevance scores show more confident matches</p>
      </div>
    </div>
  )
}