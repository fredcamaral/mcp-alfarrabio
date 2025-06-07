import { NextRequest, NextResponse } from 'next/server'
import { logger } from '@/lib/logger'

interface ErrorReport {
  message: string
  stack?: string
  component?: string
  action?: string
  metadata?: Record<string, unknown>
  timestamp: string
  userAgent?: string
  url?: string
}

export async function POST(request: NextRequest) {
  try {
    const errorReport: ErrorReport = await request.json()
    
    // Log the error server-side
    logger.error(`Client error: ${errorReport.message}`, new Error(errorReport.stack || errorReport.message), {
      component: errorReport.component || 'Unknown',
      action: errorReport.action || 'client-error',
      ...errorReport.metadata,
      userAgent: errorReport.userAgent || request.headers.get('user-agent') || undefined,
      url: errorReport.url || request.headers.get('referer') || undefined,
      timestamp: errorReport.timestamp
    })
    
    // In production, you might want to send this to an error tracking service
    // like Sentry, LogRocket, etc.
    
    return NextResponse.json({ 
      status: 'logged',
      message: 'Error report received' 
    })
  } catch (error) {
    logger.error('Failed to process error report', error, {
      component: 'ErrorReportRoute',
      action: 'process-error'
    })
    
    return NextResponse.json(
      { 
        error: 'Failed to process error report',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

export async function OPTIONS() {
  return new NextResponse(null, { status: 200 })
}