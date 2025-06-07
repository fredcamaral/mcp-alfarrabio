import { logger } from '../logger'

export interface ErrorReport {
  id: string
  timestamp: number
  error: {
    message: string
    stack?: string
    name: string
    code?: string
  }
  context: {
    url: string
    userAgent: string
    component?: string
    action?: string
    userId?: string
    sessionId?: string
  }
  metadata?: Record<string, unknown>
  severity: 'low' | 'medium' | 'high' | 'critical'
}

export interface ErrorReporter {
  report(error: Error, context?: Partial<ErrorReport['context']>, metadata?: Record<string, unknown>): void
  reportWarning(message: string, context?: Partial<ErrorReport['context']>, metadata?: Record<string, unknown>): void
  getReports(): ErrorReport[]
  clearReports(): void
}

class BrowserErrorReporter implements ErrorReporter {
  private reports: ErrorReport[] = []
  private maxReports = 100
  private reportQueue: ErrorReport[] = []
  private isOnline = true

  constructor() {
    if (typeof window !== 'undefined') {
      // Listen for online/offline events
      window.addEventListener('online', () => {
        this.isOnline = true
        this.flushQueue()
      })
      
      window.addEventListener('offline', () => {
        this.isOnline = false
      })

      // Global error handler
      window.addEventListener('error', (event) => {
        this.report(
          new Error(event.message),
          {
            component: 'window',
            action: 'unhandled-error'
          },
          {
            filename: event.filename,
            lineno: event.lineno,
            colno: event.colno
          }
        )
      })

      // Unhandled promise rejection handler
      window.addEventListener('unhandledrejection', (event) => {
        this.report(
          new Error(event.reason?.message || event.reason || 'Unhandled Promise Rejection'),
          {
            component: 'window',
            action: 'unhandled-rejection'
          },
          {
            reason: event.reason,
            promise: event.promise
          }
        )
      })
    }
  }

  report(error: Error, context?: Partial<ErrorReport['context']>, metadata?: Record<string, unknown>): void {
    const report: ErrorReport = {
      id: this.generateId(),
      timestamp: Date.now(),
      error: {
        message: error.message,
        stack: error.stack,
        name: error.name,
        code: (error as Error & { code?: string }).code
      },
      context: {
        url: typeof window !== 'undefined' ? window.location.href : '',
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
        ...context
      },
      metadata,
      severity: this.determineSeverity(error, context)
    }

    this.addReport(report)
    this.sendReport(report)

    // Log to console in development
    if (process.env.NODE_ENV === 'development') {
      logger.error('Error reported', error, {
        component: 'ErrorReporter',
        url: report.context.url,
        userAgent: report.context.userAgent
      })
    }
  }

  reportWarning(message: string, context?: Partial<ErrorReport['context']>, metadata?: Record<string, unknown>): void {
    const report: ErrorReport = {
      id: this.generateId(),
      timestamp: Date.now(),
      error: {
        message,
        name: 'Warning'
      },
      context: {
        url: typeof window !== 'undefined' ? window.location.href : '',
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
        ...context
      },
      metadata,
      severity: 'low'
    }

    this.addReport(report)
    
    // Warnings are logged but not sent to backend
    logger.warn('Warning reported', {
      component: 'ErrorReporter',
      message,
      url: report.context.url,
      userAgent: report.context.userAgent
    })
  }

  getReports(): ErrorReport[] {
    return [...this.reports]
  }

  clearReports(): void {
    this.reports = []
  }

