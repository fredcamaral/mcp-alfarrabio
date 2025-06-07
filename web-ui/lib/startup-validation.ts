/**
 * Startup validation script
 * Runs when the application starts to ensure everything is configured correctly
 */

import { env, config } from './env-validation'
import { logger } from './logger'

export async function validateStartup(): Promise<void> {
  const startTime = performance.now()
  
  logger.info('Starting application validation', {
    component: 'StartupValidation',
    environment: config.environment.name,
    nodeEnv: process.env.NODE_ENV,
  })

  try {
    // 1. Validate critical environment variables
    validateCriticalEnvVars()
    
    // 2. Check API connectivity (only in production)
    if (config.environment.isProduction) {
      await checkApiConnectivity()
    }
    
    // 3. Validate feature compatibility
    validateFeatureCompatibility()
    
    // 4. Log startup configuration
    logStartupConfiguration()
    
    const duration = performance.now() - startTime
    logger.info(`Startup validation completed successfully in ${duration.toFixed(2)}ms`, {
      component: 'StartupValidation',
      duration: `${duration.toFixed(2)}ms`,
    })
  } catch (error) {
    logger.error('Startup validation failed', error, {
      component: 'StartupValidation',
    })
    
    if (config.environment.isProduction && typeof window === 'undefined') {
      // In production on server-side, throw to stop the application
      throw new Error('Critical startup validation failed in production')
    }
  }
}

function validateCriticalEnvVars(): void {
  const criticalVars = [
    'NEXT_PUBLIC_API_URL',
    'NEXT_PUBLIC_GRAPHQL_URL',
    'NEXT_PUBLIC_WS_URL',
  ]
  
  const missing = criticalVars.filter(varName => !env[varName as keyof typeof env])
  
  if (missing.length > 0) {
    throw new Error(`Missing critical environment variables: ${missing.join(', ')}`)
  }
  
  // Validate URL formats
  try {
    new URL(env.NEXT_PUBLIC_API_URL)
    new URL(env.NEXT_PUBLIC_GRAPHQL_URL)
    
    // WebSocket URL validation
    const wsUrl = new URL(env.NEXT_PUBLIC_WS_URL)
    if (!['ws:', 'wss:'].includes(wsUrl.protocol)) {
      throw new Error('WebSocket URL must use ws:// or wss:// protocol')
    }
  } catch (error) {
    throw new Error(`Invalid URL format in environment variables: ${error}`)
  }
}

async function checkApiConnectivity(): Promise<void> {
  const timeout = 5000 // 5 seconds
  
  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)
    
    const response = await fetch(`${env.NEXT_PUBLIC_API_URL}/health`, {
      signal: controller.signal,
      method: 'GET',
    })
    
    clearTimeout(timeoutId)
    
    if (!response.ok) {
      logger.warn('API health check returned non-200 status', {
        component: 'StartupValidation',
        status: response.status,
        statusText: response.statusText,
      })
    }
  } catch (error) {
    // Log warning but don't fail startup
    logger.warn('API connectivity check failed', {
      component: 'StartupValidation',
      error: error instanceof Error ? error.message : 'Unknown error',
      apiUrl: env.NEXT_PUBLIC_API_URL,
    })
  }
}

function validateFeatureCompatibility(): void {
  // Check WebSocket compatibility with subscriptions
  if (config.features.graphqlSubscriptions && !config.features.websocket) {
    logger.warn('GraphQL subscriptions enabled but WebSocket is disabled', {
      component: 'StartupValidation',
      recommendation: 'Enable WebSocket for subscriptions to work',
    })
  }
  
  // Check mock data in production
  if (config.environment.isProduction && config.features.mockData) {
    logger.warn('Mock data is enabled in production environment', {
      component: 'StartupValidation',
      recommendation: 'Disable mock data for production',
    })
  }
  
  // Check HTTPS in production
  if (config.environment.isProduction) {
    const apiUrl = new URL(env.NEXT_PUBLIC_API_URL)
    if (apiUrl.protocol === 'http:' && apiUrl.hostname !== 'localhost') {
      logger.warn('Using insecure HTTP protocol in production', {
        component: 'StartupValidation',
        url: env.NEXT_PUBLIC_API_URL,
        recommendation: 'Use HTTPS for production deployments',
      })
    }
  }
}

function logStartupConfiguration(): void {
  logger.info('Application configuration', {
    component: 'StartupValidation',
    environment: config.environment.name,
    baseUrl: config.api.baseUrl,
    mockData: config.features.mockData,
    websocket: config.features.websocket,
    logLevel: config.monitoring.logLevel,
    hasLoggingEndpoint: !!config.monitoring.loggingEndpoint,
    hasMetricsEndpoint: !!config.monitoring.metricsEndpoint,
  })
}

// Run validation on module load (server-side only)
if (typeof window === 'undefined') {
  validateStartup().catch(error => {
    logger.error('Startup validation failed:', error)
  })
}