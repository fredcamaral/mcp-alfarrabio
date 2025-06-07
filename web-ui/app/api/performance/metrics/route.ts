import { NextRequest, NextResponse } from 'next/server'
import { headers } from 'next/headers'
import { logger } from '@/lib/logger'

// Types for performance metrics
interface PerformanceMetric {
  id: string
  name: string
  value: number
  delta: number
  rating: 'good' | 'needs-improvement' | 'poor'
  timestamp: number
  url: string
  userAgent: string
}

interface PerformanceBatch {
  metrics: PerformanceMetric[]
  sessionId: string
  timestamp: number
}

interface PerformanceInsights {
  alerts: string[]
  recommendations: string[]
  summary: Record<string, MetricSummary>
}

interface MetricSummary {
  average: number
  count: number
  poorPercentage: number
}

interface MetricStatistics {
  count: number
  average: number
  median: number
  p75: number
  p90: number
  p95: number
  min: number
  max: number
  ratings: {
    good: number
    needsImprovement: number
    poor: number
  }
}

interface MetricAccumulator {
  values: number[]
  ratings: {
    good: number
    'needs-improvement': number
    poor: number
  }
}

// In-memory storage for demo (use database in production)
const performanceData = new Map<string, PerformanceMetric[]>()

export async function POST(request: NextRequest): Promise<NextResponse> {
  try {
    const headersList = await headers()
    const userAgent = headersList.get('user-agent') || 'Unknown'
    const origin = headersList.get('origin')
    
    // CORS handling
    if (origin) {
      const response = new NextResponse()
      response.headers.set('Access-Control-Allow-Origin', origin)
      response.headers.set('Access-Control-Allow-Methods', 'POST, OPTIONS')
      response.headers.set('Access-Control-Allow-Headers', 'Content-Type')
    }

    const body = await request.json() as PerformanceBatch

    if (!body.metrics || !Array.isArray(body.metrics)) {
      return NextResponse.json(
        { error: 'Invalid metrics data' },
        { status: 400 }
      )
    }

    // Validate and process metrics
    const validMetrics = body.metrics.filter(metric => {
      return (
        metric.id &&
        metric.name &&
        typeof metric.value === 'number' &&
        typeof metric.timestamp === 'number'
      )
    })

    if (validMetrics.length === 0) {
      return NextResponse.json(
        { error: 'No valid metrics found' },
        { status: 400 }
      )
    }

    // Store metrics (in production, save to database)
    const sessionId = body.sessionId || 'anonymous'
    const existingMetrics = performanceData.get(sessionId) || []
    const allMetrics = [...existingMetrics, ...validMetrics]
    
    // Keep only last 1000 metrics per session
    if (allMetrics.length > 1000) {
      allMetrics.splice(0, allMetrics.length - 1000)
    }
    
    performanceData.set(sessionId, allMetrics)

    // Log critical metrics
    const criticalMetrics = validMetrics.filter(m => m.rating === 'poor')
    if (criticalMetrics.length > 0) {
      logger.warn('Critical performance metrics detected', {
        sessionId,
        criticalMetricsCount: String(criticalMetrics.length),
        metrics: JSON.stringify(criticalMetrics.map(m => ({
          name: m.name,
          value: m.value,
          url: m.url
        }))),
        userAgent
      })
    }

    // Performance insights
    const insights = generatePerformanceInsights(validMetrics)

    return NextResponse.json({
      success: true,
      processed: validMetrics.length,
      insights,
      timestamp: Date.now()
    })

  } catch (error) {
    logger.error('Performance metrics API error', error, {
      component: 'performance-metrics',
      action: 'POST'
    })
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function GET(request: NextRequest): Promise<NextResponse> {
  try {
    const { searchParams } = new URL(request.url)
    const sessionId = searchParams.get('sessionId')
    const metric = searchParams.get('metric')
    const limit = parseInt(searchParams.get('limit') || '100')

    if (!sessionId) {
      // Return aggregated data
      const allMetrics: PerformanceMetric[] = []
      performanceData.forEach(metrics => allMetrics.push(...metrics))
      
      const summary = generateSummaryStats(allMetrics, metric)
      
      return NextResponse.json({
        summary,
        totalSessions: performanceData.size,
        totalMetrics: allMetrics.length
      })
    }

    // Return session-specific data
    const metrics = performanceData.get(sessionId) || []
    const filteredMetrics = metric 
      ? metrics.filter(m => m.name === metric)
      : metrics

    const recentMetrics = filteredMetrics
      .sort((a, b) => b.timestamp - a.timestamp)
      .slice(0, limit)

    return NextResponse.json({
      sessionId,
      metrics: recentMetrics,
      total: filteredMetrics.length
    })

  } catch (error) {
    logger.error('Performance metrics GET error', error, {
      component: 'performance-metrics',
      action: 'GET'
    })
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function OPTIONS(): Promise<NextResponse> {
  const headersList = await headers()
  const origin = headersList.get('origin')
  
  const response = new NextResponse(null, { status: 200 })
  
  if (origin) {
    response.headers.set('Access-Control-Allow-Origin', origin)
  }
  response.headers.set('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
  response.headers.set('Access-Control-Allow-Headers', 'Content-Type')
  response.headers.set('Access-Control-Max-Age', '86400')
  
  return response
}

// Generate performance insights
function generatePerformanceInsights(metrics: PerformanceMetric[]): PerformanceInsights {
  const insights: PerformanceInsights = {
    alerts: [],
    recommendations: [],
    summary: {}
  }

  const metricsByName = metrics.reduce((acc, metric) => {
    if (!acc[metric.name]) acc[metric.name] = []
    acc[metric.name].push(metric)
    return acc
  }, {} as Record<string, PerformanceMetric[]>)

  // Analyze each metric type
  Object.entries(metricsByName).forEach(([name, values]) => {
    const averageValue = values.reduce((sum, m) => sum + m.value, 0) / values.length
    const poorCount = values.filter(m => m.rating === 'poor').length
    const poorPercentage = (poorCount / values.length) * 100

    insights.summary[name] = {
      average: Math.round(averageValue * 100) / 100,
      count: values.length,
      poorPercentage: Math.round(poorPercentage * 100) / 100
    }

    // Generate alerts and recommendations
    if (poorPercentage > 20) {
      insights.alerts.push(`${name}: ${poorPercentage.toFixed(1)}% of measurements are poor`)
    }

    switch (name) {
      case 'LCP':
        if (averageValue > 4000) {
          insights.recommendations.push('Optimize Largest Contentful Paint: consider image optimization and server-side rendering')
        }
        break
      case 'FID':
        if (averageValue > 300) {
          insights.recommendations.push('Improve First Input Delay: reduce JavaScript execution time and split code')
        }
        break
      case 'CLS':
        if (averageValue > 0.25) {
          insights.recommendations.push('Fix Cumulative Layout Shift: set dimensions for images and avoid dynamic content insertion')
        }
        break
      case 'FCP':
        if (averageValue > 3000) {
          insights.recommendations.push('Optimize First Contentful Paint: minimize render-blocking resources')
        }
        break
      case 'TTFB':
        if (averageValue > 1800) {
          insights.recommendations.push('Improve Time to First Byte: optimize server response time and use CDN')
        }
        break
    }
  })

  return insights
}

// Generate summary statistics
function generateSummaryStats(metrics: PerformanceMetric[], filterMetric?: string | null): Record<string, MetricStatistics> | { message: string } {
  const filteredMetrics = filterMetric 
    ? metrics.filter(m => m.name === filterMetric)
    : metrics

  if (filteredMetrics.length === 0) {
    return { message: 'No metrics found' }
  }

  const metricsByName = filteredMetrics.reduce((acc, metric) => {
    if (!acc[metric.name]) {
      acc[metric.name] = {
        values: [],
        ratings: { good: 0, 'needs-improvement': 0, poor: 0 }
      }
    }
    acc[metric.name].values.push(metric.value)
    acc[metric.name].ratings[metric.rating]++
    return acc
  }, {} as Record<string, MetricAccumulator>)

  const summary: Record<string, MetricStatistics> = {}

  Object.entries(metricsByName).forEach(([name, data]) => {
    const values = data.values.sort((a: number, b: number) => a - b)
    const total = values.length

    summary[name] = {
      count: total,
      average: Math.round((values.reduce((sum: number, val: number) => sum + val, 0) / total) * 100) / 100,
      median: values[Math.floor(total / 2)],
      p75: values[Math.floor(total * 0.75)],
      p90: values[Math.floor(total * 0.90)],
      p95: values[Math.floor(total * 0.95)],
      min: values[0],
      max: values[total - 1],
      ratings: {
        good: Math.round((data.ratings.good / total) * 100),
        needsImprovement: Math.round((data.ratings['needs-improvement'] / total) * 100),
        poor: Math.round((data.ratings.poor / total) * 100)
      }
    }
  })

  return summary
}