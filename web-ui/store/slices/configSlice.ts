import { createSlice, PayloadAction } from '@reduxjs/toolkit'

interface MCPConfig {
  // Server configuration
  host: string
  port: number
  protocol: 'http' | 'https'
  
  // API endpoints
  apiUrl: string
  graphqlUrl: string
  wsUrl: string
  
  // Authentication
  authEnabled: boolean
  apiKey?: string
  
  // Performance
  requestTimeout: number
  maxRetries: number
  cacheEnabled: boolean
  
  // Features
  realtimeEnabled: boolean
  analyticsEnabled: boolean
  debugMode: boolean
  
  // Transport protocols
  transportProtocols: {
    stdio: boolean
    http: boolean
    websocket: boolean
    sse: boolean
  }
  
  // Vector database
  vectorDb: {
    provider: 'qdrant' | 'chroma'
    host: string
    port: number
    collection: string
    dimension: number
  }
  
  // OpenAI integration
  openai: {
    apiKey?: string
    model: string
    maxTokens: number
    temperature: number
    timeout: number
  }
  
  // Storage and retention
  storage: {
    retentionDays: number
    backupEnabled: boolean
    backupInterval: number
  }
  
  // Memory management
  memory: {
    maxRepositories: number
    patternMinFrequency: number
    similarityThreshold: number
    enableTeamLearning: boolean
  }
  
  // Search and query
  search: {
    defaultLimit: number
    defaultListLimit: number
    minRelevance: number
    embeddingDimension: number
    maxContentLength: number
  }
  
  // Health and monitoring
  health: {
    checkTimeout: number
    readinessTimeout: number
    dbCheckTimeout: number
    vectorCheckTimeout: number
  }
  
  // Performance thresholds
  performance: {
    dbSlowThreshold: number
    vectorSlowThreshold: number
    cacheMaxSize: number
    queryTtl: number
  }
}

interface ConfigState {
  config: MCPConfig
  isLoading: boolean
  error?: string
  hasUnsavedChanges: boolean
  connectionStatus: 'connected' | 'disconnected' | 'connecting' | 'error'
  lastSaved?: string
  
  // Configuration history
  configHistory: Array<{
    timestamp: string
    config: MCPConfig
    description: string
  }>
  
  // Validation
  validationErrors: Record<string, string>
  
  // Export/Import
  exportFormat: 'json' | 'yaml' | 'env'
}

const defaultConfig: MCPConfig = {
  host: 'localhost',
  port: 9080,
  protocol: 'http',
  apiUrl: 'http://localhost:9080',
  graphqlUrl: 'http://localhost:9080/graphql',
  wsUrl: 'ws://localhost:9080/ws',
  authEnabled: false,
  requestTimeout: 30000,
  maxRetries: 3,
  cacheEnabled: true,
  realtimeEnabled: true,
  analyticsEnabled: false,
  debugMode: false,
  transportProtocols: {
    stdio: true,
    http: true,
    websocket: true,
    sse: true,
  },
  vectorDb: {
    provider: 'qdrant',
    host: 'localhost',
    port: 6333,
    collection: 'mcp_memory',
    dimension: 1536,
  },
  openai: {
    model: 'text-embedding-ada-002',
    maxTokens: 8191,
    temperature: 0.0,
    timeout: 60000,
  },
  storage: {
    retentionDays: 90,
    backupEnabled: false,
    backupInterval: 24,
  },
  memory: {
    maxRepositories: 100,
    patternMinFrequency: 3,
    similarityThreshold: 0.6,
    enableTeamLearning: true,
  },
  search: {
    defaultLimit: 20,
    defaultListLimit: 50,
    minRelevance: 0.3,
    embeddingDimension: 1536,
    maxContentLength: 8000,
  },
  health: {
    checkTimeout: 30,
    readinessTimeout: 10,
    dbCheckTimeout: 5,
    vectorCheckTimeout: 10,
  },
  performance: {
    dbSlowThreshold: 1,
    vectorSlowThreshold: 2,
    cacheMaxSize: 1000,
    queryTtl: 15,
  },
}

const initialState: ConfigState = {
  config: defaultConfig,
  isLoading: false,
  hasUnsavedChanges: false,
  connectionStatus: 'disconnected',
  configHistory: [],
  validationErrors: {},
  exportFormat: 'json',
}

