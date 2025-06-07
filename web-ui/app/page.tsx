'use client'

import { useEffect } from 'react'
import { useAppSelector, useAppDispatch } from '@/store/store'
import { selectCurrentSection, setCurrentSection, selectFilterPanelOpen, toggleFilterPanel } from '@/store/slices/uiSlice'
import { useMemoryContext } from '@/hooks/useMemoryAPI'
import { MainLayout } from '@/components/layout/MainLayout'
import { MemoryList } from '@/components/memories/MemoryList'
import { MemoryDetails } from '@/components/memories/MemoryDetails'
import { MemoryFormDialog } from '@/components/memories/MemoryFormDialog'
import { ConfigInterface } from '@/components/config/ConfigInterface'
import { MemorySearch } from '@/components/search/MemorySearch'
import { PatternsDashboard } from '@/components/patterns/PatternsDashboard'
import { RepositoryManager } from '@/components/repositories/RepositoryManager'
import { FilterPanel } from '@/components/filters/FilterPanel'
import { KnowledgeGraph } from '@/components/graph/KnowledgeGraph'
import { PerformanceDashboard } from '@/components/performance/PerformanceDashboard'
import { RealtimeMemoryFeed } from '@/components/realtime/RealtimeMemoryFeed'
import { BackupManager } from '@/components/backup/BackupManager'
import { PreferencesPanel } from '@/components/preferences/PreferencesPanel'
import { MultiRepoManager } from '@/components/multi-repo/MultiRepoManager'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Brain, Search, BarChart3, GitBranch, Sparkles, Filter, Activity } from 'lucide-react'
import { ErrorBoundary } from '@/components/error/ErrorBoundary'
import { logger } from '@/lib/logger'

