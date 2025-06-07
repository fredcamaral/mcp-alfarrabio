/**
 * GraphQL Hooks for Memory Operations
 * 
 * Custom hooks that wrap Apollo GraphQL queries and mutations
 * for memory-related operations with proper TypeScript typing.
 */

'use client'

import { useQuery, useMutation } from '@apollo/client'
import { 
  LIST_CHUNKS,
  SEARCH_MEMORIES,
  GET_PATTERNS,
  LIST_REPOSITORIES,
  GET_REPOSITORY_STATS,
  STORE_CHUNK,
  DELETE_CHUNK,
  ADD_REPOSITORY,
  SYNC_REPOSITORY,
  REMOVE_REPOSITORY
} from '@/lib/graphql/queries'
import type {
  MemoryQueryInput
} from '@/lib/graphql/queries'

// Query Hooks
export function useListChunks(repository: string, limit = 100, offset = 0, options?: any) {
  return useQuery(LIST_CHUNKS, {
    variables: { repository, limit, offset },
    ...options
  })
}

export function useSearchMemories(input: MemoryQueryInput, options?: any) {
  return useQuery(SEARCH_MEMORIES, {
    variables: { input },
    skip: !input.query?.trim(),
    ...options
  })
}

export function useGetPatterns(repository: string, options?: any) {
  return useQuery(GET_PATTERNS, {
    variables: { repository },
    ...options
  })
}

export function useListRepositories(status?: string, options?: any) {
  return useQuery(LIST_REPOSITORIES, {
    variables: { status },
    ...options
  })
}

export function useGetRepositoryStats(options?: any) {
  return useQuery(GET_REPOSITORY_STATS, options)
}

// Mutation Hooks
export function useStoreChunk() {
  return useMutation(STORE_CHUNK)
}

export function useDeleteChunk() {
  return useMutation(DELETE_CHUNK)
}

export function useAddRepository() {
  return useMutation(ADD_REPOSITORY)
}

export function useSyncRepository() {
  return useMutation(SYNC_REPOSITORY)
}

export function useRemoveRepository() {
  return useMutation(REMOVE_REPOSITORY)
}

// Subscription Hooks
// TODO: Add subscription support when available in the GraphQL schema

// Error Handling Hook
export function useGraphQLError() {
  const handleError = (error: any): string => {
    if (error.graphQLErrors?.length > 0) {
      return error.graphQLErrors[0].message
    }
    if (error.networkError) {
      return 'Network error. Please check your connection.'
    }
    return error.message || 'An unexpected error occurred'
  }

  return { handleError }
}