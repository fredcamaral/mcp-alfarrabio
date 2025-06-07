import { NextRequest, NextResponse } from 'next/server'
import { logger } from '@/lib/logger'
import apiClient from '@/lib/api-client'

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams
    const repository = searchParams.get('repository') || process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory'
    const timeframe = searchParams.get('timeframe') || 'month'

    // Call the backend API to analyze patterns
    const response = await apiClient.analyzeMemories({
      operation: 'get_patterns',
      options: {
        repository,
        timeframe
      }
    })

    // Transform the response to match expected format
    const patterns = (response.result as { patterns?: unknown[] })?.patterns || []
    
    return NextResponse.json({
      patterns,
      repository,
      timeframe,
      analyzedAt: new Date().toISOString()
    })
  } catch (error) {
    logger.error('Failed to fetch patterns', error, {
      component: 'PatternsAPI',
      action: 'GET'
    })
    
    return NextResponse.json(
      { error: 'Failed to fetch patterns' },
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { repository, operation = 'analyze_patterns' } = body

    // Analyze patterns for the repository
    const response = await apiClient.analyzeMemories({
      operation,
      options: {
        repository: repository || process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory',
        ...body.options
      }
    })

    logger.info('Pattern analysis completed', {
      component: 'PatternsAPI',
      action: 'POST',
      repository,
      operation
    })

    return NextResponse.json({
      success: true,
      result: response.result,
      operation: response.operation,
      repository: response.repository
    })
  } catch (error) {
    logger.error('Failed to analyze patterns', error, {
      component: 'PatternsAPI',
      action: 'POST'
    })
    
    return NextResponse.json(
      { error: 'Failed to analyze patterns' },
      { status: 500 }
    )
  }
}