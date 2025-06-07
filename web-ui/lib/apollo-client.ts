import { ApolloClient, InMemoryCache, createHttpLink, from, split, ApolloLink, Observable, gql, FetchResult } from '@apollo/client'
import { onError } from '@apollo/client/link/error'
import { setContext } from '@apollo/client/link/context'
import { RetryLink } from '@apollo/client/link/retry'
import { GraphQLWsLink } from '@apollo/client/link/subscriptions'
import { createClient } from 'graphql-ws'
import { getMainDefinition } from '@apollo/client/utilities'
import { getGraphQLUrl, getWebSocketUrl } from './utils'
import { logger } from './logger'
import { config } from './env-validation'
import { trackGraphQLOperation } from './monitoring/performance'
import { CSRFManager } from './csrf-client'
import { markPerformance } from '@/lib/performance/web-vitals'

// Deduplication link to prevent duplicate queries
const deduplicationLink = new ApolloLink((operation, forward) => {
  const activeRequests = new Map<string, Observable<FetchResult>>()
  
  return new Observable(observer => {
    const key = `${operation.operationName}-${JSON.stringify(operation.variables)}`
    
    if (activeRequests.has(key)) {
      // Return existing request
      return activeRequests.get(key)!.subscribe(observer)
    }
    
    const request = forward(operation)
    activeRequests.set(key, request)
    
    const subscription = request.subscribe({
      next: (result) => {
        observer.next(result)
      },
      error: (error) => {
        activeRequests.delete(key)
        observer.error(error)
      },
      complete: () => {
        activeRequests.delete(key)
        observer.complete()
      },
    })
    
    return () => {
      activeRequests.delete(key)
      subscription.unsubscribe()
    }
  })
})

// Create HTTP link to GraphQL endpoint with performance optimizations
const httpLink = createHttpLink({
  uri: getGraphQLUrl(),
  credentials: 'include',
  fetch: (uri, options) => {
    const startTime = performance.now()
    
    return fetch(uri, {
      ...options,
      headers: {
        ...options?.headers,
        // Enable compression
        'Accept-Encoding': 'gzip, deflate, br',
        // Enable keep-alive for connection reuse
        'Connection': 'keep-alive',
      },
    }).then(response => {
      const duration = performance.now() - startTime
      markPerformance('graphql-http-request', { 
        duration, 
        status: response.status,
        operation: options?.body ? JSON.parse(options.body as string).operationName : 'unknown'
      })
      return response
    })
  },
})

// Error handling link
const errorLink = onError(({ graphQLErrors, networkError, operation }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      logger.error('GraphQL error', new Error(message), {
        component: 'ApolloClient',
        locations: locations ? JSON.stringify(locations) : undefined,
        path: path ? JSON.stringify(path) : undefined
      })
    })
  }

  if (networkError) {
    // Only log network errors in development if not a connection refused error
    if (process.env.NODE_ENV === 'development') {
      if (!networkError.message?.includes('Failed to fetch') && !networkError.message?.includes('fetch')) {
        logger.error('GraphQL network error', networkError, {
          component: 'ApolloClient',
          operation: operation.operationName
        })
      } else {
        logger.debug('🔧 Backend not available - this is normal in development mode', {
          component: 'ApolloClient'
        })
      }
    } else {
      logger.error('GraphQL network error', networkError, {
        component: 'ApolloClient',
        operation: operation.operationName
      })
    }

    // Retry logic for network errors
    // if ('statusCode' in networkError && (networkError.statusCode === 503 || networkError.statusCode === 502)) {
    //   return forward(operation)
    // }
  }
})

// Enhanced performance tracking link
const performanceLink = new ApolloLink((operation, forward) => {
  const { operationName } = operation
  const operationType = operation.query.definitions[0]?.kind === 'OperationDefinition' 
    ? operation.query.definitions[0].operation 
    : 'unknown'
  
  const startTime = Date.now()
  markPerformance(`graphql-${operationName || 'unnamed'}-start`)
  
  const endTracking = trackGraphQLOperation(operationType, operationName || 'unnamed')
  
  return new Observable(observer => {
    const subscription = forward(operation).subscribe({
      next: (result) => {
        const duration = Date.now() - startTime
        const errors = result.errors?.length || 0
        
        markPerformance(`graphql-${operationName || 'unnamed'}-complete`, { 
          duration,
          cacheHit: !!result.data,
          errors
        })
        
        endTracking(errors === 0, errors)
        
        if (process.env.NODE_ENV === 'development') {
          console.log(`🚀 GraphQL ${operationName || 'unnamed'}: ${duration}ms`, {
            operation: operationName,
            variables: operation.variables,
            cacheHit: !!result.data,
            errors
          })
        }
        
        observer.next(result)
      },
      error: (error) => {
        const duration = Date.now() - startTime
        markPerformance(`graphql-${operationName || 'unnamed'}-error`, { 
          duration, 
          error: error.message 
        })
        endTracking(false, 1)
        observer.error(error)
      },
      complete: () => {
        observer.complete()
      },
    })
    
    return () => subscription.unsubscribe()
  })
})

