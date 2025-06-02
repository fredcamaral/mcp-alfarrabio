import { ApolloClient, InMemoryCache, createHttpLink, from } from '@apollo/client'
import { onError } from '@apollo/client/link/error'
import { setContext } from '@apollo/client/link/context'
import { getGraphQLUrl } from './utils'

// Create HTTP link to GraphQL endpoint
const httpLink = createHttpLink({
  uri: getGraphQLUrl(),
  credentials: 'include',
})

// Error handling link
const errorLink = onError(({ graphQLErrors, networkError, operation, forward }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      console.error(
        `GraphQL error: Message: ${message}, Location: ${locations}, Path: ${path}`
      )
    })
  }

  if (networkError) {
    console.error(`Network error: ${networkError}`)
    
    // Retry logic for network errors
    if ('statusCode' in networkError && (networkError.statusCode === 503 || networkError.statusCode === 502)) {
      return forward(operation)
    }
  }
})

// Auth link for future authentication
const authLink = setContext((_, { headers }) => {
  // Get authentication token if available
  const token = typeof window !== 'undefined' 
    ? localStorage.getItem('mcp_auth_token') 
    : null

  return {
    headers: {
      ...headers,
      ...(token && { authorization: `Bearer ${token}` }),
      'Content-Type': 'application/json',
    }
  }
})

// Create Apollo Client
export const apolloClient = new ApolloClient({
  link: from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      ConversationChunk: {
        keyFields: ['id'],
        fields: {
          related_chunks: {
            merge: (existing = [], incoming) => {
              return incoming
            }
          }
        }
      },
      SearchResults: {
        keyFields: false,
        fields: {
          results: {
            merge: (existing = [], incoming, { args }) => {
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
        keyFields: ['name']
      },
      Pattern: {
        keyFields: ['id']
      }
    }
  }),
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'all',
      notifyOnNetworkStatusChange: true,
    },
    query: {
      errorPolicy: 'all',
    },
    mutate: {
      errorPolicy: 'all',
    }
  },
  connectToDevTools: process.env.NODE_ENV === 'development',
})

// Helper function to handle GraphQL errors
export function handleGraphQLError(error: any): string {
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

// Cache management utilities
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
  updateMemory: (id: string, updates: Partial<any>) => {
    apolloClient.cache.modify({
      id: apolloClient.cache.identify({ __typename: 'ConversationChunk', id }),
      fields: {
        ...updates
      }
    })
  },

  // Invalidate search results
  invalidateSearchResults: () => {
    apolloClient.cache.evict({ fieldName: 'searchMemories' })
    apolloClient.cache.gc()
  }
}