import { NextRequest, NextResponse } from 'next/server'
import { logger } from '@/lib/logger'

const MCP_SERVER_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    
    // Forward the request to the actual MCP server
    const response = await fetch(`${MCP_SERVER_URL}/mcp`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      throw new Error(`MCP server responded with ${response.status}`)
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    logger.error('MCP proxy error', error, {
      component: 'MCPRoute',
      action: 'proxy-request'
    })
    
    return NextResponse.json(
      { 
        error: 'Failed to process MCP request',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

export async function OPTIONS() {
  return new NextResponse(null, { status: 200 })
}