/**
 * Multi-Repository API - Repositories Endpoint
 * 
 * Provides access to cross-repository data and analysis
 */

import { NextResponse } from 'next/server'
import { logger } from '@/lib/logger'

// Mock data for demonstration - in production, this would come from the backend
const mockRepositories = [
  {
    id: '00000000-0000-0000-0000-000000000001',
    name: 'lerian-mcp-memory',
    url: 'github.com/lerianstudio/lerian-mcp-memory',
    status: 'active',
    memoryCount: 342,
    lastUpdated: new Date().toISOString(),
    language: 'Go',
    size: 12400000,
    metadata: {
      stars: 45,
      contributors: 3,
      lastCommit: new Date().toISOString()
    }
  },
  {
    id: '00000000-0000-0000-0000-000000000002',
    name: 'midaz',
    url: 'github.com/lerianstudio/midaz',
    status: 'active',
    memoryCount: 156,
    lastUpdated: new Date(Date.now() - 86400000).toISOString(),
    language: 'Go',
    size: 8200000,
    metadata: {
      stars: 23,
      contributors: 5,
      lastCommit: new Date(Date.now() - 86400000).toISOString()
    }
  },
  {
    id: '00000000-0000-0000-0000-000000000003',
    name: 'transaction-api',
    url: 'github.com/lerianstudio/transaction-api',
    status: 'inactive',
    memoryCount: 89,
    lastUpdated: new Date(Date.now() - 172800000).toISOString(),
    language: 'TypeScript',
    size: 4500000,
    metadata: {
      stars: 12,
      contributors: 2,
      lastCommit: new Date(Date.now() - 604800000).toISOString()
    }
  }
]

export async function GET() {
  try {
    // In production, this would call the Go backend
    // const response = await fetch(`${BACKEND_URL}/api/multi-repo/repositories`)
    
    logger.info('Fetching multi-repository data')
    
    // Simulate some processing delay
    await new Promise(resolve => setTimeout(resolve, 100))
    
    return NextResponse.json({
      repositories: mockRepositories,
      totalCount: mockRepositories.length,
      status: 'success'
    })
  } catch (error) {
    logger.error('Error fetching repositories:', { error })
    return NextResponse.json(
      { error: 'Failed to fetch repositories' },
      { status: 500 }
    )
  }
}