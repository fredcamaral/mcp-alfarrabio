/**
 * Memory API React Hooks
 * 
 * Provides React hooks for common memory API operations with state management,
 * error handling, loading states, and automatic data synchronization.
 */

import { useState, useEffect, useCallback } from 'react'
import { 
  memoryAPI, 
  type ConversationChunk, 
  type SearchRequest, 
  type SearchResponse,
  type BackupMetadata,
  type APIError 
} from '@/lib/api-client'

// Generic API hook type
interface APIHookState<T> {
  data: T | null
  isLoading: boolean
  error: string | null
  refetch: () => Promise<void>
}

// Search memories hook
export function useSearchMemories(initialRequest?: SearchRequest) {
  const [state, setState] = useState<APIHookState<SearchResponse>>({
    data: null,
    isLoading: false,
    error: null,
    refetch: async () => {}
  })

  const search = useCallback(async (request: SearchRequest) => {
    setState(prev => ({ ...prev, isLoading: true, error: null }))
    
    try {
      const response = await memoryAPI.searchMemories(request)
      setState(prev => ({ 
        ...prev, 
        data: response, 
        isLoading: false 
      }))
      return response
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Search failed'
      setState(prev => ({ 
        ...prev, 
        error: errorMessage, 
        isLoading: false 
      }))
      throw error
    }
  }, [])

  const refetch = useCallback(async () => {
    if (initialRequest) {
      await search(initialRequest)
    }
  }, [search, initialRequest])

  useEffect(() => {
    setState(prev => ({ ...prev, refetch }))
  }, [refetch])

  useEffect(() => {
    if (initialRequest) {
      search(initialRequest)
    }
  }, [search, initialRequest])

  return {
    ...state,
    search
  }
}

// Store memory chunk hook
export function useStoreMemory() {
  const [isStoring, setIsStoring] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const store = useCallback(async (chunk: Partial<ConversationChunk>) => {
    setIsStoring(true)
    setError(null)
    
    try {
      const result = await memoryAPI.storeChunk(chunk)
      setIsStoring(false)
      return result
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to store memory'
      setError(errorMessage)
      setIsStoring(false)
      throw error
    }
  }, [])

  return {
    store,
    isStoring,
    error,
    clearError: () => setError(null)
  }
}

// Memory context hook
export function useMemoryContext(repository: string) {
  const [state, setState] = useState<APIHookState<any>>({
    data: null,
    isLoading: false,
    error: null,
    refetch: async () => {}
  })

  const fetchContext = useCallback(async () => {
    setState(prev => ({ ...prev, isLoading: true, error: null }))
    
    try {
      const context = await memoryAPI.getMemoryContext(repository)
      setState(prev => ({ 
        ...prev, 
        data: context, 
        isLoading: false 
      }))
      return context
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to fetch context'
      setState(prev => ({ 
        ...prev, 
        error: errorMessage, 
        isLoading: false 
      }))
      throw error
    }
  }, [repository])

  const refetch = useCallback(async () => {
    await fetchContext()
  }, [fetchContext])

  useEffect(() => {
    setState(prev => ({ ...prev, refetch }))
  }, [refetch])

  useEffect(() => {
    if (repository) {
      fetchContext()
    }
  }, [fetchContext, repository])

  return state
}

// Similar memories hook
export function useSimilarMemories(problem: string, repository: string) {
  const [state, setState] = useState<APIHookState<ConversationChunk[]>>({
    data: null,
    isLoading: false,
    error: null,
    refetch: async () => {}
  })

  const findSimilar = useCallback(async (newProblem?: string, newRepository?: string) => {
    const searchProblem = newProblem || problem
    const searchRepository = newRepository || repository
    
    if (!searchProblem || !searchRepository) return
    
    setState(prev => ({ ...prev, isLoading: true, error: null }))
    
    try {
      const chunks = await memoryAPI.findSimilarMemories(searchProblem, searchRepository)
      setState(prev => ({ 
        ...prev, 
        data: chunks, 
        isLoading: false 
      }))
      return chunks
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to find similar memories'
      setState(prev => ({ 
        ...prev, 
        error: errorMessage, 
        isLoading: false 
      }))
      throw error
    }
  }, [problem, repository])

  const refetch = useCallback(async () => {
    await findSimilar()
  }, [findSimilar])

  useEffect(() => {
    setState(prev => ({ ...prev, refetch }))
  }, [refetch])

  useEffect(() => {
    if (problem && repository) {
      findSimilar()
    }
  }, [findSimilar, problem, repository])

  return {
    ...state,
    findSimilar
  }
}