  private generateId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }

  private determineSeverity(error: Error, context?: Partial<ErrorReport['context']>): ErrorReport['severity'] {
    // Critical: Authentication/Security errors
    if (error.message.toLowerCase().includes('auth') || 
        error.message.toLowerCase().includes('security') ||
        error.message.toLowerCase().includes('forbidden')) {
      return 'critical'
    }

    // High: Data loss or corruption
    if (error.message.toLowerCase().includes('data') ||
        error.message.toLowerCase().includes('corrupt') ||
        error.message.toLowerCase().includes('failed to save')) {
      return 'high'
    }

    // Medium: Feature failures
    if (context?.component && 
        (error.message.toLowerCase().includes('failed') ||
         error.message.toLowerCase().includes('error'))) {
      return 'medium'
    }

    // Low: Everything else
    return 'low'
  }

  private addReport(report: ErrorReport): void {
    this.reports.push(report)
    
    // Prevent memory leaks
    if (this.reports.length > this.maxReports) {
      this.reports = this.reports.slice(-this.maxReports)
    }
  }

  private async sendReport(report: ErrorReport): Promise<void> {
    // Don't send reports in development
    if (process.env.NODE_ENV === 'development') {
      return
    }

    // Queue report if offline
    if (!this.isOnline) {
      this.reportQueue.push(report)
      return
    }

    try {
      // In production, this would send to an error tracking service
      // For now, we'll just log it
      const response = await fetch('/api/errors', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(report)
      })

      if (!response.ok) {
        throw new Error(`Failed to send error report: ${response.status}`)
      }
    } catch (error) {
      // Failed to send, add to queue
      this.reportQueue.push(report)
      
      logger.warn('Failed to send error report', {
        component: 'ErrorReporter',
        error: error instanceof Error ? error.message : 'Unknown error'
      })
    }
  }

  private async flushQueue(): Promise<void> {
    if (this.reportQueue.length === 0) return

    const queue = [...this.reportQueue]
    this.reportQueue = []

    for (const report of queue) {
      await this.sendReport(report)
    }
  }
}

// Singleton instance
export const errorReporter = new BrowserErrorReporter()

// React Error Boundary helper
export function logErrorToService(error: Error, errorInfo: { componentStack?: string }) {
  errorReporter.report(
    error,
    {
      component: errorInfo.componentStack ? 'React' : 'Unknown',
      action: 'error-boundary'
    },
    {
      componentStack: errorInfo.componentStack
    }
  )
}

// GraphQL error handler
interface GraphQLError {
  message: string
  path?: string[]
  extensions?: Record<string, unknown>
  locations?: Array<{ line: number; column: number }>
}

export function handleGraphQLErrors(errors: GraphQLError[]) {
  errors.forEach(error => {
    errorReporter.report(
      new Error(error.message),
      {
        component: 'GraphQL',
        action: error.path?.join('.') || 'unknown'
      },
      {
        extensions: error.extensions,
        locations: error.locations,
        path: error.path
      }
    )
  })
}

// API error handler
export function handleApiError(endpoint: string, method: string, status: number, error: unknown) {
  const errorMessage = (error instanceof Error ? error.message : String(error)) || `API Error: ${status}`
  
  errorReporter.report(
    new Error(errorMessage),
    {
      component: 'API',
      action: `${method} ${endpoint}`
    },
    {
      status,
      endpoint,
      method,
      response: error
    }
  )
}

// Form validation error handler
export function handleValidationError(formName: string, errors: Record<string, unknown>) {
  errorReporter.reportWarning(
    `Form validation failed: ${formName}`,
    {
      component: 'Form',
      action: formName
    },
    {
      validationErrors: errors
    }
  )
}

// WebSocket error handler
export function handleWebSocketError(error: Error | Event, url: string) {
  errorReporter.report(
    error instanceof Error ? error : new Error('WebSocket error'),
    {
      component: 'WebSocket',
      action: 'connection'
    },
    {
      url,
      readyState: typeof error === 'object' && 'target' in error ? 
        (error.target as WebSocket).readyState : undefined
    }
  )
}

// Generate error report summary
export function generateErrorSummary() {
  const reports = errorReporter.getReports()
  const summary = {
    timestamp: new Date().toISOString(),
    totalErrors: reports.length,
    bySeverity: {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0
    } as Record<string, number>,
    byComponent: {} as Record<string, number>,
    recentErrors: [] as Array<{
      id: string;
      timestamp: string;
      message: string;
      severity: string;
      component?: string;
    }>
  }

  reports.forEach(report => {
    // Count by severity
    summary.bySeverity[report.severity]++
    
    // Count by component
    const component = report.context.component || 'unknown'
    summary.byComponent[component] = (summary.byComponent[component] || 0) + 1
  })

  // Get recent errors (last 10)
  summary.recentErrors = reports
    .slice(-10)
    .map(r => ({
      id: r.id,
      timestamp: new Date(r.timestamp).toISOString(),
      message: r.error.message,
      severity: r.severity,
      component: r.context.component
    }))

  return summary
}