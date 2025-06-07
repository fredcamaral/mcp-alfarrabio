import { onCLS, onFCP, onFID, onLCP, onTTFB, Metric } from 'web-vitals'

type PerformanceMetric = {
  id: string
  name: string
  value: number
  delta: number
  rating: 'good' | 'needs-improvement' | 'poor'
  timestamp: number
  url: string
  userAgent: string
}

class PerformanceMonitor {
  private metrics: PerformanceMetric[] = []
  private reportingEndpoint: string | null = null
  private batchSize = 10
  private batchTimeout = 5000
  private pendingMetrics: PerformanceMetric[] = []
  private batchTimer: NodeJS.Timeout | null = null

  constructor(reportingEndpoint?: string) {
    this.reportingEndpoint = reportingEndpoint || null
    this.initializeVitals()
  }

  private initializeVitals(): void {
    if (typeof window === 'undefined') return

    // Core Web Vitals
    onCLS(this.handleMetric.bind(this))
    onFCP(this.handleMetric.bind(this))
    onFID(this.handleMetric.bind(this))
    onLCP(this.handleMetric.bind(this))
    onTTFB(this.handleMetric.bind(this))

    // Listen for page visibility changes to send final metrics
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        this.flushMetrics()
      }
    })

    // Listen for page unload
    window.addEventListener('beforeunload', () => {
      this.flushMetrics()
    })
  }

  private handleMetric(metric: Metric): void {
    const performanceMetric: PerformanceMetric = {
      id: metric.id,
      name: metric.name,
      value: metric.value,
      delta: metric.delta,
      rating: metric.rating,
      timestamp: Date.now(),
      url: window.location.href,
      userAgent: navigator.userAgent,
    }

    this.metrics.push(performanceMetric)
    this.pendingMetrics.push(performanceMetric)

    // Console logging for development
    if (process.env.NODE_ENV === 'development') {
      console.log(`🚀 Performance Metric: ${metric.name}`, {
        value: metric.value,
        rating: metric.rating,
        delta: metric.delta,
      })
    }

    // Batch reporting
    this.scheduleBatchReport()

    // Real-time analytics for critical metrics
    if (metric.rating === 'poor') {
      this.reportCriticalMetric(performanceMetric)
    }
  }

  private scheduleBatchReport(): void {
    if (this.batchTimer) {
      clearTimeout(this.batchTimer)
    }

    if (this.pendingMetrics.length >= this.batchSize) {
      this.flushMetrics()
    } else {
      this.batchTimer = setTimeout(() => {
        this.flushMetrics()
      }, this.batchTimeout)
    }
  }

  private async flushMetrics(): Promise<void> {
    if (this.pendingMetrics.length === 0) return

    const metricsToSend = [...this.pendingMetrics]
    this.pendingMetrics = []

    if (this.batchTimer) {
      clearTimeout(this.batchTimer)
      this.batchTimer = null
    }

    try {
      await this.sendMetrics(metricsToSend)
    } catch (error) {
      console.error('Failed to send performance metrics:', error)
      // Add metrics back to pending queue for retry
      this.pendingMetrics.unshift(...metricsToSend)
    }
  }

  private async sendMetrics(metrics: PerformanceMetric[]): Promise<void> {
    if (!this.reportingEndpoint) return

    const payload = {
      metrics,
      sessionId: this.getSessionId(),
      timestamp: Date.now(),
    }

    // Use sendBeacon for reliability during page unload
    if (navigator.sendBeacon && document.visibilityState === 'hidden') {
      navigator.sendBeacon(
        this.reportingEndpoint,
        JSON.stringify(payload)
      )
    } else {
      await fetch(this.reportingEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
        keepalive: true,
      })
    }
  }

  private async reportCriticalMetric(metric: PerformanceMetric): Promise<void> {
    if (!this.reportingEndpoint) return

    try {
      await fetch(`${this.reportingEndpoint}/critical`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          metric,
          sessionId: this.getSessionId(),
          timestamp: Date.now(),
          critical: true,
        }),
      })
    } catch (error) {
      console.error('Failed to report critical metric:', error)
    }
  }

  private getSessionId(): string {
    let sessionId = sessionStorage.getItem('performance-session-id')
    if (!sessionId) {
      sessionId = Math.random().toString(36).substring(2, 15)
      sessionStorage.setItem('performance-session-id', sessionId)
    }
    return sessionId
  }

  public getMetrics(): PerformanceMetric[] {
    return [...this.metrics]
  }

  public getMetricsByName(name: string): PerformanceMetric[] {
    return this.metrics.filter(metric => metric.name === name)
  }

  public getAverageMetric(name: string): number | null {
    const metrics = this.getMetricsByName(name)
    if (metrics.length === 0) return null

    const sum = metrics.reduce((acc, metric) => acc + metric.value, 0)
    return sum / metrics.length
  }

  public getWorstMetric(name: string): PerformanceMetric | null {
    const metrics = this.getMetricsByName(name)
    if (metrics.length === 0) return null

    return metrics.reduce((worst, current) => {
      if (current.rating === 'poor' && worst.rating !== 'poor') return current
      if (current.rating === 'needs-improvement' && worst.rating === 'good') return current
      if (current.value > worst.value) return current
      return worst
    })
  }

  public generateReport(): {
    summary: Record<string, { average: number; worst: number; rating: string }>
    details: PerformanceMetric[]
  } {
    const metricNames = [...new Set(this.metrics.map(m => m.name))]
    const summary: Record<string, { average: number; worst: number; rating: string }> = {}

    metricNames.forEach(name => {
      const average = this.getAverageMetric(name) || 0
      const worst = this.getWorstMetric(name)
      summary[name] = {
        average: Math.round(average * 100) / 100,
        worst: worst?.value || 0,
        rating: worst?.rating || 'good',
      }
    })

    return {
      summary,
      details: this.metrics,
    }
  }
}

