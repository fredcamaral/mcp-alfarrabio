'use client'

import { useAppSelector, useAppDispatch } from '@/store/store'
import {
  selectQuery,
  selectRepository,
  selectSelectedTypes,
  selectTimeRange,
  selectTags,
  selectMinRelevance,
  selectMinConfidence,
  selectSortBy,
  selectSortOrder,
  selectOutcome,
  selectDifficulty,
  selectPresets,
  setQuery,
  setRepository,
  toggleType,
  setTimeRange,
  setMinRelevance,
  setMinConfidence,
  addTag,
  removeTag,
  setSortBy,
  setSortOrder,
  setOutcome,
  setDifficulty,
  applyPreset,
  resetToDefaults,
  selectAvailableRepositories,
  selectAvailableTags
} from '@/store/slices/filtersSlice'
import { ChunkType } from '@/types/memory'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Slider } from '@/components/ui/slider'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  X,
  Filter,
  RotateCcw,
  ChevronRight,
  Sparkles,
  Clock,
  GitBranch,
  Zap,
  TrendingUp,
  Shield,
  AlertCircle
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface FilterPanelProps {
  isOpen: boolean
  onClose: () => void
  className?: string
}

export function FilterPanel({ isOpen, onClose, className }: FilterPanelProps) {
  const dispatch = useAppDispatch()
  
  // Redux state
  const query = useAppSelector(selectQuery)
  const repository = useAppSelector(selectRepository)
  const selectedTypes = useAppSelector(selectSelectedTypes)
  const timeRange = useAppSelector(selectTimeRange)
  const tags = useAppSelector(selectTags)
  const minRelevance = useAppSelector(selectMinRelevance)
  const minConfidence = useAppSelector(selectMinConfidence)
  const sortBy = useAppSelector(selectSortBy)
  const sortOrder = useAppSelector(selectSortOrder)
  const outcome = useAppSelector(selectOutcome)
  const difficulty = useAppSelector(selectDifficulty)
  const presets = useAppSelector(selectPresets)
  const availableRepositories = useAppSelector(selectAvailableRepositories)
  const availableTags = useAppSelector(selectAvailableTags)

  // Available types
  const availableTypes: ChunkType[] = [
    'problem', 'solution', 'architecture_decision', 'session_summary',
    'code_change', 'discussion', 'analysis', 'verification', 'question'
  ]

  // Type icons mapping
  const typeIcons: Record<ChunkType, React.ReactNode> = {
    problem: <AlertCircle className="h-4 w-4" />,
    solution: <Sparkles className="h-4 w-4" />,
    architecture_decision: <Shield className="h-4 w-4" />,
    session_summary: <Clock className="h-4 w-4" />,
    code_change: <GitBranch className="h-4 w-4" />,
    discussion: <ChevronRight className="h-4 w-4" />,
    analysis: <TrendingUp className="h-4 w-4" />,
    verification: <Zap className="h-4 w-4" />,
    question: <AlertCircle className="h-4 w-4" />
  }

  const handleReset = () => {
    dispatch(resetToDefaults())
  }

  return (
    <>
      {/* Backdrop */}
      {isOpen && (
        <div 
          className="fixed inset-0 bg-background/80 backdrop-blur-sm z-40"
          onClick={onClose}
        />
      )}

      {/* Panel */}
      <div className={cn(
        "fixed right-0 top-0 h-full w-96 bg-background border-l shadow-lg transform transition-transform duration-300 z-50",
        isOpen ? "translate-x-0" : "translate-x-full",
        className
      )}>
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b">
            <div className="flex items-center space-x-2">
              <Filter className="h-5 w-5" />
              <h2 className="text-lg font-semibold">Advanced Filters</h2>
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleReset}
                className="h-8"
              >
                <RotateCcw className="h-4 w-4 mr-1" />
                Reset
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onClose}
                className="h-8 w-8 p-0"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Content */}
          <ScrollArea className="flex-1">
            <div className="p-6 space-y-6">
              {/* Presets */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Quick Presets</Label>
                <div className="grid grid-cols-2 gap-2">
                  {presets.map(preset => (
                    <Button
                      key={preset.id}
                      variant="outline"
                      size="sm"
                      onClick={() => dispatch(applyPreset(preset.id))}
                      className="justify-start"
                    >
                      <span className="truncate">{preset.name}</span>
                    </Button>
                  ))}
                </div>
              </div>

              <Separator />

              {/* Search Query */}
              <div className="space-y-3">
                <Label htmlFor="search-query" className="text-sm font-medium">
                  Search Query
                </Label>
                <Input
                  id="search-query"
                  placeholder="Enter search terms..."
                  value={query}
                  onChange={(e) => dispatch(setQuery(e.target.value))}
                />
              </div>

              {/* Repository Filter */}
              <div className="space-y-3">
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
                    {availableRepositories.map(repo => (
                      <SelectItem key={repo} value={repo}>
                        {repo.split('/').pop()}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Memory Types */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Memory Types</Label>
                <div className="space-y-2">
                  {availableTypes.map(type => (
                    <div
                      key={type}
                      className={cn(
                        "flex items-center space-x-3 p-2 rounded-md cursor-pointer transition-colors",
                        selectedTypes.includes(type) 
                          ? "bg-primary/10 text-primary" 
                          : "hover:bg-muted"
                      )}
                      onClick={() => dispatch(toggleType(type))}
                    >
                      <div className="flex items-center space-x-2 flex-1">
                        {typeIcons[type]}
                        <span className="text-sm">{type.replace('_', ' ')}</span>
                      </div>
                      {selectedTypes.includes(type) && (
                        <Badge variant="secondary" className="text-xs">
                          Selected
                        </Badge>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Time Range */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Time Range</Label>
                <Select
                  value={timeRange}
                  onValueChange={(value) => dispatch(setTimeRange(value as 'recent' | 'week' | 'month' | 'quarter' | 'year' | 'all' | 'custom'))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="recent">Recent (7 days)</SelectItem>
                    <SelectItem value="week">This week</SelectItem>
                    <SelectItem value="month">This month</SelectItem>
                    <SelectItem value="quarter">This quarter</SelectItem>
                    <SelectItem value="year">This year</SelectItem>
                    <SelectItem value="all">All time</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Tags */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Tags</Label>
                <div className="space-y-2">
                  <div className="flex flex-wrap gap-2">
                    {tags.map(tag => (
                      <Badge
                        key={tag}
                        variant="secondary"
                        className="gap-1"
                      >
                        #{tag}
                        <X
                          className="h-3 w-3 cursor-pointer hover:text-destructive"
                          onClick={() => dispatch(removeTag(tag))}
                        />
                      </Badge>
                    ))}
                  </div>
                  {availableTags.length > 0 && (
                    <Select
                      value=""
                      onValueChange={(value) => value && dispatch(addTag(value))}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Add tag..." />
                      </SelectTrigger>
                      <SelectContent>
                        {availableTags
                          .filter(tag => !tags.includes(tag))
                          .map(tag => (
                            <SelectItem key={tag} value={tag}>
                              #{tag}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </div>

              {/* Quality Filters */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Quality Filters</Label>
                
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm">Min Relevance</span>
                    <span className="text-sm text-muted-foreground">
                      {Math.round(minRelevance * 100)}%
                    </span>
                  </div>
                  <Slider
                    value={[minRelevance]}
                    onValueChange={([value]) => dispatch(setMinRelevance(value))}
                    min={0}
                    max={1}
                    step={0.1}
                    className="w-full"
                  />
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm">Min Confidence</span>
                    <span className="text-sm text-muted-foreground">
                      {Math.round(minConfidence * 100)}%
                    </span>
                  </div>
                  <Slider
                    value={[minConfidence]}
                    onValueChange={([value]) => dispatch(setMinConfidence(value))}
                    min={0}
                    max={1}
                    step={0.1}
                    className="w-full"
                  />
                </div>
              </div>

              {/* Advanced Filters */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Advanced</Label>
                
                <div className="space-y-2">
                  <Label htmlFor="outcome" className="text-xs">Outcome</Label>
                  <Select
                    value={outcome || 'all'}
                    onValueChange={(value) => dispatch(setOutcome(value === 'all' ? undefined : value as 'success' | 'in_progress' | 'failed' | 'abandoned'))}
                  >
                    <SelectTrigger id="outcome">
                      <SelectValue placeholder="Any outcome" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">Any outcome</SelectItem>
                      <SelectItem value="success">Success</SelectItem>
                      <SelectItem value="in_progress">In Progress</SelectItem>
                      <SelectItem value="failed">Failed</SelectItem>
                      <SelectItem value="abandoned">Abandoned</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="difficulty" className="text-xs">Difficulty</Label>
                  <Select
                    value={difficulty || 'all'}
                    onValueChange={(value) => dispatch(setDifficulty(value === 'all' ? undefined : value as 'simple' | 'moderate' | 'complex'))}
                  >
                    <SelectTrigger id="difficulty">
                      <SelectValue placeholder="Any difficulty" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">Any difficulty</SelectItem>
                      <SelectItem value="simple">Simple</SelectItem>
                      <SelectItem value="moderate">Moderate</SelectItem>
                      <SelectItem value="complex">Complex</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Sorting */}
              <div className="space-y-3">
                <Label className="text-sm font-medium">Sort By</Label>
                <div className="flex space-x-2">
                  <Select
                    value={sortBy}
                    onValueChange={(value) => dispatch(setSortBy(value as 'relevance' | 'date' | 'confidence' | 'type'))}
                  >
                    <SelectTrigger className="flex-1">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="relevance">Relevance</SelectItem>
                      <SelectItem value="date">Date</SelectItem>
                      <SelectItem value="confidence">Confidence</SelectItem>
                      <SelectItem value="type">Type</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => dispatch(setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc'))}
                    className="w-20"
                  >
                    {sortOrder === 'asc' ? '↑ Asc' : '↓ Desc'}
                  </Button>
                </div>
              </div>
            </div>
          </ScrollArea>

          {/* Footer */}
          <div className="px-6 py-4 border-t bg-muted/50">
            <Button 
              className="w-full" 
              onClick={onClose}
            >
              Apply Filters
            </Button>
          </div>
        </div>
      </div>
    </>
  )
}