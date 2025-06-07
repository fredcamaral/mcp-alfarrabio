/**
 * Client-Side Caching System
 * 
 * Provides efficient caching for API responses, search results, and user data
 * with TTL support, cache invalidation, and memory management.
 */

import { handleError } from './error-handling'
import type { SearchResults, ConversationChunk, MemoryRelationship } from '@/types/memory'

export interface CacheOptions {
  ttl?: number // Time to live in milliseconds
  maxSize?: number // Maximum number of cache entries
  version?: string // Cache version for invalidation
  namespace?: string // Cache namespace for organization
}

export interface CacheEntry<T> {
  data: T
  timestamp: number
  ttl: number
  version: string
  namespace: string
  accessCount: number
  lastAccessed: number
}

export interface CacheStats {
  totalEntries: number
  totalSize: number
  hitRate: number
  missRate: number
  oldestEntry?: Date
  newestEntry?: Date
  namespaces: string[]
}

/**
 * Generic cache implementation with TTL and LRU eviction
 */
export class Cache<T = unknown> {
  protected cache = new Map<string, CacheEntry<T>>()
  private hits = 0
  private misses = 0
  private readonly defaultTTL: number
  private readonly maxSize: number
  private readonly defaultVersion: string
  private readonly defaultNamespace: string

  constructor(options: CacheOptions = {}) {
    this.defaultTTL = options.ttl || 5 * 60 * 1000 // 5 minutes default
    this.maxSize = options.maxSize || 1000
    this.defaultVersion = options.version || '1.0'
    this.defaultNamespace = options.namespace || 'default'
  }

  /**
   * Store item in cache
   */
  set(
    key: string, 
    data: T, 
    options: Partial<CacheOptions> = {}
  ): void {
    try {
      const now = Date.now()
      const ttl = options.ttl || this.defaultTTL
      const version = options.version || this.defaultVersion
      const namespace = options.namespace || this.defaultNamespace

      // Check if we need to evict entries
      if (this.cache.size >= this.maxSize) {
        this.evictLRU()
      }

      const entry: CacheEntry<T> = {
        data,
        timestamp: now,
        ttl,
        version,
        namespace,
        accessCount: 0,
        lastAccessed: now
      }

      this.cache.set(key, entry)
    } catch (error) {
      handleError(error, { source: 'cache_set', key })
    }
  }

  /**
   * Retrieve item from cache
   */
  get(key: string, options: Partial<CacheOptions> = {}): T | null {
    try {
      const entry = this.cache.get(key)
      
      if (!entry) {
        this.misses++
        return null
      }

      const now = Date.now()
      const version = options.version || this.defaultVersion

      // Check if entry is expired
      if (now - entry.timestamp > entry.ttl) {
        this.cache.delete(key)
        this.misses++
        return null
      }

      // Check version compatibility
      if (entry.version !== version) {
        this.cache.delete(key)
        this.misses++
        return null
      }

      // Update access statistics
      entry.accessCount++
      entry.lastAccessed = now
      this.hits++

      return entry.data
    } catch (error) {
      handleError(error, { source: 'cache_get', key })
      this.misses++
      return null
    }
  }

  /**
   * Check if key exists and is valid
   */
  has(key: string, options: Partial<CacheOptions> = {}): boolean {
    return this.get(key, options) !== null
  }

  /**
   * Delete specific key
   */
  delete(key: string): boolean {
    return this.cache.delete(key)
  }

  /**
   * Clear all cache entries
   */
  clear(): void {
    this.cache.clear()
    this.hits = 0
    this.misses = 0
  }

  /**
   * Clear entries by namespace
   */
  clearNamespace(namespace: string): number {
    let cleared = 0
    for (const [key, entry] of this.cache.entries()) {
      if (entry.namespace === namespace) {
        this.cache.delete(key)
        cleared++
      }
    }
    return cleared
  }

  /**
   * Clear entries by version (invalidate old versions)
   */
  clearVersion(version: string): number {
    let cleared = 0
    for (const [key, entry] of this.cache.entries()) {
      if (entry.version !== version) {
        this.cache.delete(key)
        cleared++
      }
    }
    return cleared
  }

