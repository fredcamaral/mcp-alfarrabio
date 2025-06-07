/**
 * Multi-Repository Intelligence Hooks
 * 
 * Provides hooks for cross-repository analysis,
 * pattern detection, and knowledge discovery
 */

import { useState, useEffect, useCallback } from 'react'
import { logger } from '@/lib/logger'

export interface Repository {
  id: string
  name: string
  url: string
  status: 'active' | 'inactive' | 'analyzing'
  memoryCount: number
  lastUpdated: string
  language?: string
  size?: number
  metadata?: Record<string, any>
}

export interface CrossRepoPattern {
  id: string
  name: string
  description: string
  repositories: string[]
  frequency: number
  impact: 'high' | 'medium' | 'low'
  type: 'architecture' | 'code' | 'dependency' | 'practice'
  confidence: number
  examples: PatternExample[]
}

export interface PatternExample {
  repository: string
  file: string
  line: number
  snippet: string
  context?: string
}

export interface KnowledgeLink {
  id: string
  sourceRepo: string
  targetRepo: string
  linkType: 'dependency' | 'pattern' | 'concept' | 'reference'
  strength: number
  description: string
  bidirectional: boolean
}

export interface ImpactAnalysis {
  changeRepo: string
  changeType: 'breaking' | 'feature' | 'fix' | 'refactor'
  affectedRepos: AffectedRepo[]
  totalImpact: 'high' | 'medium' | 'low'
  recommendations: string[]
  confidence: number
}

export interface AffectedRepo {
  repository: string
  impact: 'direct' | 'indirect'
  severity: 'high' | 'medium' | 'low'
  areas: string[]
  confidence: number
}

interface UseMultiRepoOptions {
  autoRefresh?: boolean
  refreshInterval?: number
}

/**
 * Hook to manage multi-repository data and analysis
 */
export function useMultiRepo(options: UseMultiRepoOptions = {}) {
  const { autoRefresh = false, refreshInterval = 60000 } = options
  
  const [repositories, setRepositories] = useState<Repository[]>([])
  const [patterns, setPatterns] = useState<CrossRepoPattern[]>([])
  const [knowledgeLinks, setKnowledgeLinks] = useState<KnowledgeLink[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchRepositories = useCallback(async () => {
    try {
      const response = await fetch('/api/multi-repo/repositories')
      if (!response.ok) throw new Error('Failed to fetch repositories')
      
      const data = await response.json()
      setRepositories(data.repositories || [])
    } catch (err) {
      logger.error('Failed to fetch repositories:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }, [])

  const fetchPatterns = useCallback(async () => {
    try {
      const response = await fetch('/api/multi-repo/patterns')
      if (!response.ok) throw new Error('Failed to fetch patterns')
      
      const data = await response.json()
      setPatterns(data.patterns || [])
    } catch (err) {
      logger.error('Failed to fetch patterns:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }, [])

  const fetchKnowledgeLinks = useCallback(async () => {
    try {
      const response = await fetch('/api/multi-repo/knowledge-links')
      if (!response.ok) throw new Error('Failed to fetch knowledge links')
      
      const data = await response.json()
      setKnowledgeLinks(data.links || [])
    } catch (err) {
      logger.error('Failed to fetch knowledge links:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    }
  }, [])

  const analyzeRepositories = useCallback(async (repoIds?: string[]) => {
    setIsLoading(true)
    setError(null)
    
    try {
      const response = await fetch('/api/multi-repo/analyze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repositoryIds: repoIds })
      })
      
      if (!response.ok) throw new Error('Failed to analyze repositories')
      
      const data = await response.json()
      
      // Refresh all data after analysis
      await Promise.all([
        fetchRepositories(),
        fetchPatterns(),
        fetchKnowledgeLinks()
      ])
      
      return data
    } catch (err) {
      logger.error('Failed to analyze repositories:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [fetchRepositories, fetchPatterns, fetchKnowledgeLinks])

  const analyzeImpact = useCallback(async (
    changeRepo: string,
    changeDescription: string,
    changeType: ImpactAnalysis['changeType']
  ): Promise<ImpactAnalysis | null> => {
    try {
      const response = await fetch('/api/multi-repo/impact-analysis', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repository: changeRepo,
          description: changeDescription,
          type: changeType
        })
      })
      
      if (!response.ok) throw new Error('Failed to analyze impact')
      
      const data = await response.json()
      return data.analysis
    } catch (err) {
      logger.error('Failed to analyze impact:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
      return null
    }
  }, [])

  const searchCrossRepoPatterns = useCallback(async (query: string) => {
    try {
      const response = await fetch(`/api/multi-repo/search-patterns?q=${encodeURIComponent(query)}`)
      if (!response.ok) throw new Error('Failed to search patterns')
      
      const data = await response.json()
      return data.patterns || []
    } catch (err) {
      logger.error('Failed to search patterns:', err)
      return []
    }
  }, [])

  // Initial fetch
  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true)
      await Promise.all([
        fetchRepositories(),
        fetchPatterns(),
        fetchKnowledgeLinks()
      ])
      setIsLoading(false)
    }
    
    loadData()
  }, [fetchRepositories, fetchPatterns, fetchKnowledgeLinks])

  // Auto-refresh
  useEffect(() => {
    if (!autoRefresh) return
    
    const interval = setInterval(() => {
      fetchRepositories()
      fetchPatterns()
      fetchKnowledgeLinks()
    }, refreshInterval)
    
    return () => clearInterval(interval)
  }, [autoRefresh, refreshInterval, fetchRepositories, fetchPatterns, fetchKnowledgeLinks])

  return {
    repositories,
    patterns,
    knowledgeLinks,
    isLoading,
    error,
    analyzeRepositories,
    analyzeImpact,
    searchCrossRepoPatterns,
    refresh: () => Promise.all([
      fetchRepositories(),
      fetchPatterns(),
      fetchKnowledgeLinks()
    ])
  }
}

/**
 * Hook to manage repository connections and dependencies
 */
export function useRepositoryConnections(repositoryId: string) {
  const [connections, setConnections] = useState<KnowledgeLink[]>([])
  const [dependencies, setDependencies] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const fetchConnections = async () => {
      setIsLoading(true)
      
      try {
        const response = await fetch(`/api/multi-repo/repositories/${repositoryId}/connections`)
        if (!response.ok) throw new Error('Failed to fetch connections')
        
        const data = await response.json()
        setConnections(data.connections || [])
        setDependencies(data.dependencies || [])
      } catch (err) {
        logger.error('Failed to fetch repository connections:', err)
      } finally {
        setIsLoading(false)
      }
    }
    
    if (repositoryId) {
      fetchConnections()
    }
  }, [repositoryId])

  return { connections, dependencies, isLoading }
}

/**
 * Hook to get shared learning insights across repositories
 */
export function useSharedLearning() {
  const [insights, setInsights] = useState<{
    practices: string[]
    technologies: Record<string, number>
    architecturePatterns: string[]
    commonIssues: string[]
  }>({
    practices: [],
    technologies: {},
    architecturePatterns: [],
    commonIssues: []
  })
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const fetchInsights = async () => {
      setIsLoading(true)
      
      try {
        const response = await fetch('/api/multi-repo/shared-learning')
        if (!response.ok) throw new Error('Failed to fetch shared learning')
        
        const data = await response.json()
        setInsights(data.insights)
      } catch (err) {
        logger.error('Failed to fetch shared learning insights:', err)
      } finally {
        setIsLoading(false)
      }
    }
    
    fetchInsights()
  }, [])

  return { insights, isLoading }
}