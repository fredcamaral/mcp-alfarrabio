import { useQuery, useMutation, useSubscription, gql, Reference } from '@apollo/client'
import { CHUNK_ADDED_SUBSCRIPTION, PATTERN_DETECTED_SUBSCRIPTION } from './subscriptions'
import { logger } from '@/lib/logger'
import {
    SEARCH_MEMORIES,
    GET_CHUNK,
    LIST_CHUNKS,
    GET_PATTERNS,
    SUGGEST_RELATED,
    FIND_SIMILAR,
    TRACE_SESSION,
    TRACE_RELATED,
    STORE_CHUNK,
    STORE_DECISION,
    UPDATE_CHUNK,
    DELETE_CHUNK,
    LIST_REPOSITORIES,
    GET_REPOSITORY,
    GET_REPOSITORY_STATS,
    ADD_REPOSITORY,
    UPDATE_REPOSITORY,
    SYNC_REPOSITORY,
    REMOVE_REPOSITORY,
    type MemoryQueryInput,
    type StoreChunkInput,
    type StoreDecisionInput,
    type UpdateChunkInput,
    type ConversationChunk,
    type SearchResults,
    type Pattern,
    type ContextSuggestion,
    type Repository,
    type RepositoryStats,
    type AddRepositoryInput,
    type UpdateRepositoryInput
} from './queries'

// QUERY HOOKS

export function useSearchMemories(input: MemoryQueryInput, options?: Record<string, unknown>) {
    return useQuery<{ search: SearchResults }>(SEARCH_MEMORIES, {
        variables: { input },
        errorPolicy: 'all',
        notifyOnNetworkStatusChange: true,
        ...options
    })
}

export function useGetChunk(id: string, options?: Record<string, unknown>) {
    return useQuery<{ getChunk: ConversationChunk }>(GET_CHUNK, {
        variables: { id },
        errorPolicy: 'all',
        skip: !id,
        ...options
    })
}

export function useListChunks(repository: string, limit = 100, offset = 0, options?: Record<string, unknown>) {
    return useQuery<{ listChunks: ConversationChunk[] }>(LIST_CHUNKS, {
        variables: { repository, limit, offset },
        errorPolicy: 'all',
        notifyOnNetworkStatusChange: true,
        ...options
    })
}

export function useGetPatterns(repository: string, timeframe = 'month', options?: Record<string, unknown>) {
    return useQuery<{ getPatterns: Pattern[] }>(GET_PATTERNS, {
        variables: { repository, timeframe },
        errorPolicy: 'all',
        ...options
    })
}

export function useSuggestRelated(
    currentContext: string,
    sessionId: string,
    repository?: string,
    includePatterns = true,
    maxSuggestions = 5,
    options?: Record<string, unknown>
) {
    return useQuery<{ suggestRelated: ContextSuggestion }>(SUGGEST_RELATED, {
        variables: {
            currentContext,
            sessionId,
            repository,
            includePatterns,
            maxSuggestions
        },
        errorPolicy: 'all',
        skip: !currentContext || !sessionId,
        ...options
    })
}

export function useFindSimilar(problem: string, repository?: string, limit = 5, options?: Record<string, unknown>) {
    return useQuery<{ findSimilar: ConversationChunk[] }>(FIND_SIMILAR, {
        variables: { problem, repository, limit },
        errorPolicy: 'all',
        skip: !problem,
        ...options
    })
}

export function useTraceSession(sessionId: string, options?: Record<string, unknown>) {
    return useQuery<{ traceSession: ConversationChunk[] }>(TRACE_SESSION, {
        variables: { sessionId },
        errorPolicy: 'all',
        skip: !sessionId,
        ...options
    })
}

export function useTraceRelated(chunkId: string, depth = 2, options?: Record<string, unknown>) {
    return useQuery<{ traceRelated: ConversationChunk[] }>(TRACE_RELATED, {
        variables: { chunkId, depth },
        errorPolicy: 'all',
        skip: !chunkId,
        ...options
    })
}

