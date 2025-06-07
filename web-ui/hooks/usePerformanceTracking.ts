import { useEffect, useCallback } from 'react'
import { useMonitoring } from '@/providers/MonitoringProvider'
import { 
  trackApiCall, 
  trackGraphQLOperation,
  trackComponentRender
} from '@/lib/monitoring/performance'

// Hook for tracking component performance
export function usePerformanceTracking(componentName: string) {
  const { performanceMonitor } = useMonitoring()
  
  useEffect(() => {
    const endTracking = trackComponentRender(componentName)
    return endTracking
  }, [componentName])
  
  const trackAction = useCallback((actionName: string) => {
    const endTimer = performanceMonitor.startTimer(`component.action.${componentName}.${actionName}`)
    return endTimer
  }, [componentName, performanceMonitor])
  
  return { trackAction }
}

// Hook for tracking API calls
export function useApiTracking() {
  const trackApi = useCallback((endpoint: string, method: string) => {
    const endTracking = trackApiCall(endpoint, method)
    
    return {
      success: (statusCode?: number) => endTracking(true, statusCode),
      error: (statusCode?: number) => endTracking(false, statusCode)
    }
  }, [])
  
  return { trackApi }
}

// Hook for tracking GraphQL operations
export function useGraphQLTracking() {
  const trackQuery = useCallback((operationName: string) => {
    const endTracking = trackGraphQLOperation('query', operationName)
    
    return {
      success: () => endTracking(true, 0),
      error: (errorCount: number) => endTracking(false, errorCount)
    }
  }, [])
  
  const trackMutation = useCallback((operationName: string) => {
    const endTracking = trackGraphQLOperation('mutation', operationName)
    
    return {
      success: () => endTracking(true, 0),
      error: (errorCount: number) => endTracking(false, errorCount)
    }
  }, [])
  
  const trackSubscription = useCallback((operationName: string) => {
    const endTracking = trackGraphQLOperation('subscription', operationName)
    
    return {
      success: () => endTracking(true, 0),
      error: (errorCount: number) => endTracking(false, errorCount)
    }
  }, [])
  
  return { trackQuery, trackMutation, trackSubscription }
}

// Hook for tracking user interactions
export function useInteractionTracking(componentName: string) {
  const { performanceMonitor } = useMonitoring()
  
  const trackClick = useCallback((elementName: string) => {
    performanceMonitor.recordMetric({
      name: 'user.interaction.click',
      value: 1,
      unit: 'count',
      tags: {
        component: componentName,
        element: elementName
      },
      timestamp: Date.now()
    })
  }, [componentName, performanceMonitor])
  
  const trackFormSubmit = useCallback((formName: string, success: boolean) => {
    performanceMonitor.recordMetric({
      name: `form.submit.${success ? 'success' : 'error'}`,
      value: 1,
      unit: 'count',
      tags: {
        component: componentName,
        form: formName
      },
      timestamp: Date.now()
    })
  }, [componentName, performanceMonitor])
  
  const trackSearch = useCallback((query: string, resultCount: number) => {
    performanceMonitor.recordMetric({
      name: 'search.performed',
      value: resultCount,
      unit: 'count',
      tags: {
        component: componentName,
        queryLength: query.length.toString(),
        hasResults: (resultCount > 0).toString()
      },
      timestamp: Date.now()
    })
  }, [componentName, performanceMonitor])
  
  return { trackClick, trackFormSubmit, trackSearch }
}