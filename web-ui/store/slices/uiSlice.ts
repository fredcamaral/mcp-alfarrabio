import { createSlice, PayloadAction } from '@reduxjs/toolkit'

interface UIState {
  // Navigation
  sidebarOpen: boolean
  sidebarCollapsed: boolean
  
  // Panels
  filterPanelOpen: boolean
  configPanelOpen: boolean
  relationshipPanelOpen: boolean
  
  // Modals and dialogs
  commandPaletteOpen: boolean
  settingsModalOpen: boolean
  confirmDeleteModalOpen: boolean
  showMemoryForm: boolean
  
  // Theme
  theme: 'light' | 'dark' | 'system'
  
  // Layout
  layout: 'default' | 'compact' | 'comfortable'
  
  // Current page/section
  currentSection: 'memories' | 'patterns' | 'repositories' | 'settings' | 'graph' | 'performance' | 'realtime' | 'multi-repo'
  
  // Search state
  globalSearchFocused: boolean
  recentSearches: string[]
  
  // Notifications
  notifications: Notification[]
  
  // Performance
  enableAnimations: boolean
  enableRealtime: boolean
  
  // Debug
  debugMode: boolean
  
  // Mobile
  isMobile: boolean
  
  // Keyboard shortcuts
  keyboardShortcutsEnabled: boolean
  
  // WebSocket status
  webSocketStatus: 'connecting' | 'connected' | 'disconnected' | 'error'
  webSocketStats: {
    messagesReceived: number
    messagesSent: number
    reconnectAttempts: number
    lastConnected?: string
    lastDisconnected?: string
  }
}

interface Notification {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message?: string
  action?: {
    label: string
    onClick: () => void
  }
  duration?: number
  persistent?: boolean
}

const initialState: UIState = {
  sidebarOpen: true,
  sidebarCollapsed: false,
  filterPanelOpen: false,
  configPanelOpen: false,
  relationshipPanelOpen: false,
  commandPaletteOpen: false,
  settingsModalOpen: false,
  confirmDeleteModalOpen: false,
  showMemoryForm: false,
  theme: 'dark',
  layout: 'default',
  currentSection: 'memories',
  globalSearchFocused: false,
  recentSearches: [],
  notifications: [],
  enableAnimations: true,
  enableRealtime: true,
  debugMode: false,
  isMobile: false,
  keyboardShortcutsEnabled: true,
  webSocketStatus: 'disconnected',
  webSocketStats: {
    messagesReceived: 0,
    messagesSent: 0,
    reconnectAttempts: 0,
  },
}

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    // Sidebar
    setSidebarOpen: (state, action: PayloadAction<boolean>) => {
      state.sidebarOpen = action.payload
    },
    
    toggleSidebar: (state) => {
      state.sidebarOpen = !state.sidebarOpen
    },
    
    setSidebarCollapsed: (state, action: PayloadAction<boolean>) => {
      state.sidebarCollapsed = action.payload
    },
    
    toggleSidebarCollapsed: (state) => {
      state.sidebarCollapsed = !state.sidebarCollapsed
    },
    
    // Panels
    setFilterPanelOpen: (state, action: PayloadAction<boolean>) => {
      state.filterPanelOpen = action.payload
    },
    
    toggleFilterPanel: (state) => {
      state.filterPanelOpen = !state.filterPanelOpen
    },
    
    setConfigPanelOpen: (state, action: PayloadAction<boolean>) => {
      state.configPanelOpen = action.payload
    },
    
    toggleConfigPanel: (state) => {
      state.configPanelOpen = !state.configPanelOpen
    },
    
    setRelationshipPanelOpen: (state, action: PayloadAction<boolean>) => {
      state.relationshipPanelOpen = action.payload
    },
    
    toggleRelationshipPanel: (state) => {
      state.relationshipPanelOpen = !state.relationshipPanelOpen
    },
    
    // Modals
    setCommandPaletteOpen: (state, action: PayloadAction<boolean>) => {
      state.commandPaletteOpen = action.payload
    },
    
    toggleCommandPalette: (state) => {
      state.commandPaletteOpen = !state.commandPaletteOpen
    },
    
    setSettingsModalOpen: (state, action: PayloadAction<boolean>) => {
      state.settingsModalOpen = action.payload
    },
    
    setConfirmDeleteModalOpen: (state, action: PayloadAction<boolean>) => {
      state.confirmDeleteModalOpen = action.payload
    },
    
    setShowMemoryForm: (state, action: PayloadAction<boolean>) => {
      state.showMemoryForm = action.payload
    },
    
    // Theme
    setTheme: (state, action: PayloadAction<'light' | 'dark' | 'system'>) => {
      state.theme = action.payload
    },
    
    toggleTheme: (state) => {
      state.theme = state.theme === 'dark' ? 'light' : 'dark'
    },
    
    // Layout
    setLayout: (state, action: PayloadAction<'default' | 'compact' | 'comfortable'>) => {
      state.layout = action.payload
    },
    
    // Navigation
    setCurrentSection: (state, action: PayloadAction<'memories' | 'patterns' | 'repositories' | 'settings' | 'graph' | 'performance' | 'realtime' | 'multi-repo'>) => {
      state.currentSection = action.payload
    },
    
    // Search
    setGlobalSearchFocused: (state, action: PayloadAction<boolean>) => {
      state.globalSearchFocused = action.payload
    },
    
    addRecentSearch: (state, action: PayloadAction<string>) => {
      const query = action.payload.trim()
      if (query && !state.recentSearches.includes(query)) {
        state.recentSearches.unshift(query)
        // Keep only last 10 searches
        state.recentSearches = state.recentSearches.slice(0, 10)
      }
    },
    
    clearRecentSearches: (state) => {
      state.recentSearches = []
    },
    
    removeRecentSearch: (state, action: PayloadAction<string>) => {
      state.recentSearches = state.recentSearches.filter(search => search !== action.payload)
    },
    
    // Notifications
    addNotification: (state, action: PayloadAction<Omit<Notification, 'id'>>) => {
      const notification: Notification = {
        id: Date.now().toString(),
        ...action.payload,
      }
      state.notifications.push(notification)
    },
    
    removeNotification: (state, action: PayloadAction<string>) => {
      state.notifications = state.notifications.filter(n => n.id !== action.payload)
    },
    
    clearNotifications: (state) => {
      state.notifications = []
    },
    
    // Performance
    setEnableAnimations: (state, action: PayloadAction<boolean>) => {
      state.enableAnimations = action.payload
    },
    
    setEnableRealtime: (state, action: PayloadAction<boolean>) => {
      state.enableRealtime = action.payload
    },
    
    // Debug
    setDebugMode: (state, action: PayloadAction<boolean>) => {
      state.debugMode = action.payload
    },
    
    // Mobile
    setIsMobile: (state, action: PayloadAction<boolean>) => {
      state.isMobile = action.payload
    },
    
    // Keyboard shortcuts
    setKeyboardShortcutsEnabled: (state, action: PayloadAction<boolean>) => {
      state.keyboardShortcutsEnabled = action.payload
    },
    
    // WebSocket status
    setWebSocketStatus: (state, action: PayloadAction<'connecting' | 'connected' | 'disconnected' | 'error'>) => {
      state.webSocketStatus = action.payload
      const now = new Date().toISOString()
      
      if (action.payload === 'connected') {
        state.webSocketStats.lastConnected = now
      } else if (action.payload === 'disconnected') {
        state.webSocketStats.lastDisconnected = now
      }
    },
    
    updateWebSocketStats: (state, action: PayloadAction<Partial<UIState['webSocketStats']>>) => {
      state.webSocketStats = { ...state.webSocketStats, ...action.payload }
    },
    
    // Bulk actions
    closeAllPanels: (state) => {
      state.filterPanelOpen = false
      state.configPanelOpen = false
      state.relationshipPanelOpen = false
    },
    
    closeAllModals: (state) => {
      state.commandPaletteOpen = false
      state.settingsModalOpen = false
      state.confirmDeleteModalOpen = false
      state.showMemoryForm = false
    },
    
    // Reset to defaults
    resetUI: (state) => {
      return {
        ...initialState,
        theme: state.theme, // Preserve theme preference
        isMobile: state.isMobile, // Preserve device state
      }
    },
  },
})

