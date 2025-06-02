'use client'

import { useState, useEffect, useCallback } from 'react'
import { useAppDispatch, useAppSelector } from '@/store/store'
import { Search, Filter, X, Calendar, Tag, GitBranch, Users, Clock, Zap } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Slider } from '@/components/ui/slider'
import { DatePicker } from '@/components/ui/date-picker'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

// Define search filter types
export interface SearchFilters {
  query: string
  types: string[]
  repositories: string[]
  tags: string[]
  sessionIds: string[]
  dateRange: {
    from?: Date
    to?: Date
  }
  confidenceRange: [number, number]
  recency: 'recent' | 'last_month' | 'all_time'
  sortBy: 'relevance' | 'date' | 'confidence'
  sortOrder: 'asc' | 'desc'
  limit: number
}

interface MemorySearchProps {
  onSearch: (filters: SearchFilters) => void
  isLoading?: boolean
  className?: string
}

const DEFAULT_FILTERS: SearchFilters = {
  query: '',
  types: [],
  repositories: [],
  tags: [],
  sessionIds: [],
  dateRange: {},
  confidenceRange: [0, 100],
  recency: 'recent',
  sortBy: 'relevance',
  sortOrder: 'desc',
  limit: 50
}

const MEMORY_TYPES = [
  { value: 'discussion', label: 'Discussion', icon: '💬' },
  { value: 'solution', label: 'Solution', icon: '💡' },
  { value: 'problem', label: 'Problem', icon: '❓' },
  { value: 'architecture_decision', label: 'Architecture Decision', icon: '🏗️' },
  { value: 'bug_report', label: 'Bug Report', icon: '🐛' },
  { value: 'feature_request', label: 'Feature Request', icon: '✨' },
  { value: 'code_review', label: 'Code Review', icon: '👀' },
  { value: 'documentation', label: 'Documentation', icon: '📚' }
]

const RECENCY_OPTIONS = [
  { value: 'recent', label: 'Recent (Last 7 days)' },
  { value: 'last_month', label: 'Last Month' },
  { value: 'all_time', label: 'All Time' }
]

const SORT_OPTIONS = [
  { value: 'relevance', label: 'Relevance' },
  { value: 'date', label: 'Date' },
  { value: 'confidence', label: 'Confidence' }
]