// MUTATION HOOKS

export function useStoreChunk() {
    return useMutation<{ storeChunk: ConversationChunk }, { input: StoreChunkInput }>(STORE_CHUNK, {
        errorPolicy: 'all',
        update: (cache, { data }) => {
            if (data?.storeChunk) {
                // Update the cache with the new chunk
                cache.modify({
                    fields: {
                        listChunks(existingChunks = [], { readField }) {
                            const newChunkRef = cache.writeFragment({
                                data: data.storeChunk,
                                fragment: gql`
                  fragment NewChunk on ConversationChunk {
                    id
                    sessionId
                    repository
                    timestamp
                    content
                    type
                    tags
                  }
                `
                            })

                            // Add to the beginning of the list if it's not already there
                            if (existingChunks.some((ref: Reference) => readField('id', ref) === data.storeChunk.id)) {
                                return existingChunks
                            }
                            return [newChunkRef, ...existingChunks]
                        }
                    }
                })
            }
        }
    })
}

export function useStoreDecision() {
    return useMutation<{ storeDecision: ConversationChunk }, { input: StoreDecisionInput }>(STORE_DECISION, {
        errorPolicy: 'all',
        update: (cache, { data }) => {
            if (data?.storeDecision) {
                // Similar cache update logic as storeChunk
                cache.modify({
                    fields: {
                        listChunks(existingChunks = [], { readField }) {
                            const newChunkRef = cache.writeFragment({
                                data: data.storeDecision,
                                fragment: gql`
                  fragment NewDecision on ConversationChunk {
                    id
                    sessionId
                    repository
                    timestamp
                    content
                    type
                    decisionOutcome
                    decisionRationale
                  }
                `
                            })

                            if (existingChunks.some((ref: Reference) => readField('id', ref) === data.storeDecision.id)) {
                                return existingChunks
                            }
                            return [newChunkRef, ...existingChunks]
                        }
                    }
                })
            }
        }
    })
}

export function useUpdateChunk() {
    return useMutation<{ updateChunk: ConversationChunk }, { id: string; input: UpdateChunkInput }>(UPDATE_CHUNK, {
        errorPolicy: 'all',
        update: (cache, { data }, { variables }) => {
            if (data?.updateChunk && variables?.id) {
                // Update the cache with the updated chunk
                cache.writeFragment({
                    id: cache.identify({ __typename: 'ConversationChunk', id: variables.id }),
                    fragment: gql`
                        fragment UpdatedChunk on ConversationChunk {
                            id
                            sessionId
                            repository
                            timestamp
                            content
                            type
                            tags
                            summary
                            toolsUsed
                            filePaths
                        }
                    `,
                    data: data.updateChunk
                })
            }
        }
    })
}

export function useDeleteChunk() {
    return useMutation<{ deleteChunk: boolean }, { id: string }>(DELETE_CHUNK, {
        errorPolicy: 'all',
        update: (cache, { data }, { variables }) => {
            if (data?.deleteChunk && variables?.id) {
                // Remove from cache
                cache.evict({
                    id: cache.identify({ __typename: 'ConversationChunk', id: variables.id })
                })
                cache.gc()

                // Also remove from any lists
                cache.modify({
                    fields: {
                        listChunks(existingChunks, { readField }) {
                            return existingChunks.filter((chunkRef: Reference) =>
                                readField('id', chunkRef) !== variables.id
                            )
                        }
                    }
                })
            }
        }
    })
}

// UTILITY HOOKS

export function useMemoryOperations() {
    const [storeChunk, { loading: storingChunk }] = useStoreChunk()
    const [storeDecision, { loading: storingDecision }] = useStoreDecision()
    const [updateChunk, { loading: updatingChunk }] = useUpdateChunk()
    const [deleteChunk, { loading: deletingChunk }] = useDeleteChunk()

    return {
        storeChunk,
        storeDecision,
        updateChunk,
        deleteChunk,
        isLoading: storingChunk || storingDecision || updatingChunk || deletingChunk
    }
}