// Singleton instance
let performanceMonitor: PerformanceMonitor | null = null

export function initializePerformanceMonitoring(reportingEndpoint?: string): PerformanceMonitor {
  if (!performanceMonitor) {
    performanceMonitor = new PerformanceMonitor(reportingEndpoint)
  }
  return performanceMonitor
}

export function getPerformanceMonitor(): PerformanceMonitor | null {
  return performanceMonitor
}

// Custom performance markers
export function markPerformance(name: string, detail?: unknown): void {
  if (typeof window === 'undefined') return

  performance.mark(name, { detail })

  if (process.env.NODE_ENV === 'development') {
    console.log(`📊 Performance Mark: ${name}`, detail)
  }
}

export function measurePerformance(
  name: string,
  startMark: string,
  endMark?: string
): PerformanceMeasure | null {
  if (typeof window === 'undefined') return null

  try {
    const measure = performance.measure(name, startMark, endMark)
    
    if (process.env.NODE_ENV === 'development') {
      console.log(`⏱️ Performance Measure: ${name}`, {
        duration: measure.duration,
        start: measure.startTime,
      })
    }

    return measure
  } catch (error) {
    console.warn(`Failed to measure performance for ${name}:`, error)
    return null
  }
}

// Resource timing analysis
export function analyzeResourceTiming(): {
  slowResources: PerformanceResourceTiming[]
  totalResources: number
  averageLoadTime: number
} {
  if (typeof window === 'undefined') {
    return { slowResources: [], totalResources: 0, averageLoadTime: 0 }
  }

  const resources = performance.getEntriesByType('resource') as PerformanceResourceTiming[]
  const slowThreshold = 1000 // 1 second

  const slowResources = resources.filter(
    resource => resource.duration > slowThreshold
  )

  const totalLoadTime = resources.reduce((sum, resource) => sum + resource.duration, 0)
  const averageLoadTime = resources.length > 0 ? totalLoadTime / resources.length : 0

  return {
    slowResources,
    totalResources: resources.length,
    averageLoadTime: Math.round(averageLoadTime * 100) / 100,
  }
}

export { PerformanceMonitor }
export type { PerformanceMetric }