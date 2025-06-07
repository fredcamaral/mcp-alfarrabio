/**
 * GraphQL Configuration Operations
 * 
 * Queries and mutations for system configuration
 */

import { gql } from '@apollo/client'

// Types
export interface TransportProtocols {
  http: boolean
  websocket: boolean
  grpc: boolean
}

export interface VectorDbConfig {
  provider: 'qdrant' | 'chroma'
  host: string
  port: number
  collection: string
  dimension: number
}

export interface OpenAIConfig {
  apiKey?: string
  model: string
  maxTokens: number
  temperature: number
  timeout: number
}

export interface FeatureConfig {
  csrf: boolean
  websocket: boolean
  graphql: boolean
  monitoring: boolean
  errorBoundaries: boolean
  cacheEnabled: boolean
  realtimeEnabled: boolean
  analyticsEnabled: boolean
  debugMode: boolean
  authEnabled: boolean
}

export interface SystemConfig {
  host: string
  port: number
  protocol: 'http' | 'https'
  transportProtocols: TransportProtocols
  vectorDb: VectorDbConfig
  openai: OpenAIConfig
  features: FeatureConfig
}

export interface ConfigInput {
  host?: string
  port?: number
  protocol?: 'http' | 'https'
  transportProtocols?: Partial<TransportProtocols>
  vectorDb?: Partial<VectorDbConfig>
  openai?: Partial<OpenAIConfig>
  features?: Partial<FeatureConfig>
}

// Queries
export const GET_CONFIG = gql`
  query GetConfig {
    getConfig {
      host
      port
      protocol
      transportProtocols {
        http
        websocket
        grpc
      }
      vectorDb {
        provider
        host
        port
        collection
        dimension
      }
      openai {
        model
        maxTokens
        temperature
        timeout
      }
      features {
        csrf
        websocket
        graphql
        monitoring
        errorBoundaries
        cacheEnabled
        realtimeEnabled
        analyticsEnabled
        debugMode
        authEnabled
      }
    }
  }
`

// Mutations
export const UPDATE_CONFIG = gql`
  mutation UpdateConfig($input: ConfigInput!) {
    updateConfig(input: $input) {
      host
      port
      protocol
      transportProtocols {
        http
        websocket
        grpc
      }
      vectorDb {
        provider
        host
        port
        collection
        dimension
      }
      openai {
        model
        maxTokens
        temperature
        timeout
      }
      features {
        csrf
        websocket
        graphql
        monitoring
        errorBoundaries
        cacheEnabled
        realtimeEnabled
        analyticsEnabled
        debugMode
        authEnabled
      }
    }
  }
`