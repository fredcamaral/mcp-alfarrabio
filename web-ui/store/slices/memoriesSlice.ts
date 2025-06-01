import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { ConversationChunk, SearchResults, RelationshipResult } from '@/types/memory'

interface MemoriesState {
  // Current memory data
  memories: ConversationChunk[]
  selectedMemory?: ConversationChunk
  selectedMemoryId?: string
  
  // Search results
  searchResults?: SearchResults
  lastSearchQuery: string
  
  // Related memories
  relatedMemories: ConversationChunk[]
  relationships: RelationshipResult[]
  
  // UI state
  isLoading: boolean
  error?: string
  
  // Pagination
  currentPage: number
  totalPages: number
  hasNextPage: boolean
  
  // Selection state
  selectedMemoryIds: Set<string>
  
  // View state
  viewMode: 'list' | 'graph' | 'timeline'
}

const initialState: MemoriesState = {
  memories: [],
  relatedMemories: [],
  relationships: [],
  lastSearchQuery: '',
  isLoading: false,
  currentPage: 1,
  totalPages: 1,
  hasNextPage: false,
  selectedMemoryIds: new Set(),
  viewMode: 'list',
}

const memoriesSlice = createSlice({
  name: 'memories',
  initialState,
  reducers: {
    // Memory management
    setMemories: (state, action: PayloadAction<ConversationChunk[]>) => {
      state.memories = action.payload
      state.isLoading = false
      state.error = undefined
    },
    
    addMemories: (state, action: PayloadAction<ConversationChunk[]>) => {
      // Append new memories, avoiding duplicates
      const existingIds = new Set(state.memories.map(m => m.id))
      const newMemories = action.payload.filter(m => !existingIds.has(m.id))
      state.memories.push(...newMemories)
    },
    
    updateMemory: (state, action: PayloadAction<ConversationChunk>) => {
      const index = state.memories.findIndex(m => m.id === action.payload.id)
      if (index !== -1) {
        state.memories[index] = action.payload
      }
      
      // Update selected memory if it's the same one
      if (state.selectedMemory?.id === action.payload.id) {
        state.selectedMemory = action.payload
      }
    },
    
    removeMemory: (state, action: PayloadAction<string>) => {
      state.memories = state.memories.filter(m => m.id !== action.payload)
      state.selectedMemoryIds.delete(action.payload)
      
      // Clear selected memory if it was removed
      if (state.selectedMemory?.id === action.payload) {
        state.selectedMemory = undefined
        state.selectedMemoryId = undefined
      }
    },
    
    // Selection management
    setSelectedMemory: (state, action: PayloadAction<ConversationChunk | undefined>) => {
      state.selectedMemory = action.payload
      state.selectedMemoryId = action.payload?.id
    },
    
    setSelectedMemoryId: (state, action: PayloadAction<string | undefined>) => {
      state.selectedMemoryId = action.payload
      if (action.payload) {
        const memory = state.memories.find(m => m.id === action.payload)
        state.selectedMemory = memory
      } else {
        state.selectedMemory = undefined
      }
    },
    
    toggleMemorySelection: (state, action: PayloadAction<string>) => {
      if (state.selectedMemoryIds.has(action.payload)) {
        state.selectedMemoryIds.delete(action.payload)
      } else {
        state.selectedMemoryIds.add(action.payload)
      }
    },
    
    clearMemorySelection: (state) => {
      state.selectedMemoryIds.clear()
    },
    
    selectAllMemories: (state) => {
      state.memories.forEach(memory => {
        state.selectedMemoryIds.add(memory.id)
      })
    },
    
    // Search results
    setSearchResults: (state, action: PayloadAction<SearchResults>) => {
      state.searchResults = action.payload
      state.isLoading = false
      state.error = undefined
    },
    
    setLastSearchQuery: (state, action: PayloadAction<string>) => {
      state.lastSearchQuery = action.payload
    },
    
    clearSearchResults: (state) => {
      state.searchResults = undefined
      state.lastSearchQuery = ''
    },
    
    // Related memories and relationships
    setRelatedMemories: (state, action: PayloadAction<ConversationChunk[]>) => {
      state.relatedMemories = action.payload
    },
    
    setRelationships: (state, action: PayloadAction<RelationshipResult[]>) => {
      state.relationships = action.payload
    },
    
    // Loading and error states
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload
      if (action.payload) {
        state.error = undefined
      }
    },
    
    setError: (state, action: PayloadAction<string | undefined>) => {
      state.error = action.payload
      state.isLoading = false
    },
    
    // Pagination
    setPagination: (state, action: PayloadAction<{
      currentPage: number
      totalPages: number
      hasNextPage: boolean
    }>) => {
      state.currentPage = action.payload.currentPage
      state.totalPages = action.payload.totalPages
      state.hasNextPage = action.payload.hasNextPage
    },
    
    // View mode
    setViewMode: (state, action: PayloadAction<'list' | 'graph' | 'timeline'>) => {
      state.viewMode = action.payload
    },
    
    // Reset state
    resetMemories: (state) => {
      return {
        ...initialState,
        viewMode: state.viewMode, // Preserve view mode preference
      }
    },
  },
})

export const {
  setMemories,
  addMemories,
  updateMemory,
  removeMemory,
  setSelectedMemory,
  setSelectedMemoryId,
  toggleMemorySelection,
  clearMemorySelection,
  selectAllMemories,
  setSearchResults,
  setLastSearchQuery,
  clearSearchResults,
  setRelatedMemories,
  setRelationships,
  setLoading,
  setError,
  setPagination,
  setViewMode,
  resetMemories,
} = memoriesSlice.actions

export default memoriesSlice.reducer

// Selectors
export const selectAllMemories = (state: { memories: MemoriesState }) => state.memories.memories
export const selectSelectedMemory = (state: { memories: MemoriesState }) => state.memories.selectedMemory
export const selectSelectedMemoryId = (state: { memories: MemoriesState }) => state.memories.selectedMemoryId
export const selectSearchResults = (state: { memories: MemoriesState }) => state.memories.searchResults
export const selectRelatedMemories = (state: { memories: MemoriesState }) => state.memories.relatedMemories
export const selectMemoriesLoading = (state: { memories: MemoriesState }) => state.memories.isLoading
export const selectMemoriesError = (state: { memories: MemoriesState }) => state.memories.error
export const selectSelectedMemoryIds = (state: { memories: MemoriesState }) => state.memories.selectedMemoryIds
export const selectViewMode = (state: { memories: MemoriesState }) => state.memories.viewMode
export const selectPagination = (state: { memories: MemoriesState }) => ({
  currentPage: state.memories.currentPage,
  totalPages: state.memories.totalPages,
  hasNextPage: state.memories.hasNextPage,
})