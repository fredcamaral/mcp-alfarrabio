import { gql } from '@apollo/client'

// Fragment for ConversationChunk
export const CONVERSATION_CHUNK_FRAGMENT = gql`
  fragment ConversationChunkFragment on ConversationChunk {
    id
    sessionId
    repository
    branch
    timestamp
    content
    summary
    type
    tags
    toolsUsed
    filePaths
    concepts
    entities
    decisionOutcome
    decisionRationale
    difficultyLevel
    problemDescription
    solutionApproach
    outcome
    lessonsLearned
    nextSteps
  }
`

// Fragment for ScoredChunk
export const SCORED_CHUNK_FRAGMENT = gql`
  fragment ScoredChunkFragment on ScoredChunk {
    chunk {
      ...ConversationChunkFragment
    }
    score
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

// QUERIES

export const SEARCH_MEMORIES = gql`
  query SearchMemories($input: MemoryQueryInput!) {
    search(input: $input) {
      chunks {
        ...ScoredChunkFragment
      }
    }
  }
  ${SCORED_CHUNK_FRAGMENT}
`

export const GET_CHUNK = gql`
  query GetChunk($id: String!) {
    getChunk(id: $id) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const LIST_CHUNKS = gql`
  query ListChunks($repository: String!, $limit: Int, $offset: Int) {
    listChunks(repository: $repository, limit: $limit, offset: $offset) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const GET_PATTERNS = gql`
  query GetPatterns($repository: String!, $timeframe: String) {
    getPatterns(repository: $repository, timeframe: $timeframe) {
      type
      count
      confidence
      lastSeen
      examples
    }
  }
`

export const SUGGEST_RELATED = gql`
  query SuggestRelated(
    $currentContext: String!
    $sessionId: String!
    $repository: String
    $includePatterns: Boolean
    $maxSuggestions: Int
  ) {
    suggestRelated(
      currentContext: $currentContext
      sessionId: $sessionId
      repository: $repository
      includePatterns: $includePatterns
      maxSuggestions: $maxSuggestions
    ) {
      relevantChunks {
        ...ScoredChunkFragment
      }
      suggestedTasks
      relatedConcepts
      potentialIssues
    }
  }
  ${SCORED_CHUNK_FRAGMENT}
`

export const FIND_SIMILAR = gql`
  query FindSimilar($problem: String!, $repository: String, $limit: Int) {
    findSimilar(problem: $problem, repository: $repository, limit: $limit) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const TRACE_SESSION = gql`
  query TraceSession($sessionId: String!) {
    traceSession(sessionId: $sessionId) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const TRACE_RELATED = gql`
  query TraceRelated($chunkId: String!, $depth: Int) {
    traceRelated(chunkId: $chunkId, depth: $depth) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

// MUTATIONS

export const STORE_CHUNK = gql`
  mutation StoreChunk($input: StoreChunkInput!) {
    storeChunk(input: $input) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const STORE_DECISION = gql`
  mutation StoreDecision($input: StoreDecisionInput!) {
    storeDecision(input: $input) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const UPDATE_CHUNK = gql`
  mutation UpdateChunk($id: String!, $input: UpdateChunkInput!) {
    updateChunk(id: $id, input: $input) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const DELETE_CHUNK = gql`
  mutation DeleteChunk($id: String!) {
    deleteChunk(id: $id)
  }
`

// Type definitions for TypeScript
export interface MemoryQueryInput {
    query?: string
    repository?: string
    types?: string[]
    tags?: string[]
    limit?: number
    minRelevanceScore?: number
    recency?: string
}

export interface StoreChunkInput {
    content: string
    sessionId: string
    repository?: string
    branch?: string
    tags?: string[]
    toolsUsed?: string[]
    filesModified?: string[]
}

export interface StoreDecisionInput {
    decision: string
    rationale: string
    sessionId: string
    repository?: string
    context?: string
}

export interface UpdateChunkInput {
    content?: string
    type?: string
    tags?: string[]
    summary?: string
    repository?: string
    branch?: string
    toolsUsed?: string[]
    filesModified?: string[]
}

// Use consistent types from the main type definitions
import type { ConversationChunk, ChunkType } from '@/types/memory'
export type { ConversationChunk, ChunkType } from '@/types/memory'

// GraphQL-specific interface that maps to ConversationChunk but uses GraphQL naming conventions
export interface GraphQLConversationChunk {
    id: string
    sessionId: string  // GraphQL uses sessionId while main type uses session_id
    repository?: string
    branch?: string
    timestamp: string
    content: string
    summary?: string
    type: string
    tags?: string[]
    toolsUsed?: string[]
    filePaths?: string[]
    concepts?: string[]
    entities?: string[]
    decisionOutcome?: string
    decisionRationale?: string
    difficultyLevel?: string
    problemDescription?: string
    solutionApproach?: string
    outcome?: string
    lessonsLearned?: string
    nextSteps?: string
}

export interface ScoredChunk {
    chunk: GraphQLConversationChunk
    score: number
}

// Conversion utilities between REST API (snake_case) and GraphQL (camelCase)
export function convertGraphQLToREST(chunk: GraphQLConversationChunk): ConversationChunk {
    return {
        id: chunk.id,
        session_id: chunk.sessionId,
        timestamp: chunk.timestamp,
        type: chunk.type as ChunkType,
        content: chunk.content,
        summary: chunk.summary,
        metadata: {
            repository: chunk.repository,
            branch: chunk.branch,
            tags: chunk.tags,
            tools_used: chunk.toolsUsed,
            files_modified: chunk.filePaths,
            // Map GraphQL fields to metadata structure
            extended_metadata: {
                concepts: chunk.concepts,
                entities: chunk.entities,
                decision_outcome: chunk.decisionOutcome,
                decision_rationale: chunk.decisionRationale,
                difficulty_level: chunk.difficultyLevel,
                problem_description: chunk.problemDescription,
                solution_approach: chunk.solutionApproach,
                outcome: chunk.outcome,
                lessons_learned: chunk.lessonsLearned,
                next_steps: chunk.nextSteps
            }
        },
        embeddings: [],
        related_chunks: []
    }
}

export function convertRESTToGraphQL(chunk: ConversationChunk): GraphQLConversationChunk {
    const extended = chunk.metadata.extended_metadata || {}
    return {
        id: chunk.id,
        sessionId: chunk.session_id,
        repository: chunk.metadata.repository,
        branch: chunk.metadata.branch,
        timestamp: chunk.timestamp,
        content: chunk.content,
        summary: chunk.summary,
        type: chunk.type,
        tags: chunk.metadata.tags,
        toolsUsed: chunk.metadata.tools_used,
        filePaths: chunk.metadata.files_modified,
        concepts: Array.isArray(extended.concepts) ? extended.concepts as string[] : undefined,
        entities: Array.isArray(extended.entities) ? extended.entities as string[] : undefined,
        decisionOutcome: typeof extended.decision_outcome === 'string' ? extended.decision_outcome : undefined,
        decisionRationale: typeof extended.decision_rationale === 'string' ? extended.decision_rationale : undefined,
        difficultyLevel: typeof extended.difficulty_level === 'string' ? extended.difficulty_level : undefined,
        problemDescription: typeof extended.problem_description === 'string' ? extended.problem_description : undefined,
        solutionApproach: typeof extended.solution_approach === 'string' ? extended.solution_approach : undefined,
        outcome: typeof extended.outcome === 'string' ? extended.outcome : undefined,
        lessonsLearned: typeof extended.lessons_learned === 'string' ? extended.lessons_learned : undefined,
        nextSteps: typeof extended.next_steps === 'string' ? extended.next_steps : undefined
    }
}

export interface SearchResults {
    chunks: ScoredChunk[]
}

export interface Pattern {
    type: string
    count: number
    confidence: number
    lastSeen: string
    examples: string[]
}

export interface ContextSuggestion {
    relevantChunks: ScoredChunk[]
    suggestedTasks: string[]
    relatedConcepts: string[]
    potentialIssues: string[]
}

// Pattern Queries
export const GET_PATTERN_STATS = gql`
  query GetPatternStats($repository: String!) {
    getPatternStats(repository: $repository) {
      type
      count
      examples
      confidence
      lastSeen
    }
  }
`

export const TRACE_SESSION_DETAILED = gql`
  query TraceSessionDetailed($sessionId: String!) {
    traceSessionDetailed(sessionId: $sessionId) {
      sessionId
      repository
      startTime
      endTime
      chunks {
        ...ConversationChunkFragment
      }
      patterns {
        type
        count
        examples
        confidence
        lastSeen
      }
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

export const TRACE_RELATED_CHUNKS = gql`
  query TraceRelatedChunks($chunkId: String!) {
    traceRelatedChunks(chunkId: $chunkId) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

// Repository Queries
export const LIST_REPOSITORIES = gql`
  query ListRepositories($status: RepositoryStatus, $limit: Int, $offset: Int) {
    listRepositories(status: $status, limit: $limit, offset: $offset) {
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
      metadata {
        technologies
        contributors
        branches
        syncedAt
      }
    }
  }
`

export const GET_REPOSITORY = gql`
  query GetRepository($id: String!) {
    getRepository(id: $id) {
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
      metadata {
        technologies
        contributors
        branches
        syncedAt
      }
    }
  }
`

export const GET_REPOSITORY_STATS = gql`
  query GetRepositoryStats {
    getRepositoryStats {
      totalRepositories
      activeRepositories
      totalMemories
      totalPatterns
      recentActivity
    }
  }
`

// Repository Mutations
export const ADD_REPOSITORY = gql`
  mutation AddRepository($input: AddRepositoryInput!) {
    addRepository(input: $input) {
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
  }
`

export const UPDATE_REPOSITORY = gql`
  mutation UpdateRepository($input: UpdateRepositoryInput!) {
    updateRepository(input: $input) {
      id
      url
      name
      description
      status
      memoryCount
      patternCount
      lastActivity
      updatedAt
    }
  }
`

export const SYNC_REPOSITORY = gql`
  mutation SyncRepository($id: String!) {
    syncRepository(id: $id) {
      id
      status
      metadata {
        syncedAt
      }
    }
  }
`

export const REMOVE_REPOSITORY = gql`
  mutation RemoveRepository($id: String!) {
    removeRepository(id: $id)
  }
`

// Repository Types
export interface Repository {
  id: string
  url: string
  name: string
  description?: string
  status: 'ACTIVE' | 'INACTIVE' | 'SYNCING' | 'ERROR'
  memoryCount: number
  patternCount: number
  lastActivity?: string
  createdAt: string
  updatedAt: string
  metadata?: {
    technologies?: string[]
    contributors?: number
    branches?: string[]
    syncedAt?: string
  }
}

export interface RepositoryStats {
  totalRepositories: number
  activeRepositories: number
  totalMemories: number
  totalPatterns: number
  recentActivity: number
}

export interface AddRepositoryInput {
  url: string
  description?: string
}

export interface UpdateRepositoryInput {
  id: string
  description?: string
  status?: 'ACTIVE' | 'INACTIVE' | 'SYNCING' | 'ERROR'
} 