const configSlice = createSlice({
  name: 'config',
  initialState,
  reducers: {
    // Config management
    setConfig: (state, action: PayloadAction<MCPConfig>) => {
      state.config = action.payload
      state.hasUnsavedChanges = true
      state.validationErrors = {}
    },
    
    updateConfig: (state, action: PayloadAction<Partial<MCPConfig>>) => {
      state.config = { ...state.config, ...action.payload }
      state.hasUnsavedChanges = true
    },
    
    resetConfig: (state) => {
      state.config = defaultConfig
      state.hasUnsavedChanges = true
      state.validationErrors = {}
    },
    
    // Specific config updates
    updateServerConfig: (state, action: PayloadAction<{
      host?: string
      port?: number
      protocol?: 'http' | 'https'
    }>) => {
      Object.assign(state.config, action.payload)
      
      // Update derived URLs
      const baseUrl = `${state.config.protocol}://${state.config.host}:${state.config.port}`
      state.config.apiUrl = baseUrl
      state.config.graphqlUrl = `${baseUrl}/graphql`
      state.config.wsUrl = `${state.config.protocol === 'https' ? 'wss' : 'ws'}://${state.config.host}:${state.config.port}/ws`
      
      state.hasUnsavedChanges = true
    },
    
    updateVectorDbConfig: (state, action: PayloadAction<Partial<ConfigState['config']['vectorDb']>>) => {
      state.config.vectorDb = { ...state.config.vectorDb, ...action.payload }
      state.hasUnsavedChanges = true
    },
    
    updateOpenAIConfig: (state, action: PayloadAction<Partial<ConfigState['config']['openai']>>) => {
      state.config.openai = { ...state.config.openai, ...action.payload }
      state.hasUnsavedChanges = true
    },
    
    updateTransportProtocols: (state, action: PayloadAction<Partial<ConfigState['config']['transportProtocols']>>) => {
      state.config.transportProtocols = { ...state.config.transportProtocols, ...action.payload }
      state.hasUnsavedChanges = true
    },
    
    // State management
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload
    },
    
    setError: (state, action: PayloadAction<string | undefined>) => {
      state.error = action.payload
    },
    
    setConnectionStatus: (state, action: PayloadAction<ConfigState['connectionStatus']>) => {
      state.connectionStatus = action.payload
    },
    
    // Save management
    markSaved: (state) => {
      state.hasUnsavedChanges = false
      state.lastSaved = new Date().toISOString()
      
      // Add to history
      state.configHistory.unshift({
        timestamp: new Date().toISOString(),
        config: { ...state.config },
        description: 'Configuration saved',
      })
      
      // Keep only last 20 history entries
      state.configHistory = state.configHistory.slice(0, 20)
    },
    
    // Validation
    setValidationErrors: (state, action: PayloadAction<Record<string, string>>) => {
      state.validationErrors = action.payload
    },
    
    clearValidationErrors: (state) => {
      state.validationErrors = {}
    },
    
    // History management
    restoreFromHistory: (state, action: PayloadAction<number>) => {
      const historyEntry = state.configHistory[action.payload]
      if (historyEntry) {
        state.config = historyEntry.config
        state.hasUnsavedChanges = true
      }
    },
    
    clearConfigHistory: (state) => {
      state.configHistory = []
    },
    
    // Export/Import
    setExportFormat: (state, action: PayloadAction<ConfigState['exportFormat']>) => {
      state.exportFormat = action.payload
    },
    
    importConfig: (state, action: PayloadAction<MCPConfig>) => {
      state.config = action.payload
      state.hasUnsavedChanges = true
      state.validationErrors = {}
    },
    
    // Development helpers
    enableDebugMode: (state) => {
      state.config.debugMode = true
      state.hasUnsavedChanges = true
    },
    
    disableDebugMode: (state) => {
      state.config.debugMode = false
      state.hasUnsavedChanges = true
    },
    
    toggleFeature: (state, action: PayloadAction<{
      feature: 'realtimeEnabled' | 'analyticsEnabled' | 'cacheEnabled' | 'debugMode'
    }>) => {
      const { feature } = action.payload
      state.config[feature] = !state.config[feature]
      state.hasUnsavedChanges = true
    },
  },
})

export const {
  setConfig,
  updateConfig,
  resetConfig,
  updateServerConfig,
  updateVectorDbConfig,
  updateOpenAIConfig,
  updateTransportProtocols,
  setLoading,
  setError,
  setConnectionStatus,
  markSaved,
  setValidationErrors,
  clearValidationErrors,
  restoreFromHistory,
  clearConfigHistory,
  setExportFormat,
  importConfig,
  enableDebugMode,
  disableDebugMode,
  toggleFeature,
} = configSlice.actions

export default configSlice.reducer

// Selectors
export const selectConfig = (state: { config: ConfigState }) => state.config.config
export const selectConfigLoading = (state: { config: ConfigState }) => state.config.isLoading
export const selectConfigError = (state: { config: ConfigState }) => state.config.error
export const selectHasUnsavedChanges = (state: { config: ConfigState }) => state.config.hasUnsavedChanges
export const selectConnectionStatus = (state: { config: ConfigState }) => state.config.connectionStatus
export const selectConfigHistory = (state: { config: ConfigState }) => state.config.configHistory
export const selectValidationErrors = (state: { config: ConfigState }) => state.config.validationErrors
export const selectServerConfig = (state: { config: ConfigState }) => ({
  host: state.config.config.host,
  port: state.config.config.port,
  protocol: state.config.config.protocol,
})
export const selectVectorDbConfig = (state: { config: ConfigState }) => state.config.config.vectorDb
export const selectOpenAIConfig = (state: { config: ConfigState }) => state.config.config.openai