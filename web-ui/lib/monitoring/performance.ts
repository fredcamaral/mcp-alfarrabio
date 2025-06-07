import { logger } from '../logger'

// Web API type extensions
interface PerformanceEventTiming extends PerformanceEntry {
  processingStart: number
  startTime: number
}

interface LayoutShift extends PerformanceEntry {
  value: number
  hadRecentInput: boolean
}

interface PerformanceMemory {
  usedJSHeapSize: number
  totalJSHeapSize: number
  jsHeapSizeLimit: number
}

export interface PerformanceMetric {
  name: string
  value: number
  unit: 'ms' | 'bytes' | 'count'
  tags?: Record<string, string>
  timestamp: number
}

export interface PerformanceMonitor {
  startTimer(name: string): () => void
  recordMetric(metric: PerformanceMetric): void
  getMetrics(): PerformanceMetric[]
  clearMetrics(): void
}

class BrowserPerformanceMonitor implements PerformanceMonitor {
  private metrics: PerformanceMetric[] = []
  private maxMetrics = 1000

  startTimer(name: string): () => void {
    const startTime = performance.now()
    
    return () => {
      const duration = performance.now() - startTime
      this.recordMetric({
        name,
        value: duration,
        unit: 'ms',
        timestamp: Date.now()
      })
    }
  }

  recordMetric(metric: PerformanceMetric): void {
    this.metrics.push(metric)
    
    // Prevent memory leaks by limiting stored metrics
    if (this.metrics.length > this.maxMetrics) {
      this.metrics = this.metrics.slice(-this.maxMetrics)
    }

    // Log significant performance issues
    if (metric.unit === 'ms' && metric.value > 1000) {
      logger.warn('Slow operation detected', {
        component: 'PerformanceMonitor',
        metric: metric.name,
        duration: metric.value,
        tagsCount: metric.tags ? Object.keys(metric.tags).length : 0
      })
    }
  }

  getMetrics(): PerformanceMetric[] {
    return [...this.metrics]
  }

  clearMetrics(): void {
    this.metrics = []
  }

  // Get aggregated stats for a specific metric
  getStats(metricName: string) {
    const relevantMetrics = this.metrics.filter(m => m.name === metricName)
    
    if (relevantMetrics.length === 0) {
      return null
    }

    const values = relevantMetrics.map(m => m.value)
    const sum = values.reduce((a, b) => a + b, 0)
    const avg = sum / values.length
    const sorted = values.sort((a, b) => a - b)
    const p50 = sorted[Math.floor(sorted.length * 0.5)]
    const p90 = sorted[Math.floor(sorted.length * 0.9)]
    const p99 = sorted[Math.floor(sorted.length * 0.99)]

    return {
      count: values.length,
      sum,
      avg,
      min: Math.min(...values),
      max: Math.max(...values),
      p50,
      p90,
      p99
    }
  }
}

// Singleton instance
export const performanceMonitor = new BrowserPerformanceMonitor()

// Web Vitals monitoring
export function initWebVitals() {
  if (typeof window === 'undefined') return

  // First Contentful Paint (FCP)
  const fcpObserver = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      if (entry.name === 'first-contentful-paint') {
        performanceMonitor.recordMetric({
          name: 'web-vitals.fcp',
          value: entry.startTime,
          unit: 'ms',
          timestamp: Date.now()
        })
      }
    }
  })

  // Largest Contentful Paint (LCP)
  const lcpObserver = new PerformanceObserver((list) => {
    const entries = list.getEntries()
    const lastEntry = entries[entries.length - 1]
    
    performanceMonitor.recordMetric({
      name: 'web-vitals.lcp',
      value: lastEntry.startTime,
      unit: 'ms',
      timestamp: Date.now()
    })
  })

  // First Input Delay (FID)
  const fidObserver = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      const fidEntry = entry as PerformanceEventTiming
      if (fidEntry.processingStart) {
        performanceMonitor.recordMetric({
          name: 'web-vitals.fid',
          value: fidEntry.processingStart - fidEntry.startTime,
          unit: 'ms',
          timestamp: Date.now()
        })
      }
    }
  })

  // Cumulative Layout Shift (CLS)
  let clsValue = 0
  const clsObserver = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      const layoutShift = entry as LayoutShift
      if (!layoutShift.hadRecentInput && layoutShift.value) {
        clsValue += layoutShift.value
        performanceMonitor.recordMetric({
          name: 'web-vitals.cls',
          value: clsValue,
          unit: 'count',
          timestamp: Date.now()
        })
      }
    }
  })

  try {
    fcpObserver.observe({ type: 'paint', buffered: true })
    lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true })
    fidObserver.observe({ type: 'first-input', buffered: true })
    clsObserver.observe({ type: 'layout-shift', buffered: true })
  } catch {
    logger.warn('Web Vitals observers not supported', {
      component: 'PerformanceMonitor'
    })
  }
}

