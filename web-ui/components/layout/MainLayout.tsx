'use client'

import { useEffect } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import { selectSidebarOpen, setSidebarOpen, setCommandPaletteOpen } from '@/store/slices/uiSlice'
import { Sidebar } from '@/components/navigation/Sidebar'
import { TopBar } from '@/components/navigation/TopBar'
import { WebSocketStatus } from '@/components/websocket/WebSocketStatus'
import { CommandPalette } from '@/components/command/CommandPalette'
import { SkipLinks } from '@/components/accessibility/SkipLinks'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Toaster } from '@/components/ui/use-toast'
import { X } from 'lucide-react'

interface MainLayoutProps {
  children: React.ReactNode
}

export function MainLayout({ children }: MainLayoutProps) {
  const dispatch = useAppDispatch()
  const sidebarOpen = useAppSelector(selectSidebarOpen)

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Command/Ctrl + K to open command palette
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        dispatch(setCommandPaletteOpen(true))
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [dispatch])

  return (
    <div className="flex min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
      {/* Accessibility: Skip Links */}
      <SkipLinks />
      
      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={() => dispatch(setSidebarOpen(false))}
        />
      )}

      {/* Sidebar */}
      <div className={cn(
        "fixed inset-y-0 left-0 z-50 w-64 transform transition-transform duration-300 ease-in-out lg:relative lg:translate-x-0",
        sidebarOpen ? "translate-x-0" : "-translate-x-full"
      )}>
        <div className="h-screen flex flex-col bg-card/80 backdrop-blur-xl border-r border-border/50">
          {/* Mobile close button */}
          <div className="flex items-center justify-between p-4 lg:hidden">
            <h2 className="text-lg font-semibold">MCP Memory</h2>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => dispatch(setSidebarOpen(false))}
              className="h-8 w-8 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          <nav id="main-navigation">
            <Sidebar />
          </nav>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col h-screen overflow-hidden">
        {/* Top bar */}
        <div className="bg-background/80 backdrop-blur-xl border-b border-border/50">
          <TopBar />
        </div>

        {/* Page content */}
        <main id="main-content" className="flex-1 overflow-auto">
          <div className="relative">
            {children}
          </div>
        </main>

        {/* WebSocket status indicator */}
        <div className="fixed bottom-4 right-4 z-20">
          <WebSocketStatus />
        </div>
      </div>

      {/* Command Palette */}
      <CommandPalette />

      {/* Toast notifications */}
      <Toaster />
    </div>
  )
}