  /**
   * Evict expired entries
   */
  evictExpired(): number {
    const now = Date.now()
    let evicted = 0

    for (const [key, entry] of this.cache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.cache.delete(key)
        evicted++
      }
    }

    return evicted
  }

  /**
   * Evict least recently used entry
   */
  private evictLRU(): void {
    let oldestKey: string | null = null
    let oldestTime = Date.now()

    for (const [key, entry] of this.cache.entries()) {
      if (entry.lastAccessed < oldestTime) {
        oldestTime = entry.lastAccessed
        oldestKey = key
      }
    }

    if (oldestKey) {
      this.cache.delete(oldestKey)
    }
  }

  /**
   * Get cache statistics
   */
  getStats(): CacheStats {
    const entries = Array.from(this.cache.values())
    const total = this.hits + this.misses
    
    const timestamps = entries.map(e => e.timestamp)
    const namespaces = [...new Set(entries.map(e => e.namespace))]

    return {
      totalEntries: this.cache.size,
      totalSize: this.cache.size,
      hitRate: total > 0 ? this.hits / total : 0,
      missRate: total > 0 ? this.misses / total : 0,
      oldestEntry: timestamps.length > 0 ? new Date(Math.min(...timestamps)) : undefined,
      newestEntry: timestamps.length > 0 ? new Date(Math.max(...timestamps)) : undefined,
      namespaces
    }
  }

  /**
   * Get all keys in cache
   */
  keys(): string[] {
    return Array.from(this.cache.keys())
  }

  /**
   * Get cache size
   */
  size(): number {
    return this.cache.size
  }
}

/**
 * Search result cache with query normalization
 */
export class SearchCache extends Cache<SearchResults> {
  constructor() {
    super({
      ttl: 10 * 60 * 1000, // 10 minutes for search results
      maxSize: 500,
      namespace: 'search'
    })
  }

  /**
   * Generate cache key from search parameters
   */
  private generateSearchKey(query: string, filters: Record<string, unknown> = {}): string {
    // Normalize query
    const normalizedQuery = query.toLowerCase().trim()
    
    // Sort filters for consistent key generation
    const sortedFilters = Object.keys(filters)
      .sort()
      .reduce((acc, key) => {
        acc[key] = filters[key]
        return acc
      }, {} as Record<string, unknown>)

    return `search:${normalizedQuery}:${JSON.stringify(sortedFilters)}`
  }

  /**
   * Cache search results
   */
  cacheSearch(query: string, filters: Record<string, unknown>, results: SearchResults): void {
    const key = this.generateSearchKey(query, filters)
    this.set(key, results)
  }

  /**
   * Get cached search results
   */
  getSearch(query: string, filters: Record<string, unknown> = {}): SearchResults | null {
    const key = this.generateSearchKey(query, filters)
    return this.get(key)
  }

  /**
   * Invalidate search results containing specific terms
   */
  invalidateSearchTerm(term: string): number {
    let invalidated = 0
    const lowerTerm = term.toLowerCase()

    for (const key of this.keys()) {
      if (key.includes(lowerTerm)) {
        this.delete(key)
        invalidated++
      }
    }

    return invalidated
  }
}

/**
 * API response cache with endpoint-based organization
 */
export class APICache extends Cache<unknown> {
  constructor() {
    super({
      ttl: 5 * 60 * 1000, // 5 minutes for API responses
      maxSize: 1000,
      namespace: 'api'
    })
  }

  /**
   * Generate cache key for API endpoint
   */
  private generateAPIKey(endpoint: string, params: Record<string, unknown> = {}): string {
    const sortedParams = Object.keys(params)
      .sort()
      .reduce((acc, key) => {
        acc[key] = params[key]
        return acc
      }, {} as Record<string, unknown>)

    return `api:${endpoint}:${JSON.stringify(sortedParams)}`
  }

  /**
   * Cache API response
   */
  cacheResponse(endpoint: string, params: Record<string, unknown>, response: unknown): void {
    const key = this.generateAPIKey(endpoint, params)
    this.set(key, response)
  }

  /**
   * Get cached API response
   */
  getResponse(endpoint: string, params: Record<string, unknown> = {}): unknown | null {
    const key = this.generateAPIKey(endpoint, params)
    return this.get(key)
  }

