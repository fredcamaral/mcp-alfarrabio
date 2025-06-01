'use client'

import { useAppSelector } from '@/store/store'
import { selectCurrentSection } from '@/store/slices/uiSlice'
import { MainLayout } from '@/components/layout/MainLayout'
import { MemoryList } from '@/components/memories/MemoryList'
import { MemoryDetails } from '@/components/memories/MemoryDetails'
import { ConfigInterface } from '@/components/config/ConfigInterface'

export default function HomePage() {
  const currentSection = useAppSelector(selectCurrentSection)

  const renderContent = () => {
    switch (currentSection) {
      case 'memories':
        return (
          <div className="flex h-full">
            <div className="flex-1 p-6">
              <MemoryList />
            </div>
            <div className="w-96 border-l border-border">
              <MemoryDetails />
            </div>
          </div>
        )
      case 'patterns':
        return (
          <div className="p-6">
            <div className="text-center py-12">
              <div className="text-6xl mb-4">📊</div>
              <h2 className="text-2xl font-semibold mb-2">Patterns & Insights</h2>
              <p className="text-muted-foreground">
                Pattern recognition and insights coming soon
              </p>
            </div>
          </div>
        )
      case 'repositories':
        return (
          <div className="p-6">
            <div className="text-center py-12">
              <div className="text-6xl mb-4">🔗</div>
              <h2 className="text-2xl font-semibold mb-2">Repository Management</h2>
              <p className="text-muted-foreground">
                Multi-repository features coming soon
              </p>
            </div>
          </div>
        )
      case 'settings':
        return (
          <div className="p-6">
            <ConfigInterface />
          </div>
        )
      default:
        return (
          <div className="p-6">
            <div className="text-center py-12">
              <div className="text-6xl mb-4">🧠</div>
              <h2 className="text-2xl font-semibold mb-2">Welcome to MCP Memory</h2>
              <p className="text-muted-foreground">
                Your AI memory management system
              </p>
            </div>
          </div>
        )
    }
  }

  return (
    <MainLayout>
      {renderContent()}
    </MainLayout>
  )
}