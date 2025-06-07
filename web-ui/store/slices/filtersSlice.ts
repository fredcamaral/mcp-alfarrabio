import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { ChunkType } from '@/types/memory'

interface FilterState {
  // Search query
  query: string
  
  // Repository filter
  repository?: string
  availableRepositories: string[]
  
  // Type filters
  selectedTypes: ChunkType[]
  availableTypes: ChunkType[]
  
  // Time range
  timeRange: 'recent' | 'week' | 'month' | 'quarter' | 'year' | 'all' | 'custom'
  customTimeRange?: {
    start: string
    end: string
  }
  
  // Quality filters
  minRelevance: number
  minConfidence: number
  
  // Advanced filters
  tags: string[]
  availableTags: string[]
  outcome?: 'success' | 'in_progress' | 'failed' | 'abandoned'
  difficulty?: 'simple' | 'moderate' | 'complex'
  
  // Session filters
  sessionId?: string
  
  // Sorting
  sortBy: 'relevance' | 'date' | 'confidence' | 'type'
  sortOrder: 'asc' | 'desc'
  
  // Pagination
  limit: number
  offset: number
  
  // Quick filters (presets)
  activePreset?: string
  presets: FilterPreset[]
  
  // Search history
  searchHistory: string[]
  
  // Filter state
  hasActiveFilters: boolean
}

interface FilterPreset {
  id: string
  name: string
  description?: string
  filters: Partial<FilterState>
  isDefault?: boolean
}

const defaultPresets: FilterPreset[] = [
  {
    id: 'recent-problems',
    name: 'Recent Problems',
    description: 'Problems from the last week',
    filters: {
      selectedTypes: ['problem'],
      timeRange: 'week',
      sortBy: 'date',
      sortOrder: 'desc',
    }
  },
  {
    id: 'solutions',
    name: 'Solutions',
    description: 'All solutions and fixes',
    filters: {
      selectedTypes: ['solution'],
      sortBy: 'confidence',
      sortOrder: 'desc',
    }
  },
  {
    id: 'architecture',
    name: 'Architecture Decisions',
    description: 'Design decisions and architectural choices',
    filters: {
      selectedTypes: ['architecture_decision'],
      sortBy: 'date',
      sortOrder: 'desc',
    }
  },
  {
    id: 'high-confidence',
    name: 'High Confidence',
    description: 'Memories with high confidence scores',
    filters: {
      minConfidence: 0.8,
      sortBy: 'confidence',
      sortOrder: 'desc',
    }
  }
]

const initialState: FilterState = {
  query: '',
  availableRepositories: [],
  selectedTypes: [],
  availableTypes: [
    'problem',
    'solution', 
    'architecture_decision',
    'session_summary',
    'code_change',
    'discussion',
    'analysis',
    'verification',
    'question'
  ],
  timeRange: 'all',
  minRelevance: 0.3,
  minConfidence: 0.0,
  tags: [],
  availableTags: [],
  sortBy: 'relevance',
  sortOrder: 'desc',
  limit: 20,
  offset: 0,
  presets: defaultPresets,
  searchHistory: [],
  hasActiveFilters: false,
}