  /**
   * Invalidate cache for specific endpoint
   */
  invalidateEndpoint(endpoint: string): number {
    let invalidated = 0

    for (const key of this.keys()) {
      if (key.startsWith(`api:${endpoint}:`)) {
        this.delete(key)
        invalidated++
      }
    }

    return invalidated
  }
}

/**
 * Memory chunk cache for frequently accessed memories
 */
export class MemoryCache extends Cache<ConversationChunk | MemoryRelationship[]> {
  constructor() {
    super({
      ttl: 15 * 60 * 1000, // 15 minutes for memory chunks
      maxSize: 2000,
      namespace: 'memory'
    })
  }

  /**
   * Cache memory chunk
   */
  cacheChunk(chunkId: string, chunk: ConversationChunk): void {
    this.set(`chunk:${chunkId}`, chunk)
  }

  /**
   * Get cached memory chunk
   */
  getChunk(chunkId: string): ConversationChunk | null {
    const result = this.get(`chunk:${chunkId}`)
    return result && typeof result === 'object' && 'id' in result ? result as ConversationChunk : null
  }

  /**
   * Cache memory relationships
   */
  cacheRelationships(chunkId: string, relationships: MemoryRelationship[]): void {
    this.set(`relationships:${chunkId}`, relationships)
  }

  /**
   * Get cached relationships
   */
  getRelationships(chunkId: string): MemoryRelationship[] | null {
    const result = this.get(`relationships:${chunkId}`)
    return result && Array.isArray(result) ? result as MemoryRelationship[] : null
  }

  /**
   * Invalidate memory by repository
   */
  invalidateRepository(repository: string): number {
    let invalidated = 0

    for (const [key, entry] of this.cache.entries()) {
      // Check if this is a ConversationChunk with repository metadata
      if (
        entry.data && 
        typeof entry.data === 'object' && 
        'metadata' in entry.data &&
        typeof entry.data.metadata === 'object' &&
        entry.data.metadata &&
        'repository' in entry.data.metadata &&
        entry.data.metadata.repository === repository
      ) {
        this.cache.delete(key)
        invalidated++
      }
    }

    return invalidated
  }
}

// Global cache instances
export const searchCache = new SearchCache()
export const apiCache = new APICache()
export const memoryCache = new MemoryCache()

/**
 * Cache manager for coordinating all caches
 */
export class CacheManager {
  private caches: Map<string, Cache> = new Map()

  constructor() {
    this.caches.set('search', searchCache)
    this.caches.set('api', apiCache)
    this.caches.set('memory', memoryCache)
  }

  /**
   * Add a new cache instance
   */
  addCache(name: string, cache: Cache): void {
    this.caches.set(name, cache)
  }

  /**
   * Get cache by name
   */
  getCache(name: string): Cache | undefined {
    return this.caches.get(name)
  }

  /**
   * Clear all caches
   */
  clearAll(): void {
    for (const cache of this.caches.values()) {
      cache.clear()
    }
  }

  /**
   * Evict expired entries from all caches
   */
  evictExpiredAll(): number {
    let totalEvicted = 0
    for (const cache of this.caches.values()) {
      totalEvicted += cache.evictExpired()
    }
    return totalEvicted
  }

  /**
   * Get combined statistics for all caches
   */
  getAllStats(): Record<string, CacheStats> {
    const stats: Record<string, CacheStats> = {}
    
    for (const [name, cache] of this.caches.entries()) {
      stats[name] = cache.getStats()
    }

    return stats
  }

  /**
   * Invalidate cache when data changes
   */
  invalidateOnDataChange(type: 'memory' | 'search' | 'api', identifier?: string): void {
    switch (type) {
      case 'memory':
        memoryCache.clearNamespace('memory')
        if (identifier) {
          searchCache.invalidateSearchTerm(identifier)
        }
        break
      case 'search':
        searchCache.clear()
        break
      case 'api':
        if (identifier) {
          apiCache.invalidateEndpoint(identifier)
        } else {
          apiCache.clear()
        }
        break
    }
  }
}

// Global cache manager instance
export const cacheManager = new CacheManager()

// Setup automatic cleanup
if (typeof window !== 'undefined') {
  // Clean up expired entries every 5 minutes
  setInterval(() => {
    cacheManager.evictExpiredAll()
  }, 5 * 60 * 1000)
}