// Re-export types for external use
export type { StoreChunkInput, StoreDecisionInput, UpdateChunkInput } from './queries'

// Error handling utility
interface GraphQLError {
    message: string
    extensions?: Record<string, unknown>
}

interface NetworkError {
    statusCode?: number
    message?: string
}

interface ApolloError {
    networkError?: NetworkError
    graphQLErrors?: GraphQLError[]
    message?: string
}

function isApolloError(error: unknown): error is ApolloError {
    return typeof error === 'object' && error !== null && ('networkError' in error || 'graphQLErrors' in error)
}

export function useGraphQLError() {
    const handleError = (error: unknown) => {
        if (isApolloError(error)) {
            if (error.networkError) {
                if (error.networkError.statusCode === 503) {
                    return 'Service temporarily unavailable. Please try again.'
                }
                if (error.networkError.statusCode === 500) {
                    return 'Server error. Please contact support if this persists.'
                }
                return 'Network error. Please check your connection and ensure the backend server is running.'
            }

            if (error.graphQLErrors && error.graphQLErrors.length > 0) {
                return error.graphQLErrors[0].message
            }

            if (error.message) {
                return error.message
            }
        }

        if (error instanceof Error) {
            return error.message
        }

        return 'An unexpected error occurred'
    }

    return { handleError }
}

// REPOSITORY HOOKS

export function useListRepositories(status?: 'ACTIVE' | 'INACTIVE' | 'SYNCING' | 'ERROR', limit = 50, offset = 0, options?: Record<string, unknown>) {
    return useQuery<{ listRepositories: Repository[] }>(LIST_REPOSITORIES, {
        variables: { status, limit, offset },
        errorPolicy: 'all',
        notifyOnNetworkStatusChange: true,
        ...options
    })
}

export function useGetRepository(id: string, options?: Record<string, unknown>) {
    return useQuery<{ getRepository: Repository }>(GET_REPOSITORY, {
        variables: { id },
        errorPolicy: 'all',
        skip: !id,
        ...options
    })
}

export function useGetRepositoryStats(options?: Record<string, unknown>) {
    return useQuery<{ getRepositoryStats: RepositoryStats }>(GET_REPOSITORY_STATS, {
        errorPolicy: 'all',
        ...options
    })
}

export function useAddRepository() {
    return useMutation<{ addRepository: Repository }, { input: AddRepositoryInput }>(ADD_REPOSITORY, {
        errorPolicy: 'all',
        update: (cache, { data }) => {
            if (data?.addRepository) {
                // Add to repository list
                cache.modify({
                    fields: {
                        listRepositories(existingRepos = []) {
                            const newRepoRef = cache.writeFragment({
                                data: data.addRepository,
                                fragment: gql`
                                    fragment NewRepository on Repository {
                                        id
                                        url
                                        name
                                        description
                                        status
                                        memoryCount
                                        patternCount
                                        lastActivity
                                        createdAt
                                        updatedAt
                                    }
                                `
                            })
                            return [...existingRepos, newRepoRef]
                        }
                    }
                })
                
                // Update stats
                cache.modify({
                    fields: {
                        getRepositoryStats(existingStats) {
                            if (existingStats) {
                                return {
                                    ...existingStats,
                                    totalRepositories: existingStats.totalRepositories + 1,
                                    activeRepositories: data.addRepository.status === 'ACTIVE' 
                                        ? existingStats.activeRepositories + 1 
                                        : existingStats.activeRepositories
                                }
                            }
                            return existingStats
                        }
                    }
                })
            }
        }
    })
}

export function useUpdateRepository() {
    return useMutation<{ updateRepository: Repository }, { input: UpdateRepositoryInput }>(UPDATE_REPOSITORY, {
        errorPolicy: 'all'
    })
}