// Retry link for resilient queries
const retryLink = new RetryLink({
  delay: {
    initial: 300,
    max: Infinity,
    jitter: true,
  },
  attempts: {
    max: 3,
    retryIf: (error) => {
      // Retry on network errors but not on GraphQL errors
      return !!error && !error.message.includes('GraphQL') && !error.message.includes('Unauthorized')
    },
  },
})

// Auth link with CSRF protection
const authLink = setContext((_, { headers }) => {
  // Get authentication token if available
  // NOTE: localStorage is vulnerable to XSS attacks. In production,
  // consider using httpOnly cookies for authentication tokens.
  // Currently disabled as authentication is not required for local usage.
  const token = typeof window !== 'undefined'
    ? localStorage.getItem('mcp_auth_token')
    : null

  // Get CSRF headers
  const csrfHeaders = CSRFManager.getHeaders()

  return {
    headers: {
      ...headers,
      ...csrfHeaders, // Include CSRF headers
      ...(token && { authorization: `Bearer ${token}` }),
      'Content-Type': 'application/json',
    }
  }
})

// Create WebSocket link for subscriptions (only if enabled)
let wsLink: GraphQLWsLink | null = null

if (config.features.websocket && config.features.graphqlSubscriptions) {
  try {
    wsLink = new GraphQLWsLink(
      createClient({
        url: getWebSocketUrl().replace('ws://', 'ws://').replace('wss://', 'wss://') + '/graphql',
        connectionParams: () => {
          // NOTE: See security note above about localStorage usage
          const token = typeof window !== 'undefined'
            ? localStorage.getItem('mcp_auth_token')
            : null
          
          // Include CSRF headers in WebSocket connection
          const csrfHeaders = CSRFManager.getHeaders()
          
          return {
            ...csrfHeaders,
            ...(token && { authorization: `Bearer ${token}` })
          }
        },
        on: {
          connected: () => {
            logger.info('GraphQL WebSocket connected', {
              component: 'ApolloClient'
            })
          },
          closed: () => {
            logger.info('GraphQL WebSocket closed', {
              component: 'ApolloClient'
            })
          },
          error: (error) => {
            logger.error('GraphQL WebSocket error', error as Error, {
              component: 'ApolloClient'
            })
          }
        },
        shouldRetry: () => true,
        retryAttempts: 5,
        retryWait: async (retries) => {
          // Exponential backoff
          await new Promise(resolve => setTimeout(resolve, Math.min(1000 * Math.pow(2, retries), 30000)))
          return
        }
      })
    )
  } catch (error) {
    logger.warn('Failed to create WebSocket link', {
      component: 'ApolloClient',
      error: error instanceof Error ? error.message : 'Unknown error'
    })
  }
}

// Create split link to route subscriptions through WebSocket
const splitLink = wsLink
  ? split(
      ({ query }) => {
        const definition = getMainDefinition(query)
        return (
          definition.kind === 'OperationDefinition' &&
          definition.operation === 'subscription'
        )
      },
      wsLink,
      from([deduplicationLink, performanceLink, retryLink, errorLink, authLink, httpLink])
    )
  : from([deduplicationLink, performanceLink, retryLink, errorLink, authLink, httpLink])

// Enhanced cache configuration with performance optimizations
const cache = new InMemoryCache({
  typePolicies: {
    ConversationChunk: {
      keyFields: ['id'],
      fields: {
        related_chunks: {
          merge: (existing = [], incoming) => {
            // Merge related chunks by ID to avoid duplicates
            const existingIds = new Set(existing.map((chunk: { id: string }) => chunk.id))
            const newChunks = incoming.filter((chunk: { id: string }) => !existingIds.has(chunk.id))
            return [...existing, ...newChunks]
          }
        },
        content: {
          merge: true
        },
        metadata: {
          merge: true
        }
      }
    },
    SearchResults: {
      keyFields: ['query', 'filters'],
      fields: {
        results: {
          keyArgs: ['query', 'filters'],
          merge(existing = [], incoming, { args }) {
            // Handle pagination by appending new results
            if (args?.offset && args.offset > 0) {
              return [...existing, ...incoming]
            }
            return incoming
          }
        }
      }
    },
    Repository: {
      keyFields: ['name'],
      fields: {
        patterns: {
          merge(existing = [], incoming) {
            // Merge patterns by ID to avoid duplicates
            const existingIds = new Set(existing.map((p: { id: string }) => p.id))
            const newPatterns = incoming.filter((p: { id: string }) => !existingIds.has(p.id))
            return [...existing, ...newPatterns]
          }
        }
      }
    },
    Pattern: {
      keyFields: ['id'],
      fields: {
        occurrences: {
          merge: (_, incoming) => incoming
        }
      }
    },
    Query: {
      fields: {
        memories: {
          keyArgs: ['filters', 'sortBy'],
          merge(existing = [], incoming, { args }) {
            if (args?.offset) {
              return [...existing, ...incoming]
            }
            return incoming
          }
        }
      }
    }
  },
  // Custom data ID function for better cache normalization
  dataIdFromObject: (object) => {
    if (object.__typename && object.id) {
      return `${object.__typename}:${object.id}`
    }
    return undefined
  },
  // Enable cache garbage collection
  possibleTypes: {
    // Define possible types for union/interface types if needed
  }
})

