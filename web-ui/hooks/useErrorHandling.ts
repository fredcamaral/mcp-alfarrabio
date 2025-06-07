import { useCallback } from 'react'
import { useMonitoring } from '@/providers/MonitoringProvider'
import { 
  handleApiError,
  handleGraphQLErrors,
  handleValidationError,
  handleWebSocketError
} from '@/lib/monitoring/error-reporter'
import { toast } from '@/components/ui/use-toast'

export interface ErrorHandlingOptions {
  showToast?: boolean
  componentName?: string
  action?: string
  metadata?: Record<string, any>
}

export function useErrorHandling() {
  const { errorReporter } = useMonitoring()
  
  const handleError = useCallback((
    error: Error,
    options: ErrorHandlingOptions = {}
  ) => {
    const { showToast = true, componentName, action, metadata } = options
    
    // Report error
    errorReporter.report(
      error,
      {
        component: componentName,
        action
      },
      metadata
    )
    
    // Show toast notification if requested
    if (showToast) {
      toast({
        title: 'Error',
        description: error.message || 'An unexpected error occurred',
        variant: 'destructive'
      })
    }
  }, [errorReporter])
  
  const handleApiErrorWithToast = useCallback((
    endpoint: string,
    method: string,
    status: number,
    error: any
  ) => {
    handleApiError(endpoint, method, status, error)
    
    let message = 'An error occurred while processing your request'
    if (status === 404) {
      message = 'The requested resource was not found'
    } else if (status === 403) {
      message = 'You do not have permission to perform this action'
    } else if (status === 401) {
      message = 'Please sign in to continue'
    } else if (status >= 500) {
      message = 'A server error occurred. Please try again later'
    }
    
    toast({
      title: 'API Error',
      description: message,
      variant: 'destructive'
    })
  }, [])
  
  const handleGraphQLErrorsWithToast = useCallback((errors: any[]) => {
    handleGraphQLErrors(errors)
    
    const userMessage = errors[0]?.message || 'An error occurred with your request'
    
    toast({
      title: 'Request Failed',
      description: userMessage,
      variant: 'destructive'
    })
  }, [])
  
  const handleFormValidationError = useCallback((
    formName: string,
    errors: Record<string, any>,
    showToast = true
  ) => {
    handleValidationError(formName, errors)
    
    if (showToast) {
      const errorCount = Object.keys(errors).length
      toast({
        title: 'Validation Error',
        description: `Please fix ${errorCount} ${errorCount === 1 ? 'error' : 'errors'} in the form`,
        variant: 'destructive'
      })
    }
  }, [])
  
  const handleWebSocketErrorWithRetry = useCallback((
    error: Error | Event,
    url: string,
    onRetry?: () => void
  ) => {
    handleWebSocketError(error, url)
    
    if (onRetry) {
      toast({
        title: 'Connection Error',
        description: 'Lost connection to server. Click to retry.',
        variant: 'destructive',
        onClick: onRetry
      } as any)
    } else {
      toast({
        title: 'Connection Error',
        description: 'Lost connection to server. Retrying...',
        variant: 'destructive'
      })
    }
  }, [])
  
  const wrapAsync = useCallback(<T extends any[], R>(
    fn: (...args: T) => Promise<R>,
    options: ErrorHandlingOptions = {}
  ) => {
    return async (...args: T): Promise<R | undefined> => {
      try {
        return await fn(...args)
      } catch (error) {
        handleError(
          error instanceof Error ? error : new Error('Unknown error'),
          options
        )
        return undefined
      }
    }
  }, [handleError])
  
  return {
    handleError,
    handleApiError: handleApiErrorWithToast,
    handleGraphQLErrors: handleGraphQLErrorsWithToast,
    handleFormValidationError,
    handleWebSocketError: handleWebSocketErrorWithRetry,
    wrapAsync
  }
}