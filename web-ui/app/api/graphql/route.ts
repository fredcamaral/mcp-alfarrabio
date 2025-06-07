/**
 * GraphQL API Route
 * 
 * Proxies GraphQL requests to the backend GraphQL server
 */

import { NextRequest, NextResponse } from 'next/server'
import { logger } from '@/lib/logger'

const GRAPHQL_ENDPOINT = process.env.GRAPHQL_ENDPOINT || 'http://localhost:8082/graphql'

export async function POST(request: NextRequest) {
  try {
    const body = await request.text()
    
    // Forward the request to the backend GraphQL server
    const response = await fetch(GRAPHQL_ENDPOINT, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body,
    })

    if (!response.ok) {
      logger.error('GraphQL backend error:', {
        status: response.status,
        statusText: response.statusText,
        endpoint: GRAPHQL_ENDPOINT
      })
      
      return NextResponse.json(
        { error: 'GraphQL backend error', status: response.status },
        { status: response.status }
      )
    }

    const data = await response.json()
    
    // Log GraphQL errors if present
    if (data.errors) {
      logger.warn('GraphQL query errors:', { errors: data.errors })
    }

    return NextResponse.json(data)
  } catch (error) {
    logger.error('GraphQL proxy error:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to connect to GraphQL server',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

export async function GET(request: NextRequest) {
  // Handle GraphQL introspection queries via GET
  const { searchParams } = new URL(request.url)
  const query = searchParams.get('query')
  
  if (!query) {
    return NextResponse.json(
      { error: 'Missing query parameter' },
      { status: 400 }
    )
  }

  try {
    const response = await fetch(GRAPHQL_ENDPOINT, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify({ query }),
    })

    if (!response.ok) {
      return NextResponse.json(
        { error: 'GraphQL backend error', status: response.status },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    logger.error('GraphQL GET proxy error:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to connect to GraphQL server',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    },
  })
}