const filtersSlice = createSlice({
  name: 'filters',
  initialState,
  reducers: {
    // Query
    setQuery: (state, action: PayloadAction<string>) => {
      state.query = action.payload
      state.offset = 0 // Reset pagination when query changes
      state.hasActiveFilters = state.query !== '' || hasOtherActiveFilters(state)
    },
    
    // Repository
    setRepository: (state, action: PayloadAction<string | undefined>) => {
      state.repository = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters({ ...state, repository: action.payload })
    },
    
    setAvailableRepositories: (state, action: PayloadAction<string[]>) => {
      state.availableRepositories = action.payload
    },
    
    // Types
    setSelectedTypes: (state, action: PayloadAction<ChunkType[]>) => {
      state.selectedTypes = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters({ ...state, selectedTypes: action.payload })
    },
    
    toggleType: (state, action: PayloadAction<ChunkType>) => {
      const type = action.payload
      if (state.selectedTypes.includes(type)) {
        state.selectedTypes = state.selectedTypes.filter(t => t !== type)
      } else {
        state.selectedTypes.push(type)
      }
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    clearTypes: (state) => {
      state.selectedTypes = []
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    // Time range
    setTimeRange: (state, action: PayloadAction<FilterState['timeRange']>) => {
      state.timeRange = action.payload
      if (action.payload !== 'custom') {
        state.customTimeRange = undefined
      }
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    setCustomTimeRange: (state, action: PayloadAction<{ start: string; end: string }>) => {
      state.customTimeRange = action.payload
      state.timeRange = 'custom'
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    // Quality filters
    setMinRelevance: (state, action: PayloadAction<number>) => {
      state.minRelevance = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    setMinConfidence: (state, action: PayloadAction<number>) => {
      state.minConfidence = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    // Tags
    setTags: (state, action: PayloadAction<string[]>) => {
      state.tags = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    addTag: (state, action: PayloadAction<string>) => {
      if (!state.tags.includes(action.payload)) {
        state.tags.push(action.payload)
        state.offset = 0
        state.hasActiveFilters = hasAnyActiveFilters(state)
      }
    },
    
    removeTag: (state, action: PayloadAction<string>) => {
      state.tags = state.tags.filter(tag => tag !== action.payload)
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    setAvailableTags: (state, action: PayloadAction<string[]>) => {
      state.availableTags = action.payload
    },
    
    // Advanced filters
    setOutcome: (state, action: PayloadAction<FilterState['outcome']>) => {
      state.outcome = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    setDifficulty: (state, action: PayloadAction<FilterState['difficulty']>) => {
      state.difficulty = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    setSessionId: (state, action: PayloadAction<string | undefined>) => {
      state.sessionId = action.payload
      state.offset = 0
      state.hasActiveFilters = hasAnyActiveFilters(state)
    },
    
    // Sorting
    setSortBy: (state, action: PayloadAction<FilterState['sortBy']>) => {
      state.sortBy = action.payload
      state.offset = 0
    },
    
    setSortOrder: (state, action: PayloadAction<FilterState['sortOrder']>) => {
      state.sortOrder = action.payload
      state.offset = 0
    },
    
    toggleSortOrder: (state) => {
      state.sortOrder = state.sortOrder === 'asc' ? 'desc' : 'asc'
      state.offset = 0
    },
    
    // Pagination
    setLimit: (state, action: PayloadAction<number>) => {
      state.limit = action.payload
      state.offset = 0
    },
    
    setOffset: (state, action: PayloadAction<number>) => {
      state.offset = action.payload
    },
    
    nextPage: (state) => {
      state.offset += state.limit
    },
    
    prevPage: (state) => {
      state.offset = Math.max(0, state.offset - state.limit)
    },
    
    resetPagination: (state) => {
      state.offset = 0
    },
    
    // Presets
    applyPreset: (state, action: PayloadAction<string>) => {
      const preset = state.presets.find(p => p.id === action.payload)
      if (preset) {
        Object.assign(state, {
          ...state,
          ...preset.filters,
          activePreset: preset.id,
          offset: 0,
        })
        state.hasActiveFilters = hasAnyActiveFilters(state)
      }
    },
    
    savePreset: (state, action: PayloadAction<Omit<FilterPreset, 'id'>>) => {
      const preset: FilterPreset = {
        id: Date.now().toString(),
        ...action.payload,
      }
      state.presets.push(preset)
    },
    
    removePreset: (state, action: PayloadAction<string>) => {
      state.presets = state.presets.filter(p => p.id !== action.payload)
      if (state.activePreset === action.payload) {
        state.activePreset = undefined
      }
    },
    
    // Search history
    addToSearchHistory: (state, action: PayloadAction<string>) => {
      const query = action.payload.trim()
      if (query && !state.searchHistory.includes(query)) {
        state.searchHistory.unshift(query)
        // Keep only last 20 searches
        state.searchHistory = state.searchHistory.slice(0, 20)
      }
    },
    
    clearSearchHistory: (state) => {
      state.searchHistory = []
    },
    
    // Reset filters
    resetFilters: (state) => {
      return {
        ...initialState,
        availableRepositories: state.availableRepositories,
        availableTags: state.availableTags,
        presets: state.presets,
        searchHistory: state.searchHistory,
      }
    },
    
    resetToDefaults: (state) => {
      state.query = ''
      state.repository = undefined
      state.selectedTypes = []
      state.timeRange = 'all'
      state.customTimeRange = undefined
      state.minRelevance = 0.3
      state.minConfidence = 0.0
      state.tags = []
      state.outcome = undefined
      state.difficulty = undefined
      state.sessionId = undefined
      state.sortBy = 'relevance'
      state.sortOrder = 'desc'
      state.offset = 0
      state.activePreset = undefined
      state.hasActiveFilters = false
    },
  },
})

// Helper functions
function hasOtherActiveFilters(state: FilterState): boolean {
  return (
    !!state.repository ||
    state.selectedTypes.length > 0 ||
    state.timeRange !== 'all' ||
    state.minRelevance > 0.3 ||
    state.minConfidence > 0.0 ||
    state.tags.length > 0 ||
    !!state.outcome ||
    !!state.difficulty ||
    !!state.sessionId
  )
}

function hasAnyActiveFilters(state: FilterState): boolean {
  return state.query !== '' || hasOtherActiveFilters(state)
}

export const {
  setQuery,
  setRepository,
  setAvailableRepositories,
  setSelectedTypes,
  toggleType,
  clearTypes,
  setTimeRange,
  setCustomTimeRange,
  setMinRelevance,
  setMinConfidence,
  setTags,
  addTag,
  removeTag,
  setAvailableTags,
  setOutcome,
  setDifficulty,
  setSessionId,
  setSortBy,
  setSortOrder,
  toggleSortOrder,
  setLimit,
  setOffset,
  nextPage,
  prevPage,
  resetPagination,
  applyPreset,
  savePreset,
  removePreset,
  addToSearchHistory,
  clearSearchHistory,
  resetFilters,
  resetToDefaults,
} = filtersSlice.actions

export default filtersSlice.reducer

// Selectors
export const selectFilters = (state: { filters: FilterState }) => state.filters
export const selectQuery = (state: { filters: FilterState }) => state.filters.query
export const selectRepository = (state: { filters: FilterState }) => state.filters.repository
export const selectSelectedTypes = (state: { filters: FilterState }) => state.filters.selectedTypes
export const selectTimeRange = (state: { filters: FilterState }) => state.filters.timeRange
export const selectTags = (state: { filters: FilterState }) => state.filters.tags
export const selectSortBy = (state: { filters: FilterState }) => state.filters.sortBy
export const selectSortOrder = (state: { filters: FilterState }) => state.filters.sortOrder
export const selectHasActiveFilters = (state: { filters: FilterState }) => state.filters.hasActiveFilters
export const selectPresets = (state: { filters: FilterState }) => state.filters.presets
export const selectSearchHistory = (state: { filters: FilterState }) => state.filters.searchHistory
export const selectMinRelevance = (state: { filters: FilterState }) => state.filters.minRelevance
export const selectMinConfidence = (state: { filters: FilterState }) => state.filters.minConfidence
export const selectOutcome = (state: { filters: FilterState }) => state.filters.outcome
export const selectDifficulty = (state: { filters: FilterState }) => state.filters.difficulty
export const selectAvailableRepositories = (state: { filters: FilterState }) => state.filters.availableRepositories
export const selectAvailableTags = (state: { filters: FilterState }) => state.filters.availableTags