'use client'

import { createContext, useContext, useEffect, ReactNode } from 'react'
import { 
  performanceMonitor, 
  initWebVitals, 
  startPerformanceReporting,
  trackComponentRender,
  generatePerformanceReport
} from '@/lib/monitoring/performance'
import { 
  errorReporter, 
  generateErrorSummary,
  logErrorToService
} from '@/lib/monitoring/error-reporter'
import { logger } from '@/lib/logger'
import { config } from '@/lib/env-validation'

interface MonitoringContextValue {
  performanceMonitor: typeof performanceMonitor
  errorReporter: typeof errorReporter
  trackComponentRender: typeof trackComponentRender
  logError: typeof logErrorToService
}

const MonitoringContext = createContext<MonitoringContextValue | null>(null)

export function useMonitoring() {
  const context = useContext(MonitoringContext)
  if (!context) {
    throw new Error('useMonitoring must be used within MonitoringProvider')
  }
  return context
}

interface MonitoringProviderProps {
  children: ReactNode
  enablePerformanceReporting?: boolean
  reportingInterval?: number
}

export function MonitoringProvider({ 
  children, 
  enablePerformanceReporting = true,
  reportingInterval = 60000 // 1 minute
}: MonitoringProviderProps) {
  useEffect(() => {
    // Initialize Web Vitals monitoring
    initWebVitals()
    
    // Start performance reporting if enabled
    let stopReporting: (() => void) | null = null
    
    if (enablePerformanceReporting && config.environment.isProduction) {
      stopReporting = startPerformanceReporting(reportingInterval)
    }
    
    // Log initial monitoring setup
    logger.info('Monitoring initialized', {
      component: 'MonitoringProvider',
      performanceReporting: enablePerformanceReporting,
      environment: config.environment.name
    })
    
    // Set up periodic error summary logging (every 5 minutes)
    const errorSummaryInterval = setInterval(() => {
      const errorSummary = generateErrorSummary()
      const perfReport = generatePerformanceReport()
      
      if (errorSummary.totalErrors > 0 || config.environment.isDevelopment) {
        logger.info('Monitoring summary', {
          component: 'MonitoringProvider',
          totalErrors: errorSummary.totalErrors,
          totalMetrics: perfReport.totalMetrics
        })
      }
    }, 5 * 60 * 1000)
    
    return () => {
      if (stopReporting) {
        stopReporting()
      }
      clearInterval(errorSummaryInterval)
    }
  }, [enablePerformanceReporting, reportingInterval])
  
  const value: MonitoringContextValue = {
    performanceMonitor,
    errorReporter,
    trackComponentRender,
    logError: logErrorToService
  }
  
  return (
    <MonitoringContext.Provider value={value}>
      {children}
    </MonitoringContext.Provider>
  )
}

// Hook for tracking component performance
export function useComponentPerformance(componentName: string) {
  useEffect(() => {
    const endTracking = trackComponentRender(componentName)
    return endTracking
  }, [componentName])
}