// React component render tracking
export function trackComponentRender(componentName: string) {
  const endTimer = performanceMonitor.startTimer(`component.render.${componentName}`)
  
  return () => {
    endTimer()
  }
}

// API call tracking
export function trackApiCall(endpoint: string, method: string) {
  const endTimer = performanceMonitor.startTimer(`api.${method}.${endpoint}`)
  
  return (success: boolean, statusCode?: number) => {
    endTimer()
    
    performanceMonitor.recordMetric({
      name: `api.${success ? 'success' : 'error'}`,
      value: 1,
      unit: 'count',
      tags: {
        endpoint,
        method,
        statusCode: statusCode?.toString() || 'unknown'
      },
      timestamp: Date.now()
    })
  }
}

// GraphQL operation tracking
export function trackGraphQLOperation(operationType: string, operationName: string) {
  const endTimer = performanceMonitor.startTimer(`graphql.${operationType}.${operationName}`)
  
  return (success: boolean, errorCount = 0) => {
    endTimer()
    
    if (!success || errorCount > 0) {
      performanceMonitor.recordMetric({
        name: 'graphql.errors',
        value: errorCount || 1,
        unit: 'count',
        tags: {
          operationType,
          operationName
        },
        timestamp: Date.now()
      })
    }
  }
}

// Memory usage tracking
export function trackMemoryUsage() {
  if ('memory' in performance) {
    const memory = (performance as Performance & { memory?: PerformanceMemory }).memory
    if (!memory) return
    
    performanceMonitor.recordMetric({
      name: 'memory.used',
      value: memory.usedJSHeapSize,
      unit: 'bytes',
      timestamp: Date.now()
    })
    
    performanceMonitor.recordMetric({
      name: 'memory.total',
      value: memory.totalJSHeapSize,
      unit: 'bytes',
      timestamp: Date.now()
    })
    
    performanceMonitor.recordMetric({
      name: 'memory.limit',
      value: memory.jsHeapSizeLimit,
      unit: 'bytes',
      timestamp: Date.now()
    })
  }
}

// Bundle size tracking
export function trackBundleSize() {
  if (typeof window !== 'undefined' && window.performance) {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    
    if (navigation) {
      performanceMonitor.recordMetric({
        name: 'bundle.transfer-size',
        value: navigation.transferSize || 0,
        unit: 'bytes',
        timestamp: Date.now()
      })
      
      performanceMonitor.recordMetric({
        name: 'bundle.encoded-size',
        value: navigation.encodedBodySize || 0,
        unit: 'bytes',
        timestamp: Date.now()
      })
      
      performanceMonitor.recordMetric({
        name: 'bundle.decoded-size',
        value: navigation.decodedBodySize || 0,
        unit: 'bytes',
        timestamp: Date.now()
      })
    }
  }
}

// Performance report generator
export function generatePerformanceReport() {
  const metrics = performanceMonitor.getMetrics()
  const uniqueMetrics = [...new Set(metrics.map(m => m.name))]
  
  const report = {
    timestamp: new Date().toISOString(),
    totalMetrics: metrics.length,
    metrics: {} as Record<string, unknown>
  }
  
  for (const metricName of uniqueMetrics) {
    const stats = performanceMonitor.getStats(metricName)
    if (stats) {
      report.metrics[metricName] = stats
    }
  }
  
  return report
}

// Auto-report performance metrics periodically
export function startPerformanceReporting(intervalMs = 60000) {
  const reportInterval = setInterval(() => {
    const report = generatePerformanceReport()
    
    logger.info('Performance report', {
      component: 'PerformanceMonitor',
      totalMetrics: report.totalMetrics
    })
    
    // Track memory usage
    trackMemoryUsage()
    
    // Clear old metrics to prevent memory leaks
    const metrics = performanceMonitor.getMetrics()
    const cutoffTime = Date.now() - (5 * 60 * 1000) // Keep last 5 minutes
    const recentMetrics = metrics.filter(m => m.timestamp > cutoffTime)
    
    performanceMonitor.clearMetrics()
    recentMetrics.forEach(m => performanceMonitor.recordMetric(m))
  }, intervalMs)
  
  return () => clearInterval(reportInterval)
}