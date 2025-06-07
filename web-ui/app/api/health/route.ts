import { NextResponse } from 'next/server'
import { logger } from '@/lib/logger'
import apiClient from '@/lib/api-client'

export async function GET() {
  try {
    // Check backend health
    const backendHealth = await apiClient.getHealth()
    
    // Check WebSocket status (optional)
    const wsStatus = process.env.NEXT_PUBLIC_WS_URL ? 'configured' : 'not configured'
    
    // Check GraphQL endpoint
    const graphqlStatus = process.env.NEXT_PUBLIC_GRAPHQL_URL ? 'configured' : 'not configured'
    
    const response = {
      status: 'healthy',
      timestamp: new Date().toISOString(),
      services: {
        backend: {
          status: backendHealth.status,
          timestamp: backendHealth.timestamp
        },
        websocket: {
          status: wsStatus,
          url: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:9080/ws'
        },
        graphql: {
          status: graphqlStatus,
          url: process.env.NEXT_PUBLIC_GRAPHQL_URL || 'http://localhost:9080/graphql'
        }
      },
      environment: {
        nodeEnv: process.env.NODE_ENV,
        apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'
      }
    }

    return NextResponse.json(response)
  } catch (error) {
    logger.error('Health check failed', error, {
      component: 'HealthAPI',
      action: 'GET'
    })
    
    const errorResponse = {
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      error: error instanceof Error ? error.message : 'Backend connection failed',
      services: {
        backend: {
          status: 'error',
          message: 'Unable to connect to backend'
        }
      }
    }
    
    return NextResponse.json(errorResponse, { status: 503 })
  }
}