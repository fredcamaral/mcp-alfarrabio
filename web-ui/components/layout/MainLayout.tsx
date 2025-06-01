'use client'

import { useAppSelector } from '@/store/store'
import { selectSidebarOpen } from '@/store/slices/uiSlice'
import { cn } from '@/lib/utils'
import { Sidebar } from '@/components/navigation/Sidebar'
import { TopBar } from '@/components/navigation/TopBar'

interface MainLayoutProps {
  children: React.ReactNode
}

export function MainLayout({ children }: MainLayoutProps) {
  const sidebarOpen = useAppSelector(selectSidebarOpen)

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Top navigation */}
      <TopBar />
      
      {/* Main content area */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar */}
        <Sidebar />
        
        {/* Main content */}
        <main className={cn(
          "flex-1 flex flex-col overflow-hidden",
          "transition-all duration-300"
        )}>
          <div className="flex-1 overflow-auto">
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}