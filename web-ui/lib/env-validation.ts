/**
 * Environment variable validation and configuration
 * Ensures all required environment variables are set before the app starts
 */

import { z } from 'zod'
import { logger } from './logger'

// Define the environment schema
const envSchema = z.object({
  // Public environment variables (available on client)
  NEXT_PUBLIC_API_URL: z.string().url().default('http://localhost:9080'),
  NEXT_PUBLIC_GRAPHQL_URL: z.string().url().default('http://localhost:9080/graphql'),
  NEXT_PUBLIC_WS_URL: z.string().url().default('ws://localhost:9080/ws'),
  NEXT_PUBLIC_ENVIRONMENT: z.enum(['development', 'staging', 'production']).default('development'),
  NEXT_PUBLIC_LOG_LEVEL: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
  
  // Optional monitoring endpoints
  NEXT_PUBLIC_LOGGING_ENDPOINT: z.string().url().optional(),
  NEXT_PUBLIC_METRICS_ENDPOINT: z.string().url().optional(),
  
  // Server-side only variables
  NODE_ENV: z.enum(['development', 'test', 'production']).default('development'),
  
  // Repository configuration
  NEXT_PUBLIC_DEFAULT_REPOSITORY: z.string().default('github.com/lerianstudio/lerian-mcp-memory'),
  NEXT_PUBLIC_GITHUB_REPO_URL: z.string().url().default('https://github.com/lerianstudio/lerian-mcp-memory'),
  NEXT_PUBLIC_GITHUB_ISSUES_URL: z.string().url().default('https://github.com/lerianstudio/lerian-mcp-memory/issues'),
  
  // Optional feature flags
  NEXT_PUBLIC_ENABLE_WEBSOCKET: z.string().transform(val => val === 'true').default('true'),
  NEXT_PUBLIC_ENABLE_GRAPHQL_SUBSCRIPTIONS: z.string().transform(val => val === 'true').default('false'),
  NEXT_PUBLIC_ENABLE_RATE_LIMITING: z.string().transform(val => val === 'true').default('true'),
  NEXT_PUBLIC_ENABLE_MOCK_DATA: z.string().transform(val => val === 'true').default('false'),
})

// Type for the validated environment
export type ValidatedEnv = z.infer<typeof envSchema>

// Validate environment variables
function validateEnv(): ValidatedEnv {
  try {
    const env = envSchema.parse(process.env)
    
    // Log successful validation in development
    if (process.env.NODE_ENV === 'development') {
      logger.info('Environment validation successful', {
        component: 'EnvValidation',
        environment: env.NEXT_PUBLIC_ENVIRONMENT,
      })
    }
    
    return env
  } catch (error) {
    if (error instanceof z.ZodError) {
      const missingVars = error.errors.map(err => err.path.join('.')).join(', ')
      const errorMessage = `Missing or invalid environment variables: ${missingVars}`
      
      logger.error('Environment validation failed', new Error(errorMessage), {
        component: 'EnvValidation',
        errorCount: error.errors.length,
      })
      
      // In production, throw to prevent startup
      if (process.env.NODE_ENV === 'production') {
        throw new Error(errorMessage)
      }
      
      // In development, log warning and continue with defaults
      logger.warn('Using default values for missing environment variables', {
        component: 'EnvValidation',
        missingVars,
      })
    }
    
    // Return parsed environment with defaults
    return envSchema.parse({})
  }
}

// Export validated environment
export const env = validateEnv()

// Helper to check if we're in production
export const isProduction = () => env.NODE_ENV === 'production'

// Helper to check if we're in development
export const isDevelopment = () => env.NODE_ENV === 'development'

// Helper to get the appropriate API URL based on environment
export const getApiUrl = () => {
  if (typeof window === 'undefined') {
    // Server-side: use internal URL if available
    return process.env.INTERNAL_API_URL || env.NEXT_PUBLIC_API_URL
  }
  return env.NEXT_PUBLIC_API_URL
}

// Helper to get WebSocket URL with proper protocol
export const getWebSocketUrl = () => {
  const wsUrl = env.NEXT_PUBLIC_WS_URL
  
  // In production, ensure wss:// for secure connections
  if (isProduction() && wsUrl.startsWith('ws://')) {
    return wsUrl.replace('ws://', 'wss://')
  }
  
  return wsUrl
}

// Export configuration object
export const config = {
  api: {
    baseUrl: env.NEXT_PUBLIC_API_URL,
    graphqlUrl: env.NEXT_PUBLIC_GRAPHQL_URL,
    wsUrl: getWebSocketUrl(),
  },
  features: {
    mockData: env.NEXT_PUBLIC_ENABLE_MOCK_DATA,
    websocket: env.NEXT_PUBLIC_ENABLE_WEBSOCKET,
    graphqlSubscriptions: env.NEXT_PUBLIC_ENABLE_GRAPHQL_SUBSCRIPTIONS,
    rateLimiting: env.NEXT_PUBLIC_ENABLE_RATE_LIMITING,
  },
  monitoring: {
    loggingEndpoint: env.NEXT_PUBLIC_LOGGING_ENDPOINT,
    metricsEndpoint: env.NEXT_PUBLIC_METRICS_ENDPOINT,
    logLevel: env.NEXT_PUBLIC_LOG_LEVEL,
  },
  environment: {
    name: env.NEXT_PUBLIC_ENVIRONMENT,
    isDevelopment: isDevelopment(),
    isProduction: isProduction(),
  },
  repository: {
    default: env.NEXT_PUBLIC_DEFAULT_REPOSITORY,
    githubUrl: env.NEXT_PUBLIC_GITHUB_REPO_URL,
    issuesUrl: env.NEXT_PUBLIC_GITHUB_ISSUES_URL,
  },
} as const