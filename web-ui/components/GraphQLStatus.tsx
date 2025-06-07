'use client'

import { useQuery, gql } from '@apollo/client'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle2, Loader2 } from 'lucide-react'

// Use a simple query to check GraphQL connectivity
const CONNECTIVITY_CHECK = gql`
  query ConnectivityCheck {
    listChunks(repository: "test", limit: 1) {
      id
    }
  }
`

export function GraphQLStatus() {
  const { loading, error } = useQuery(CONNECTIVITY_CHECK, {
    pollInterval: 30000, // Poll every 30 seconds
    errorPolicy: 'all',
    fetchPolicy: 'network-only'
  })

  if (loading) {
    return (
      <Badge variant="secondary" className="gap-1">
        <Loader2 className="h-3 w-3 animate-spin" />
        GraphQL: Connecting...
      </Badge>
    )
  }

  if (error) {
    return (
      <Badge variant="destructive" className="gap-1">
        <AlertCircle className="h-3 w-3" />
        GraphQL: Offline
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="gap-1 text-success border-success">
      <CheckCircle2 className="h-3 w-3" />
      GraphQL: Connected
    </Badge>
  )
}