export function MemorySearch({ onSearch, isLoading = false, className }: MemorySearchProps) {
  const [filters, setFilters] = useState<SearchFilters>(DEFAULT_FILTERS)
  const [isFiltersOpen, setIsFiltersOpen] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)

  // Debounced search effect
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      if (filters.query.trim() || hasActiveFilters()) {
        onSearch(filters)
      }
    }, 300)

    return () => clearTimeout(timeoutId)
  }, [filters, onSearch])

  const hasActiveFilters = useCallback(() => {
    return (
      filters.types.length > 0 ||
      filters.repositories.length > 0 ||
      filters.tags.length > 0 ||
      filters.sessionIds.length > 0 ||
      filters.dateRange.from ||
      filters.dateRange.to ||
      filters.confidenceRange[0] > 0 ||
      filters.confidenceRange[1] < 100 ||
      filters.recency !== 'recent'
    )
  }, [filters])

  const updateFilter = <K extends keyof SearchFilters>(
    key: K,
    value: SearchFilters[K]
  ) => {
    setFilters(prev => ({ ...prev, [key]: value }))
  }

  const addFilter = (type: keyof Pick<SearchFilters, 'types' | 'repositories' | 'tags' | 'sessionIds'>, value: string) => {
    setFilters(prev => ({
      ...prev,
      [type]: [...prev[type], value]
    }))
  }

  const removeFilter = (type: keyof Pick<SearchFilters, 'types' | 'repositories' | 'tags' | 'sessionIds'>, value: string) => {
    setFilters(prev => ({
      ...prev,
      [type]: prev[type].filter(item => item !== value)
    }))
  }

  const clearAllFilters = () => {
    setFilters(DEFAULT_FILTERS)
  }

  const handleSearch = () => {
    onSearch(filters)
  }

  const renderActiveFilters = () => {
    const activeFilters: Array<{ type: string, value: string, onRemove: () => void }> = []

    // Add type filters
    filters.types.forEach(type => {
      const typeInfo = MEMORY_TYPES.find(t => t.value === type)
      activeFilters.push({
        type: 'Type',
        value: typeInfo ? `${typeInfo.icon} ${typeInfo.label}` : type,
        onRemove: () => removeFilter('types', type)
      })
    })

    // Add repository filters
    filters.repositories.forEach(repo => {
      activeFilters.push({
        type: 'Repository',
        value: repo,
        onRemove: () => removeFilter('repositories', repo)
      })
    })

    // Add tag filters
    filters.tags.forEach(tag => {
      activeFilters.push({
        type: 'Tag',
        value: `#${tag}`,
        onRemove: () => removeFilter('tags', tag)
      })
    })

    // Add date range filter
    if (filters.dateRange.from || filters.dateRange.to) {
      const from = filters.dateRange.from?.toLocaleDateString() || 'Start'
      const to = filters.dateRange.to?.toLocaleDateString() || 'End'
      activeFilters.push({
        type: 'Date',
        value: `${from} - ${to}`,
        onRemove: () => updateFilter('dateRange', {})
      })
    }

    // Add recency filter
    if (filters.recency !== 'recent') {
      const recencyOption = RECENCY_OPTIONS.find(r => r.value === filters.recency)
      activeFilters.push({
        type: 'Recency',
        value: recencyOption?.label || filters.recency,
        onRemove: () => updateFilter('recency', 'recent')
      })
    }

    return activeFilters.length > 0 ? (
      <div className="flex flex-wrap gap-2 mb-4">
        {activeFilters.map((filter, index) => (
          <Badge 
            key={index} 
            variant="secondary" 
            className="px-3 py-1 text-xs flex items-center gap-2"
          >
            <span className="text-muted-foreground">{filter.type}:</span>
            <span>{filter.value}</span>
            <Button
              variant="ghost"
              size="sm"
              className="h-4 w-4 p-0 hover:bg-transparent"
              onClick={filter.onRemove}
            >
              <X className="h-3 w-3" />
            </Button>
          </Badge>
        ))}
        {activeFilters.length > 1 && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearAllFilters}
            className="h-7 px-2 text-xs text-muted-foreground"
          >
            Clear all
          </Button>
        )}
      </div>
    ) : null
  }

  const renderFiltersPopover = () => (
    <PopoverContent className="w-96 p-0" align="end">
      <div className="p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">Search Filters</h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={clearAllFilters}
            className="h-8 px-2 text-xs"
          >
            Clear all
          </Button>
        </div>

        <Separator />

        {/* Memory Types */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Memory Types</Label>
          <div className="grid grid-cols-2 gap-2">
            {MEMORY_TYPES.map(type => (
              <div key={type.value} className="flex items-center space-x-2">
                <Checkbox
                  id={`type-${type.value}`}
                  checked={filters.types.includes(type.value)}
                  onCheckedChange={(checked) => {
                    if (checked) {
                      addFilter('types', type.value)
                    } else {
                      removeFilter('types', type.value)
                    }
                  }}
                />
                <Label 
                  htmlFor={`type-${type.value}`}
                  className="text-xs cursor-pointer flex items-center gap-1"
                >
                  <span>{type.icon}</span>
                  <span>{type.label}</span>
                </Label>
              </div>
            ))}
          </div>
        </div>

        <Separator />

        {/* Date Range */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Date Range</Label>
          <div className="flex items-center space-x-2">
            <DatePicker
              date={filters.dateRange.from}
              onSelect={(date) => updateFilter('dateRange', { ...filters.dateRange, from: date })}
              placeholder="From date"
              className="flex-1"
            />
            <span className="text-muted-foreground">to</span>
            <DatePicker
              date={filters.dateRange.to}
              onSelect={(date) => updateFilter('dateRange', { ...filters.dateRange, to: date })}
              placeholder="To date"
              className="flex-1"
            />
          </div>
        </div>

        <Separator />

        {/* Recency */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Recency</Label>
          <Select 
            value={filters.recency} 
            onValueChange={(value) => updateFilter('recency', value as SearchFilters['recency'])}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {RECENCY_OPTIONS.map(option => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <Separator />

        {/* Confidence Range */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">
            Confidence Range: {filters.confidenceRange[0]}% - {filters.confidenceRange[1]}%
          </Label>
          <Slider
            value={filters.confidenceRange}
            onValueChange={(value) => updateFilter('confidenceRange', value as [number, number])}
            max={100}
            min={0}
            step={5}
            className="w-full"
          />
        </div>

        <Separator />

        {/* Sort Options */}
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium">Sort By</Label>
            <Select 
              value={filters.sortBy} 
              onValueChange={(value) => updateFilter('sortBy', value as SearchFilters['sortBy'])}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SORT_OPTIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium">Order</Label>
            <Select 
              value={filters.sortOrder} 
              onValueChange={(value) => updateFilter('sortOrder', value as 'asc' | 'desc')}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="desc">Descending</SelectItem>
                <SelectItem value="asc">Ascending</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Separator />

        {/* Limit */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Results Limit: {filters.limit}</Label>
          <Slider
            value={[filters.limit]}
            onValueChange={(value) => updateFilter('limit', value[0])}
            max={200}
            min={10}
            step={10}
            className="w-full"
          />
        </div>
      </div>
    </PopoverContent>
  )

  return (
    <div className={cn("space-y-4", className)}>
      {/* Search Input */}
      <div className="flex items-center space-x-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search memories... (e.g., 'authentication', 'GraphQL', 'bug fix')"
            value={filters.query}
            onChange={(e) => updateFilter('query', e.target.value)}
            className="pl-10 pr-4"
          />
        </div>
        
        <Popover open={isFiltersOpen} onOpenChange={setIsFiltersOpen}>
          <PopoverTrigger asChild>
            <Button 
              variant="outline" 
              size="sm"
              className={cn(
                "relative",
                hasActiveFilters() && "border-primary bg-primary/5"
              )}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
              {hasActiveFilters() && (
                <Badge 
                  variant="destructive" 
                  className="absolute -top-2 -right-2 h-5 w-5 p-0 flex items-center justify-center text-xs"
                >
                  {filters.types.length + filters.repositories.length + filters.tags.length + 
                   (filters.dateRange.from || filters.dateRange.to ? 1 : 0) +
                   (filters.recency !== 'recent' ? 1 : 0)}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          {renderFiltersPopover()}
        </Popover>

        <Button 
          onClick={handleSearch}
          disabled={isLoading}
          size="sm"
        >
          {isLoading ? (
            <>
              <Zap className="h-4 w-4 mr-2 animate-spin" />
              Searching...
            </>
          ) : (
            <>
              <Search className="h-4 w-4 mr-2" />
              Search
            </>
          )}
        </Button>
      </div>

      {/* Active Filters */}
      {renderActiveFilters()}

      {/* Search Stats */}
      {filters.query && (
        <div className="text-xs text-muted-foreground">
          Searching for: <span className="font-medium">"{filters.query}"</span>
          {hasActiveFilters() && " with filters applied"}
        </div>
      )}
    </div>
  )
}