import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import {
  UserPreferences,
  loadPreferences,
  savePreferences,
  resetPreferences as resetPrefs,
  exportPreferences as exportPrefs,
  importPreferences as importPrefs
} from '@/lib/preferences'

interface PreferencesState extends UserPreferences {
  isLoading: boolean
  lastSaved: string | null
  saveError: string | null
}

// Load initial state from localStorage
const loadedPreferences = loadPreferences()

const initialState: PreferencesState = {
  ...loadedPreferences,
  isLoading: false,
  lastSaved: null,
  saveError: null
}

const preferencesSlice = createSlice({
  name: 'preferences',
  initialState,
  reducers: {
    // Theme
    setTheme: (state, action: PayloadAction<'light' | 'dark' | 'system'>) => {
      state.theme = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Layout
    setLayout: (state, action: PayloadAction<'default' | 'compact' | 'comfortable'>) => {
      state.layout = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setSidebarCollapsed: (state, action: PayloadAction<boolean>) => {
      state.sidebarCollapsed = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // UI preferences
    setEnableAnimations: (state, action: PayloadAction<boolean>) => {
      state.enableAnimations = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setEnableRealtime: (state, action: PayloadAction<boolean>) => {
      state.enableRealtime = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setKeyboardShortcutsEnabled: (state, action: PayloadAction<boolean>) => {
      state.keyboardShortcutsEnabled = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Performance
    setCacheEnabled: (state, action: PayloadAction<boolean>) => {
      state.cacheEnabled = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setDebugMode: (state, action: PayloadAction<boolean>) => {
      state.debugMode = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Memory preferences
    setDefaultView: (state, action: PayloadAction<'memories' | 'patterns' | 'repositories'>) => {
      state.defaultView = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setMemoryListLayout: (state, action: PayloadAction<'grid' | 'list'>) => {
      state.memoryListLayout = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setMemoriesPerPage: (state, action: PayloadAction<number>) => {
      state.memoriesPerPage = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Filter preferences
    setDefaultFilters: (state, action: PayloadAction<UserPreferences['defaultFilters']>) => {
      state.defaultFilters = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    updateDefaultFilter: <K extends keyof UserPreferences['defaultFilters']>(
      state: PreferencesState,
      action: PayloadAction<{ key: K; value: UserPreferences['defaultFilters'][K] }>
    ) => {
      state.defaultFilters[action.payload.key] = action.payload.value
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Search preferences
    addRecentSearch: (state, action: PayloadAction<string>) => {
      const search = action.payload.trim()
      if (search && !state.recentSearches.includes(search)) {
        state.recentSearches.unshift(search)
        state.recentSearches = state.recentSearches.slice(0, state.maxRecentSearches)
        savePreferences(state)
        state.lastSaved = new Date().toISOString()
      }
    },

    removeRecentSearch: (state, action: PayloadAction<string>) => {
      state.recentSearches = state.recentSearches.filter(s => s !== action.payload)
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    clearRecentSearches: (state) => {
      state.recentSearches = []
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setSearchHistoryEnabled: (state, action: PayloadAction<boolean>) => {
      state.searchHistoryEnabled = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Advanced preferences
    setAutoBackupEnabled: (state, action: PayloadAction<boolean>) => {
      state.autoBackupEnabled = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setBackupFrequency: (state, action: PayloadAction<'daily' | 'weekly' | 'monthly'>) => {
      state.backupFrequency = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setNotificationSound: (state, action: PayloadAction<boolean>) => {
      state.notificationSound = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // WebSocket preferences
    setAutoReconnect: (state, action: PayloadAction<boolean>) => {
      state.autoReconnect = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    setReconnectDelay: (state, action: PayloadAction<number>) => {
      state.reconnectDelay = action.payload
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Bulk update
    updatePreferences: (state, action: PayloadAction<Partial<UserPreferences>>) => {
      Object.assign(state, action.payload)
      savePreferences(state)
      state.lastSaved = new Date().toISOString()
    },

    // Reset
    resetPreferences: (state) => {
      const defaults = resetPrefs()
      Object.assign(state, defaults)
      state.lastSaved = new Date().toISOString()
      state.saveError = null
    },

    // Import/Export
    importPreferences: (state, action: PayloadAction<string>) => {
      state.isLoading = true
      const imported = importPrefs(action.payload)
      if (imported) {
        Object.assign(state, imported)
        state.lastSaved = new Date().toISOString()
        state.saveError = null
      } else {
        state.saveError = 'Failed to import preferences'
      }
      state.isLoading = false
    },

    // Error handling
    setSaveError: (state, action: PayloadAction<string | null>) => {
      state.saveError = action.payload
    }
  }
})

export const {
  setTheme,
  setLayout,
  setSidebarCollapsed,
  setEnableAnimations,
  setEnableRealtime,
  setKeyboardShortcutsEnabled,
  setCacheEnabled,
  setDebugMode,
  setDefaultView,
  setMemoryListLayout,
  setMemoriesPerPage,
  setDefaultFilters,
  updateDefaultFilter,
  addRecentSearch,
  removeRecentSearch,
  clearRecentSearches,
  setSearchHistoryEnabled,
  setAutoBackupEnabled,
  setBackupFrequency,
  setNotificationSound,
  setAutoReconnect,
  setReconnectDelay,
  updatePreferences,
  resetPreferences,
  importPreferences,
  setSaveError
} = preferencesSlice.actions

export default preferencesSlice.reducer

// Selectors
export const selectPreferences = (state: { preferences: PreferencesState }) => state.preferences
export const selectTheme = (state: { preferences: PreferencesState }) => state.preferences.theme
export const selectLayout = (state: { preferences: PreferencesState }) => state.preferences.layout
export const selectSidebarCollapsed = (state: { preferences: PreferencesState }) => state.preferences.sidebarCollapsed
export const selectEnableAnimations = (state: { preferences: PreferencesState }) => state.preferences.enableAnimations
export const selectEnableRealtime = (state: { preferences: PreferencesState }) => state.preferences.enableRealtime
export const selectMemoryListLayout = (state: { preferences: PreferencesState }) => state.preferences.memoryListLayout
export const selectMemoriesPerPage = (state: { preferences: PreferencesState }) => state.preferences.memoriesPerPage
export const selectDefaultFilters = (state: { preferences: PreferencesState }) => state.preferences.defaultFilters
export const selectRecentSearches = (state: { preferences: PreferencesState }) => state.preferences.recentSearches
export const selectAutoBackupEnabled = (state: { preferences: PreferencesState }) => state.preferences.autoBackupEnabled
export const selectKeyboardShortcutsEnabled = (state: { preferences: PreferencesState }) => state.preferences.keyboardShortcutsEnabled
export const selectDebugMode = (state: { preferences: PreferencesState }) => state.preferences.debugMode
export const selectLastSaved = (state: { preferences: PreferencesState }) => state.preferences.lastSaved

// Thunk to export preferences
export const exportPreferencesThunk = () => {
  return exportPrefs()
}