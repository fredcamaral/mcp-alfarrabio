'use client'

import { Provider } from 'react-redux'
import { ApolloProvider } from '@apollo/client'
import { store } from '@/store/store'
import { apolloClient } from '@/lib/apollo-client'
import { CSRFProvider } from './CSRFProvider'
import { ThemeProvider } from './ThemeProvider'
import { MonitoringProvider } from './MonitoringProvider'
import { PerformanceProvider, PerformanceIndicator } from './PerformanceProvider'
import { WebSocketProvider } from './WebSocketProvider'
import { ErrorBoundary, AsyncErrorBoundary } from '@/components/error/ErrorBoundary'
import { MonitoredErrorBoundary } from '@/components/error/MonitoredErrorBoundary'
import { logger } from '@/lib/logger'

interface ProvidersProps {
  children: React.ReactNode
}

export function Providers({ children }: ProvidersProps) {
  return (
    // Outermost error boundary for critical system errors
    <AsyncErrorBoundary
      enableRetry={true}
      enableLogging={true}
      showErrorDetails={process.env.NODE_ENV === 'development'}
      onError={(error, errorInfo) => {
        // Log critical provider errors
        logger.error('Critical application error in providers:', { error, errorInfo })
      }}
    >
      <Provider store={store}>
        {/* Error boundary for Redux state errors */}
        <ErrorBoundary
          enableRetry={true}
          onError={(error) => {
            logger.error('Redux Provider error:', error)
          }}
        >
          <ApolloProvider client={apolloClient}>
            {/* Error boundary for GraphQL/Apollo errors */}
            <ErrorBoundary
              enableRetry={true}
              onError={(error) => {
                logger.error('Apollo Provider error:', error)
              }}
            >
              <MonitoringProvider>
                <PerformanceProvider>
                  {/* Main application error boundary with monitoring */}
                  <MonitoredErrorBoundary>
                    <CSRFProvider>
                      <ThemeProvider defaultTheme="dark" storageKey="mcp-memory-theme">
                        <WebSocketProvider>
                          {/* Final error boundary for application content */}
                          <ErrorBoundary
                            enableRetry={true}
                            enableLogging={true}
                            showErrorDetails={process.env.NODE_ENV === 'development'}
                          >
                            {children}
                            <PerformanceIndicator />
                          </ErrorBoundary>
                        </WebSocketProvider>
                      </ThemeProvider>
                    </CSRFProvider>
                  </MonitoredErrorBoundary>
                </PerformanceProvider>
              </MonitoringProvider>
            </ErrorBoundary>
          </ApolloProvider>
        </ErrorBoundary>
      </Provider>
    </AsyncErrorBoundary>
  )
}