export default function HomePage() {
  const dispatch = useAppDispatch()
  const currentSection = useAppSelector(selectCurrentSection)
  const filterPanelOpen = useAppSelector(selectFilterPanelOpen)

  // Initialize with default repository context
  const { data: contextData, isLoading: contextLoading } = useMemoryContext(process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory')

  useEffect(() => {
    // Load initial context data
    if (contextData && !contextLoading) {
      // This would populate initial memories if available
      // Context loaded successfully
    }
  }, [contextData, contextLoading])

  const renderContent = () => {
    switch (currentSection) {
      case 'memories':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Memory Explorer error:', error)}
            enableRetry={true}
          >
            <div className="flex min-h-[calc(100vh-4rem)]">
              <div className="flex-1 flex flex-col">
                {/* Search Header */}
                <div className="px-6 py-6 border-b bg-card/50 backdrop-blur-sm">
                  <div>
                    <div className="flex items-center justify-between mb-4">
                      <h1 className="text-2xl font-bold flex items-center gap-3">
                        <Brain className="h-7 w-7 text-primary" />
                        Memory Explorer
                      </h1>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => dispatch(toggleFilterPanel())}
                        className="gap-2"
                      >
                        <Filter className="h-4 w-4" />
                        Advanced Filters
                      </Button>
                    </div>
                    <ErrorBoundary
                      onError={(error) => logger.error('Memory Search error:', error)}
                    >
                      <MemorySearch className="w-full" />
                    </ErrorBoundary>
                  </div>
                </div>

                {/* Memory List */}
                <div className="flex-1 px-6 py-6">
                  <ErrorBoundary
                    onError={(error) => logger.error('Memory List error:', error)}
                    enableRetry={true}
                  >
                    <MemoryList />
                  </ErrorBoundary>
                </div>
              </div>

              {/* Details Panel */}
              <div className="w-96 border-l bg-card/30 backdrop-blur-sm">
                <ErrorBoundary
                  onError={(error) => logger.error('Memory Details error:', error)}
                  enableRetry={true}
                >
                  <MemoryDetails />
                </ErrorBoundary>
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'patterns':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Patterns Dashboard error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-6xl">
                <div className="mb-8">
                  <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
                    <BarChart3 className="h-8 w-8 text-primary" />
                    Patterns & Insights
                  </h1>
                  <p className="text-muted-foreground text-lg">
                    Discover patterns and insights across your conversation memories
                  </p>
                </div>
                <PatternsDashboard />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'repositories':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Repository Manager error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-6xl">
                <div className="mb-8">
                  <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
                    <GitBranch className="h-8 w-8 text-primary" />
                    Repository Management
                  </h1>
                  <p className="text-muted-foreground text-lg">
                    Manage memory across multiple repositories and projects
                  </p>
                </div>
                <RepositoryManager />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'multi-repo':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Multi-Repo Manager error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-7xl mx-auto">
                <MultiRepoManager />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'graph':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Knowledge Graph error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] p-6">
              <KnowledgeGraph className="h-[calc(100vh-8rem)]" />
            </div>
          </ErrorBoundary>
        )

      case 'performance':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Performance Dashboard error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-7xl mx-auto">
                <div className="mb-8">
                  <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
                    <Activity className="h-8 w-8 text-primary" />
                    Performance Dashboard
                  </h1>
                  <p className="text-muted-foreground text-lg">
                    Monitor real-time application performance and optimize your experience
                  </p>
                </div>
                <PerformanceDashboard />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'realtime':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Realtime Feed error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-4xl mx-auto">
                <div className="mb-8">
                  <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
                    <Activity className="h-8 w-8 text-primary" />
                    Realtime Memory Feed
                  </h1>
                  <p className="text-muted-foreground text-lg">
                    Live updates of memories and patterns as they are detected
                  </p>
                </div>
                <RealtimeMemoryFeed 
                  repository={process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || "github.com/lerianstudio/lerian-mcp-memory"}
                  maxItems={30}
                />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'backup':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Backup Manager error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-6xl mx-auto">
                <BackupManager />
              </div>
            </div>
          </ErrorBoundary>
        )

      case 'settings':
        return (
          <ErrorBoundary
            onError={(error) => logger.error('Settings/Config error:', error)}
            enableRetry={true}
          >
            <div className="min-h-[calc(100vh-4rem)] px-6 py-6">
              <div className="max-w-6xl mx-auto space-y-8">
                <ConfigInterface />
                <div className="border-t pt-8">
                  <PreferencesPanel />
                </div>
              </div>
            </div>
          </ErrorBoundary>
        )

      default:
        return (
          <div className="min-h-[calc(100vh-4rem)] flex items-center justify-center px-6 py-6">
            <div className="text-center max-w-2xl">
              <div className="mb-8">
                <Sparkles className="h-20 w-20 text-primary mx-auto mb-6" />
                <h1 className="text-4xl font-bold mb-4 bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
                  Welcome to MCP Memory
                </h1>
                <p className="text-xl text-muted-foreground mb-8">
                  Your intelligent memory management system for AI conversations
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
                <Card 
                  className="p-6 hover:shadow-lg transition-all cursor-pointer border-2 hover:border-primary/50"
                  onClick={() => dispatch(setCurrentSection('memories'))}
                >
                  <CardContent className="p-0">
                    <Brain className="h-12 w-12 text-primary mb-4" />
                    <h3 className="text-lg font-semibold mb-2">Explore Memories</h3>
                    <p className="text-sm text-muted-foreground">
                      Browse and search your conversation memories
                    </p>
                  </CardContent>
                </Card>

                <Card 
                  className="p-6 hover:shadow-lg transition-all cursor-pointer border-2 hover:border-primary/50"
                  onClick={() => dispatch(setCurrentSection('patterns'))}
                >
                  <CardContent className="p-0">
                    <BarChart3 className="h-12 w-12 text-primary mb-4" />
                    <h3 className="text-lg font-semibold mb-2">View Patterns</h3>
                    <p className="text-sm text-muted-foreground">
                      Discover insights and patterns in your data
                    </p>
                  </CardContent>
                </Card>
              </div>

              <Button 
                size="lg" 
                className="px-8"
                onClick={() => dispatch(setCurrentSection('memories'))}
              >
                <Search className="mr-2 h-5 w-5" />
                Start Exploring
              </Button>
            </div>
          </div>
        )
    }
  }

  return (
    <MainLayout>
      {renderContent()}
      <MemoryFormDialog />
      <FilterPanel 
        isOpen={filterPanelOpen} 
        onClose={() => dispatch(toggleFilterPanel())} 
      />
    </MainLayout>
  )
}