export const {
  setSidebarOpen,
  toggleSidebar,
  setSidebarCollapsed,
  toggleSidebarCollapsed,
  setFilterPanelOpen,
  toggleFilterPanel,
  setConfigPanelOpen,
  toggleConfigPanel,
  setRelationshipPanelOpen,
  toggleRelationshipPanel,
  setCommandPaletteOpen,
  toggleCommandPalette,
  setSettingsModalOpen,
  setConfirmDeleteModalOpen,
  setShowMemoryForm,
  setTheme,
  toggleTheme,
  setLayout,
  setCurrentSection,
  setGlobalSearchFocused,
  addRecentSearch,
  clearRecentSearches,
  removeRecentSearch,
  addNotification,
  removeNotification,
  clearNotifications,
  setEnableAnimations,
  setEnableRealtime,
  setDebugMode,
  setIsMobile,
  setKeyboardShortcutsEnabled,
  setWebSocketStatus,
  updateWebSocketStats,
  closeAllPanels,
  closeAllModals,
  resetUI,
} = uiSlice.actions

export default uiSlice.reducer

// Selectors
export const selectSidebarOpen = (state: { ui: UIState }) => state.ui.sidebarOpen
export const selectSidebarCollapsed = (state: { ui: UIState }) => state.ui.sidebarCollapsed
export const selectFilterPanelOpen = (state: { ui: UIState }) => state.ui.filterPanelOpen
export const selectTheme = (state: { ui: UIState }) => state.ui.theme
export const selectCurrentSection = (state: { ui: UIState }) => state.ui.currentSection
export const selectNotifications = (state: { ui: UIState }) => state.ui.notifications
export const selectRecentSearches = (state: { ui: UIState }) => state.ui.recentSearches
export const selectIsMobile = (state: { ui: UIState }) => state.ui.isMobile
export const selectCommandPaletteOpen = (state: { ui: UIState }) => state.ui.commandPaletteOpen
export const selectShowMemoryForm = (state: { ui: UIState }) => state.ui.showMemoryForm
export const selectGlobalSearchFocused = (state: { ui: UIState }) => state.ui.globalSearchFocused