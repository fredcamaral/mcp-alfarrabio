import { gql } from '@apollo/client'
import { CONVERSATION_CHUNK_FRAGMENT } from './queries'

// Subscription for new chunks added to a repository
export const CHUNK_ADDED_SUBSCRIPTION = gql`
  subscription ChunkAdded($repository: String!) {
    chunkAdded(repository: $repository) {
      ...ConversationChunkFragment
    }
  }
  ${CONVERSATION_CHUNK_FRAGMENT}
`

// Subscription for pattern detection
export const PATTERN_DETECTED_SUBSCRIPTION = gql`
  subscription PatternDetected($repository: String!) {
    patternDetected(repository: $repository) {
      type
      count
      examples
      confidence
      lastSeen
    }
  }
`

// Future subscriptions can be added here:
// - Repository status changes
// - Memory relationships created
// - Conflict detection alerts
// - Session activity updates