// Create Apollo Client with enhanced performance configuration
export const apolloClient = new ApolloClient({
  link: splitLink,
  connectToDevTools: process.env.NODE_ENV === 'development',
  cache,
  // Enable query deduplication
  queryDeduplication: true,
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'all',
      notifyOnNetworkStatusChange: true,
      fetchPolicy: 'cache-and-network',
      nextFetchPolicy: 'cache-first',
    },
    query: {
      errorPolicy: 'all',
      fetchPolicy: 'cache-first',
    },
    mutate: {
      errorPolicy: 'all',
      fetchPolicy: 'no-cache',
    }
  }
})

// Cache garbage collection for memory optimization
if (typeof window !== 'undefined') {
  // Run cache cleanup every 5 minutes
  setInterval(() => {
    apolloClient.cache.gc()
    
    if (process.env.NODE_ENV === 'development') {
      const extract = apolloClient.cache.extract()
      const size = JSON.stringify(extract).length
      console.log(`📦 Apollo Cache Size: ${(size / 1024).toFixed(2)} KB`)
    }
  }, 5 * 60 * 1000)
}

// Helper function to handle GraphQL errors
interface GraphQLErrorType {
  networkError?: {
    statusCode?: number
    message?: string
  }
  graphQLErrors?: Array<{
    message: string
  }>
  message?: string
}

export function handleGraphQLError(error: GraphQLErrorType): string {
  if (error.networkError) {
    if (error.networkError.statusCode === 503) {
      return 'Service temporarily unavailable. Please try again.'
    }
    if (error.networkError.statusCode === 500) {
      return 'Server error. Please contact support if this persists.'
    }
    return 'Network error. Please check your connection.'
  }

  if (error.graphQLErrors && error.graphQLErrors.length > 0) {
    return error.graphQLErrors[0].message
  }

  return error.message || 'An unexpected error occurred'
}

// Enhanced cache management utilities
export const cacheUtils = {
  // Clear all cached data
  clearCache: () => {
    apolloClient.cache.reset()
  },

  // Remove specific memory from cache
  evictMemory: (id: string) => {
    apolloClient.cache.evict({
      id: apolloClient.cache.identify({ __typename: 'ConversationChunk', id })
    })
    apolloClient.cache.gc()
  },

  // Update memory in cache
  updateMemory: (id: string, updates: Record<string, unknown>) => {
    apolloClient.cache.modify({
      id: apolloClient.cache.identify({ __typename: 'ConversationChunk', id }),
      fields: Object.fromEntries(
        Object.entries(updates).map(([key, value]) => [
          key,
          () => value
        ])
      )
    })
  },

  // Invalidate search results
  invalidateSearchResults: () => {
    apolloClient.cache.evict({ fieldName: 'searchMemories' })
    apolloClient.cache.gc()
  },

  // Get cache size for monitoring
  getCacheSize: () => {
    const extract = apolloClient.cache.extract()
    return JSON.stringify(extract).length
  },

  // Force garbage collection
  forceGC: () => {
    apolloClient.cache.gc()
  },

  // Prefetch critical data
  prefetchMemories: async (variables?: { limit?: number; offset?: number }) => {
    try {
      await apolloClient.query({
        query: MEMORIES_QUERY,
        variables,
        fetchPolicy: 'cache-first',
      })
    } catch (error) {
      console.warn('Failed to prefetch memories:', error)
    }
  },

  // Cache warming utilities
  warmCache: async () => {
    // Implement cache warming strategies here
    console.log('🔥 Warming Apollo cache...')
  },

  // Export cache state for debugging
  exportCacheState: () => {
    if (process.env.NODE_ENV === 'development') {
      const state = apolloClient.cache.extract()
      console.log('📊 Apollo Cache State:', state)
      return state
    }
    return null
  }
}

// Performance monitoring exports
export function logCacheSize() {
  if (process.env.NODE_ENV === 'development') {
    const size = cacheUtils.getCacheSize()
    console.log(`📦 Apollo Cache Size: ${(size / 1024).toFixed(2)} KB`)
  }
}

export function clearCache() {
  return apolloClient.clearStore()
}

export function resetCache() {
  return apolloClient.resetStore()
}

// Export cache instance for direct access
export { cache }

// Mock query for prefetching (replace with actual query)
const MEMORIES_QUERY = gql`
  query GetMemories($limit: Int, $offset: Int) {
    memories(limit: $limit, offset: $offset) {
      id
      content
      metadata
      createdAt
    }
  }
`