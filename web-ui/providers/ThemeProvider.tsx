'use client'

import { createContext, useContext, useEffect, useState } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import { selectTheme, setTheme } from '@/store/slices/uiSlice'

type Theme = 'dark' | 'light' | 'system'

type ThemeProviderProps = {
    children: React.ReactNode
    defaultTheme?: Theme
    storageKey?: string
}

type ThemeProviderState = {
    theme: Theme
    setTheme: (theme: Theme) => void
}

const initialState: ThemeProviderState = {
    theme: 'system',
    setTheme: () => null,
}

const ThemeProviderContext = createContext<ThemeProviderState>(initialState)

export function ThemeProvider({
    children,
    defaultTheme = 'system',
    storageKey = 'mcp-memory-theme',
    ...props
}: ThemeProviderProps) {
    const dispatch = useAppDispatch()
    const reduxTheme = useAppSelector(selectTheme)
    const [theme, setThemeState] = useState<Theme>(reduxTheme || defaultTheme)

    useEffect(() => {
        // This is acceptable DOM manipulation for theming
        // It needs to be on the document root for CSS variables to work properly
        const root = document.documentElement

        root.classList.remove('light', 'dark')

        if (theme === 'system') {
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)')
                .matches
                ? 'dark'
                : 'light'

            root.classList.add(systemTheme)
            return
        }

        root.classList.add(theme)
    }, [theme])

    useEffect(() => {
        // Sync with Redux state
        setThemeState(reduxTheme)
    }, [reduxTheme])

    const value = {
        theme,
        setTheme: (theme: Theme) => {
            localStorage.setItem(storageKey, theme)
            setThemeState(theme)
            dispatch(setTheme(theme))
        },
    }

    return (
        <ThemeProviderContext.Provider {...props} value={value}>
            {children}
        </ThemeProviderContext.Provider>
    )
}

export const useTheme = () => {
    const context = useContext(ThemeProviderContext)

    if (context === undefined)
        throw new Error('useTheme must be used within a ThemeProvider')

    return context
} 