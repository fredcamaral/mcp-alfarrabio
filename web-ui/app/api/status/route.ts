import { NextResponse } from 'next/server'
import { logger } from '@/lib/logger'
import apiClient from '@/lib/api-client'
import { generatePerformanceReport } from '@/lib/monitoring/performance'
import { generateErrorSummary } from '@/lib/monitoring/error-reporter'
import type { SystemStatus } from '@/types/api'

export async function GET(): Promise<NextResponse> {
  try {
    // Get backend status
    let backendStatus: { status: string; [key: string]: unknown } = { status: 'unknown' }
    try {
      const result = await apiClient.getStatus()
      backendStatus = result || { status: 'unknown' }
    } catch (error) {
      logger.warn('Backend status unavailable', { 
        error: String(error),
        component: 'StatusAPI'
      })
    }

    // Get performance metrics
    const performanceReport = generatePerformanceReport()
    
    // Get error summary
    const errorSummary = generateErrorSummary()
    
    // Get memory usage (only available in Node.js runtime)
    const memoryUsage = typeof process !== 'undefined' && 'memoryUsage' in process && typeof process.memoryUsage === 'function' ? process.memoryUsage() : null
    
    const response: SystemStatus = {
      status: 'operational',
      timestamp: new Date().toISOString(),
      uptime: typeof process !== 'undefined' && 'uptime' in process && typeof process.uptime === 'function' ? process.uptime() : 0,
      backend: backendStatus,
      performance: {
        avgResponseTime: 0, // Will be calculated from metrics
        requestsPerMinute: 0, // Will be calculated from metrics
        errorRate: 0, // Will be calculated from metrics
        ...performanceReport,
      },
      errors: {
        total: (errorSummary as { total?: number }).total || 0,
        byType: (errorSummary as { byType?: Record<string, number> }).byType || {},
        recent: (errorSummary as { recent?: Array<{ timestamp: string; error: string; count: number }> }).recent || [],
      },
      system: {
        memory: memoryUsage ? {
          rss: `${Math.round(memoryUsage.rss / 1024 / 1024)}MB`,
          heapTotal: `${Math.round(memoryUsage.heapTotal / 1024 / 1024)}MB`,
          heapUsed: `${Math.round(memoryUsage.heapUsed / 1024 / 1024)}MB`,
          external: `${Math.round(memoryUsage.external / 1024 / 1024)}MB`
        } : null,
        node: {
          version: typeof process !== 'undefined' && 'version' in process ? (process as NodeJS.Process).version : 'unknown',
          platform: typeof process !== 'undefined' && 'platform' in process ? (process as NodeJS.Process).platform : 'unknown',
          arch: typeof process !== 'undefined' && 'arch' in process ? (process as NodeJS.Process).arch : 'unknown'
        }
      },
      features: {
        csrf: true,
        websocket: !!process.env.NEXT_PUBLIC_WS_URL,
        graphql: !!process.env.NEXT_PUBLIC_GRAPHQL_URL,
        monitoring: true,
        errorBoundaries: true
      }
    }

    return NextResponse.json<SystemStatus>(response)
  } catch (error) {
    logger.error('Status check failed', error, {
      component: 'StatusAPI',
      action: 'GET'
    })
    
    return NextResponse.json(
      { 
        status: 'degraded',
        timestamp: new Date().toISOString(),
        error: 'Failed to gather status information'
      },
      { status: 500 }
    )
  }
}