export function useSyncRepository() {
    return useMutation<{ syncRepository: Repository }, { id: string }>(SYNC_REPOSITORY, {
        errorPolicy: 'all',
        optimisticResponse: ({ id }) => ({
            syncRepository: {
                __typename: 'Repository' as const,
                id,
                url: '',
                name: '',
                status: 'SYNCING' as const,
                memoryCount: 0,
                patternCount: 0,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                metadata: {
                    syncedAt: new Date().toISOString()
                }
            }
        })
    })
}

export function useRemoveRepository() {
    return useMutation<{ removeRepository: boolean }, { id: string }>(REMOVE_REPOSITORY, {
        errorPolicy: 'all',
        update: (cache, { data }, { variables }) => {
            if (data?.removeRepository && variables?.id) {
                // Remove from cache
                cache.evict({
                    id: cache.identify({ __typename: 'Repository', id: variables.id })
                })
                cache.gc()
                
                // Update stats
                cache.modify({
                    fields: {
                        getRepositoryStats(existingStats) {
                            if (existingStats) {
                                return {
                                    ...existingStats,
                                    totalRepositories: Math.max(0, existingStats.totalRepositories - 1)
                                }
                            }
                            return existingStats
                        }
                    }
                })
            }
        }
    })
} 

// SUBSCRIPTION HOOKS

export function useChunkAddedSubscription(repository: string, options?: Record<string, unknown>) {
    return useSubscription<{ chunkAdded: ConversationChunk }>(
        CHUNK_ADDED_SUBSCRIPTION,
        {
            variables: { repository },
            skip: !repository,
            onError: (error) => {
                logger.error('Chunk subscription error:', error)
            },
            ...options
        }
    )
}

export function usePatternDetectedSubscription(repository: string, options?: Record<string, unknown>) {
    return useSubscription<{ patternDetected: Pattern }>(
        PATTERN_DETECTED_SUBSCRIPTION,
        {
            variables: { repository },
            skip: !repository,
            onError: (error) => {
                logger.error('Pattern subscription error:', error)
            },
            ...options
        }
    )
}

// Hook to combine real-time updates with queries
export function useRealtimeChunks(repository: string, _sessionId?: string, limit = 100, offset = 0) {
    // Regular query for initial data
    const queryResult = useListChunks(repository, limit, offset)
    
    // Subscribe to new chunks
    const { data: subscriptionData } = useChunkAddedSubscription(repository, {
        onData: ({ data }: { data: { data?: { chunkAdded: ConversationChunk } } }) => {
            if (data && data.data?.chunkAdded) {
                // Update cache with new chunk
                queryResult.client.cache.modify({
                    fields: {
                        listChunks(existingChunks = []) {
                            const newChunkRef = queryResult.client.cache.writeFragment({
                                data: data.data!.chunkAdded,
                                fragment: gql`
                                    fragment NewChunk on ConversationChunk {
                                        id
                                        sessionId
                                        repository
                                        timestamp
                                        content
                                        summary
                                        type
                                        tags
                                    }
                                `
                            })
                            return [newChunkRef, ...existingChunks]
                        }
                    }
                })
            }
        }
    })
    
    return {
        ...queryResult,
        hasNewChunk: !!subscriptionData?.chunkAdded
    }
}

// Hook for real-time pattern updates
export function useRealtimePatterns(repository: string) {
    // Regular query for initial patterns
    const queryResult = useGetPatterns(repository)
    
    // Subscribe to new patterns
    const { data: subscriptionData } = usePatternDetectedSubscription(repository, {
        onData: ({ data }: { data: { data?: { patternDetected: Pattern } } }) => {
            if (data && data.data?.patternDetected) {
                // Refetch patterns when new one is detected
                queryResult.refetch()
            }
        }
    })
    
    return {
        ...queryResult,
        hasNewPattern: !!subscriptionData?.patternDetected,
        latestPattern: subscriptionData?.patternDetected
    }
}
