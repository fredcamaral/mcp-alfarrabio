import { configureStore } from '@reduxjs/toolkit'
import { TypedUseSelectorHook, useDispatch, useSelector } from 'react-redux'

import memoriesReducer from './slices/memoriesSlice'
import uiReducer from './slices/uiSlice'
import filtersReducer from './slices/filtersSlice'
import configReducer from './slices/configSlice'
import preferencesReducer from './slices/preferencesSlice'

export const store = configureStore({
  reducer: {
    memories: memoriesReducer,
    ui: uiReducer,
    filters: filtersReducer,
    config: configReducer,
    preferences: preferencesReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: ['persist/PERSIST', 'persist/REHYDRATE'],
      },
    }),
  devTools: process.env.NODE_ENV !== 'production',
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch

// Type-safe hooks
export const useAppDispatch = () => useDispatch<AppDispatch>()
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector