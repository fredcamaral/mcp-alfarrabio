/**
 * Multi-Repository Intelligence Manager
 * 
 * Provides cross-repository pattern analysis, knowledge discovery,
 * and impact analysis across multiple repositories
 */

'use client'

import { useState } from 'react'
import { useMultiRepo } from '@/hooks/useMultiRepo'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import {
  GitBranch,
  Globe,
  TrendingUp,
  Network,
  Layers3,
  AlertTriangle,
  CheckCircle,
  Code,
  FileText,
  Users,
  Activity,
  Link,
  Zap,
  Brain,
  BarChart3,
  RefreshCw
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { logger } from '@/lib/logger'

interface MultiRepoManagerProps {
  className?: string
}

export function MultiRepoManager({ className }: MultiRepoManagerProps) {
  const [activeTab, setActiveTab] = useState('overview')
  
  // Use the multi-repo hook
  const {
    repositories,
    patterns,
    knowledgeLinks,
    isLoading,
    analyzeRepositories,
  } = useMultiRepo({ autoRefresh: true, refreshInterval: 60000 })

  const handleAnalyzeAll = async () => {
    try {
      await analyzeRepositories()
    } catch (err) {
      logger.error('Failed to analyze repositories:', err)
    }
  }

  const formatBytes = (bytes: number): string => {
    const sizes = ['B', 'KB', 'MB', 'GB']
    if (bytes === 0) return '0 B'
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
  }

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const hours = Math.floor(diff / (1000 * 60 * 60))
    
    if (hours < 1) return 'Just now'
    if (hours < 24) return `${hours}h ago`
    if (hours < 48) return 'Yesterday'
    return date.toLocaleDateString()
  }


  const getImpactColor = (impact: 'high' | 'medium' | 'low') => {
    switch (impact) {
      case 'high': return 'destructive'
      case 'medium': return 'secondary'
      case 'low': return 'outline'
    }
  }

  return (
    <div className={cn("space-y-6", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight flex items-center gap-3">
            <Globe className="h-8 w-8 text-primary" />
            Multi-Repository Intelligence
          </h2>
          <p className="text-muted-foreground">
            Cross-repository pattern analysis and knowledge discovery
          </p>
        </div>
        
        <Button onClick={handleAnalyzeAll} disabled={isLoading}>
          {isLoading ? (
            <>
              <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
              Analyzing...
            </>
          ) : (
            <>
              <Brain className="mr-2 h-4 w-4" />
              Analyze All
            </>
          )}
        </Button>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Repositories</p>
                <p className="text-2xl font-bold">{repositories.length}</p>
              </div>
              <GitBranch className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Cross-Repo Patterns</p>
                <p className="text-2xl font-bold">{patterns.length}</p>
              </div>
              <Network className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Knowledge Links</p>
                <p className="text-2xl font-bold">{knowledgeLinks.length}</p>
              </div>
              <Link className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Memories</p>
                <p className="text-2xl font-bold">
                  {repositories.reduce((sum, repo) => sum + repo.memoryCount, 0)}
                </p>
              </div>
              <Brain className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="patterns">Patterns</TabsTrigger>
          <TabsTrigger value="knowledge">Knowledge Graph</TabsTrigger>
          <TabsTrigger value="impact">Impact Analysis</TabsTrigger>
          <TabsTrigger value="learning">Shared Learning</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Repository Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {repositories.map((repo) => (
                  <div
                    key={repo.id}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex items-center gap-4">
                      <div className={cn(
                        "w-2 h-2 rounded-full",
                        repo.status === 'active' && "bg-success",
                        repo.status === 'inactive' && "bg-muted-foreground",
                        repo.status === 'analyzing' && "bg-warning animate-pulse"
                      )} />
                      <div>
                        <h4 className="font-medium">{repo.name}</h4>
                        <p className="text-sm text-muted-foreground">{repo.url}</p>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <p className="text-sm font-medium">{repo.memoryCount} memories</p>
                        <p className="text-xs text-muted-foreground">
                          {repo.language} • {repo.size && formatBytes(repo.size)}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm text-muted-foreground">
                          Updated {formatDate(repo.lastUpdated)}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Insights</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <TrendingUp className="h-4 w-4 text-success" />
                    <h4 className="font-medium">Most Active Repository</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <strong>lerian-mcp-memory</strong> with 342 memories and daily updates
                  </p>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Layers3 className="h-4 w-4 text-primary" />
                    <h4 className="font-medium">Common Technology Stack</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <strong>Go</strong> is used in 66% of repositories
                  </p>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Network className="h-4 w-4 text-warning" />
                    <h4 className="font-medium">Highest Pattern Frequency</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <strong>Error Handling Pattern</strong> found 45 times
                  </p>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Zap className="h-4 w-4 text-purple" />
                    <h4 className="font-medium">Strongest Connection</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <strong>midaz ↔ transaction-api</strong> (92% strength)
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Patterns Tab */}
        <TabsContent value="patterns" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Cross-Repository Patterns</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {patterns.map((pattern) => (
                  <div key={pattern.id} className="border rounded-lg p-4">
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <h4 className="font-medium flex items-center gap-2">
                          {pattern.name}
                          <Badge variant={getImpactColor(pattern.impact)}>
                            {pattern.impact} impact
                          </Badge>
                        </h4>
                        <p className="text-sm text-muted-foreground mt-1">
                          {pattern.description}
                        </p>
                      </div>
                      <Badge variant="secondary">{pattern.frequency} occurrences</Badge>
                    </div>
                    
                    <div className="mt-3 flex items-center gap-4 text-sm">
                      <div className="flex items-center gap-1">
                        <GitBranch className="h-3 w-3" />
                        <span>{pattern.repositories.length} repositories</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Code className="h-3 w-3" />
                        <span>{pattern.type}</span>
                      </div>
                    </div>
                    
                    {pattern.examples && pattern.examples.length > 0 && (
                      <div className="mt-3 p-3 bg-muted/50 rounded text-xs font-mono">
                        <p className="text-muted-foreground mb-1">
                          {pattern.examples[0].repository}:{pattern.examples[0].file}:{pattern.examples[0].line}
                        </p>
                        <pre className="text-foreground">{pattern.examples[0].snippet}</pre>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Knowledge Graph Tab */}
        <TabsContent value="knowledge" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Repository Knowledge Links</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {knowledgeLinks.map((link) => (
                  <div key={link.id} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex items-center gap-4">
                      <div className="text-center">
                        <p className="font-medium">{link.sourceRepo}</p>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="h-px w-8 bg-border" />
                        <Badge variant="outline">{link.linkType}</Badge>
                        <div className="h-px w-8 bg-border" />
                      </div>
                      <div className="text-center">
                        <p className="font-medium">{link.targetRepo}</p>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-4">
                      <Progress value={link.strength * 100} className="w-24" />
                      <span className="text-sm font-medium">{Math.round(link.strength * 100)}%</span>
                    </div>
                  </div>
                ))}
              </div>
              
              <Alert className="mt-4">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Visual knowledge graph visualization coming soon. This will show
                  an interactive network diagram of repository relationships.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Impact Analysis Tab */}
        <TabsContent value="impact" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Change Impact Analysis</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="p-4 border rounded-lg">
                  <h4 className="font-medium mb-2">Analyze Potential Impact</h4>
                  <p className="text-sm text-muted-foreground mb-4">
                    Select a repository and describe the change to see its impact across other repositories
                  </p>
                  <div className="flex gap-2">
                    <Input
                      placeholder="Describe the change (e.g., 'Update API authentication method')"
                      className="flex-1"
                    />
                    <Button>Analyze Impact</Button>
                  </div>
                </div>
                
                <Alert>
                  <CheckCircle className="h-4 w-4" />
                  <AlertDescription>
                    Impact analysis helps predict how changes in one repository might
                    affect others based on historical patterns and dependencies.
                  </AlertDescription>
                </Alert>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Shared Learning Tab */}
        <TabsContent value="learning" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Shared Learning Insights</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="p-4 border rounded-lg">
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <Users className="h-4 w-4" />
                    Team Practices
                  </h4>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    <li>• Consistent error handling across 67% of repositories</li>
                    <li>• Shared logging patterns in Go projects</li>
                    <li>• Common testing strategies identified</li>
                  </ul>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <FileText className="h-4 w-4" />
                    Documentation Patterns
                  </h4>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    <li>• README structure consistency: 85%</li>
                    <li>• API documentation format shared</li>
                    <li>• Code comment style alignment</li>
                  </ul>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <Activity className="h-4 w-4" />
                    Performance Optimizations
                  </h4>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    <li>• Circuit breaker patterns in 3 repos</li>
                    <li>• Caching strategies shared</li>
                    <li>• Database query optimizations</li>
                  </ul>
                </div>
                
                <div className="p-4 border rounded-lg">
                  <h4 className="font-medium mb-2 flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Architecture Decisions
                  </h4>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    <li>• Microservices patterns adopted</li>
                    <li>• Event-driven architecture in 40%</li>
                    <li>• Consistent API design principles</li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}