// Backup management hook
export function useBackups() {
  const [state, setState] = useState<APIHookState<BackupMetadata[]>>({
    data: null,
    isLoading: false,
    error: null,
    refetch: async () => {}
  })

  const [isCreating, setIsCreating] = useState(false)
  const [isRestoring, setIsRestoring] = useState(false)
  const [isCleaning, setIsCleaning] = useState(false)

  const fetchBackups = useCallback(async () => {
    setState(prev => ({ ...prev, isLoading: true, error: null }))
    
    try {
      const backups = await memoryAPI.listBackups()
      setState(prev => ({ 
        ...prev, 
        data: backups, 
        isLoading: false 
      }))
      return backups
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to fetch backups'
      setState(prev => ({ 
        ...prev, 
        error: errorMessage, 
        isLoading: false 
      }))
      throw error
    }
  }, [])

  const createBackup = useCallback(async (name?: string, repository?: string) => {
    setIsCreating(true)
    
    try {
      const backup = await memoryAPI.createBackup(name, repository)
      setIsCreating(false)
      await fetchBackups() // Refresh list
      return backup
    } catch (error) {
      setIsCreating(false)
      throw error
    }
  }, [fetchBackups])

  const restoreBackup = useCallback(async (backupFile: string, overwrite: boolean = false) => {
    setIsRestoring(true)
    
    try {
      const result = await memoryAPI.restoreBackup(backupFile, overwrite)
      setIsRestoring(false)
      return result
    } catch (error) {
      setIsRestoring(false)
      throw error
    }
  }, [])

  const cleanupBackups = useCallback(async () => {
    setIsCleaning(true)
    
    try {
      const result = await memoryAPI.cleanupBackups()
      setIsCleaning(false)
      await fetchBackups() // Refresh list
      return result
    } catch (error) {
      setIsCleaning(false)
      throw error
    }
  }, [fetchBackups])

  const refetch = useCallback(async () => {
    await fetchBackups()
  }, [fetchBackups])

  useEffect(() => {
    setState(prev => ({ ...prev, refetch }))
  }, [refetch])

  useEffect(() => {
    fetchBackups()
  }, [fetchBackups])

  return {
    ...state,
    createBackup,
    restoreBackup,
    cleanupBackups,
    isCreating,
    isRestoring,
    isCleaning
  }
}

// Connection status hook
export function useAPIConnection() {
  const [isConnected, setIsConnected] = useState<boolean | null>(null)
  const [isChecking, setIsChecking] = useState(false)

  const checkConnection = useCallback(async () => {
    setIsChecking(true)
    
    try {
      const connected = await memoryAPI.testConnection()
      setIsConnected(connected)
      setIsChecking(false)
      return connected
    } catch (error) {
      setIsConnected(false)
      setIsChecking(false)
      return false
    }
  }, [])

  useEffect(() => {
    checkConnection()
    
    // Check connection every 30 seconds
    const interval = setInterval(checkConnection, 30000)
    return () => clearInterval(interval)
  }, [checkConnection])

  return {
    isConnected,
    isChecking,
    checkConnection
  }
}

// Memory relationships hook
export function useMemoryRelationships(chunkId: string, repository: string) {
  const [state, setState] = useState<APIHookState<any>>({
    data: null,
    isLoading: false,
    error: null,
    refetch: async () => {}
  })

  const fetchRelationships = useCallback(async () => {
    if (!chunkId || !repository) return
    
    setState(prev => ({ ...prev, isLoading: true, error: null }))
    
    try {
      const relationships = await memoryAPI.getMemoryRelationships(chunkId, repository)
      setState(prev => ({ 
        ...prev, 
        data: relationships, 
        isLoading: false 
      }))
      return relationships
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to fetch relationships'
      setState(prev => ({ 
        ...prev, 
        error: errorMessage, 
        isLoading: false 
      }))
      throw error
    }
  }, [chunkId, repository])

  const refetch = useCallback(async () => {
    await fetchRelationships()
  }, [fetchRelationships])

  useEffect(() => {
    setState(prev => ({ ...prev, refetch }))
  }, [refetch])

  useEffect(() => {
    if (chunkId && repository) {
      fetchRelationships()
    }
  }, [fetchRelationships, chunkId, repository])

  return state
}

// Export/Import hook
export function useMemoryTransfer() {
  const [isExporting, setIsExporting] = useState(false)
  const [isImporting, setIsImporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const exportMemories = useCallback(async (repository: string, format: string = 'json') => {
    setIsExporting(true)
    setError(null)
    
    try {
      const result = await memoryAPI.exportMemories(repository, format)
      setIsExporting(false)
      return result
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Export failed'
      setError(errorMessage)
      setIsExporting(false)
      throw error
    }
  }, [])

  const importMemories = useCallback(async (data: string, repository: string, sessionId: string) => {
    setIsImporting(true)
    setError(null)
    
    try {
      const result = await memoryAPI.importMemories(data, repository, sessionId)
      setIsImporting(false)
      return result
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Import failed'
      setError(errorMessage)
      setIsImporting(false)
      throw error
    }
  }, [])

  return {
    exportMemories,
    importMemories,
    isExporting,
    isImporting,
    error,
    clearError: () => setError(null)
  }
}