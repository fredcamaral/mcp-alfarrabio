'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useGetPatterns } from '@/lib/graphql/hooks'
import { logger } from '@/lib/logger'
import type { Pattern } from '@/lib/graphql/queries'
import {
    BarChart3,
    TrendingUp,
    Brain,
    GitBranch,
    Clock,
    Zap,
    Target,
    RefreshCw
} from 'lucide-react'

// Extended pattern interface for UI
interface UIPattern {
    id: string
    type: string
    count: number
    confidence: number
    examples: string[]
    lastSeen: string
    trend: 'up' | 'down' | 'stable'
    category: 'problem' | 'solution' | 'architecture' | 'workflow'
}

interface PatternStats {
    totalPatterns: number
    activePatterns: number
    topCategories: Array<{ name: string; count: number }>
    recentActivity: number
}

export function PatternsDashboard() {
    const [selectedCategory, setSelectedCategory] = useState<string>('all')
    const repository = process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory'
    
    const { data, loading, error, refetch } = useGetPatterns(repository)
    
    // Log for debugging
    if (process.env.NODE_ENV === 'development') {
        console.log('Patterns data:', data)
        console.log('Patterns error:', error)
    }
    
    // Transform GraphQL patterns to UI patterns
    const patterns = useMemo<UIPattern[]>(() => {
        if (!data?.getPatterns) return []
        
        return data.getPatterns.map((pattern: Pattern, index: number) => {
            // Categorize based on pattern type
            let category: UIPattern['category'] = 'solution'
            const patternType = (pattern.type || '').toLowerCase()
            if (patternType.includes('error') || patternType.includes('issue') || patternType.includes('problem')) {
                category = 'problem'
            } else if (patternType.includes('architecture') || patternType.includes('design') || patternType.includes('pattern')) {
                category = 'architecture'
            } else if (patternType.includes('workflow') || patternType.includes('process') || patternType.includes('flow')) {
                category = 'workflow'
            }
            
            // Determine trend based on recent activity
            const daysSinceLastSeen = Math.floor(
                (Date.now() - new Date(pattern.lastSeen).getTime()) / (1000 * 60 * 60 * 24)
            )
            const trend: UIPattern['trend'] = daysSinceLastSeen < 7 ? 'up' : 
                                               daysSinceLastSeen > 30 ? 'down' : 'stable'
            
            return {
                id: `pattern-${index}`,
                type: pattern.type || '',
                count: pattern.count || 0,
                confidence: pattern.confidence || 0,
                examples: pattern.examples || [],
                lastSeen: pattern.lastSeen || new Date().toISOString(),
                trend,
                category
            }
        })
    }, [data])
    
    // Calculate stats from patterns
    const stats = useMemo<PatternStats>(() => {
        if (!patterns.length) {
            return {
                totalPatterns: 0,
                activePatterns: 0,
                topCategories: [],
                recentActivity: 0
            }
        }
        
        const activePatterns = patterns.filter(p => p.trend === 'up').length
        const recentActivity = patterns.filter(p => {
            const daysSince = Math.floor(
                (Date.now() - new Date(p.lastSeen).getTime()) / (1000 * 60 * 60 * 24)
            )
            return daysSince <= 7
        }).length
        
        // Count by category
        const categoryCount = patterns.reduce((acc, pattern) => {
            acc[pattern.category] = (acc[pattern.category] || 0) + 1
            return acc
        }, {} as Record<string, number>)
        
        const topCategories = Object.entries(categoryCount)
            .map(([name, count]) => ({ 
                name: name.charAt(0).toUpperCase() + name.slice(1) + 's', 
                count 
            }))
            .sort((a, b) => b.count - a.count)
        
        return {
            totalPatterns: patterns.length,
            activePatterns,
            topCategories,
            recentActivity
        }
    }, [patterns])
    
    const handleRefresh = () => {
        logger.info('Refreshing patterns', { 
            component: 'PatternsDashboard',
            repository 
        })
        refetch()
    }

    const filteredPatterns = useMemo(() => {
        return selectedCategory === 'all'
            ? patterns
            : patterns.filter(p => p.category === selectedCategory)
    }, [patterns, selectedCategory])

    const getTrendIcon = (trend: string) => {
        switch (trend) {
            case 'up': return <TrendingUp className="h-4 w-4 text-success" />
            case 'down': return <TrendingUp className="h-4 w-4 text-destructive rotate-180" />
            default: return <Target className="h-4 w-4 text-info" />
        }
    }

    const getCategoryColor = (category: string) => {
        switch (category) {
            case 'problem': return 'bg-destructive/10 text-destructive border-destructive/20'
            case 'solution': return 'bg-success-muted text-success border-success-muted'
            case 'architecture': return 'bg-info-muted text-info border-info-muted'
            case 'workflow': return 'bg-purple-muted text-purple border-purple-muted'
            default: return 'bg-muted text-muted-foreground border-border'
        }
    }

    if (loading) {
        return (
            <div className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                    {Array.from({ length: 4 }).map((_, i) => (
                        <Card key={i} className="animate-pulse">
                            <CardHeader className="pb-2">
                                <div className="h-4 bg-muted rounded w-3/4" />
                            </CardHeader>
                            <CardContent>
                                <div className="h-8 bg-muted rounded w-1/2" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )
    }
    
    if (error) {
        logger.error('Failed to load patterns', error, {
            component: 'PatternsDashboard',
            repository
        })
        
        return (
            <Card className="border-destructive">
                <CardHeader>
                    <CardTitle className="text-destructive flex items-center gap-2">
                        <Brain className="h-5 w-5" />
                        Error Loading Patterns
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground mb-4">
                        Unable to load pattern analysis data. Please try again.
                    </p>
                    <Button onClick={() => refetch()} variant="outline">
                        <RefreshCw className="h-4 w-4 mr-2" />
                        Retry
                    </Button>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            {/* Stats Overview */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Patterns</CardTitle>
                        <Brain className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{stats.totalPatterns}</div>
                        <p className="text-xs text-muted-foreground">
                            Discovered across all repositories
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active Patterns</CardTitle>
                        <Zap className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-success">{stats.activePatterns}</div>
                        <p className="text-xs text-muted-foreground">
                            Currently trending
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Recent Activity</CardTitle>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-info">{stats.recentActivity}</div>
                        <p className="text-xs text-muted-foreground">
                            New patterns this week
                        </p>
                    </CardContent>
                </Card>

                <Card className="border-2 hover:shadow-lg transition-all">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Repositories</CardTitle>
                        <GitBranch className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold text-purple">5</div>
                        <p className="text-xs text-muted-foreground">
                            Contributing to patterns
                        </p>
                    </CardContent>
                </Card>
            </div>

            {/* Pattern Categories */}
            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center gap-2">
                            <BarChart3 className="h-5 w-5" />
                            Pattern Analysis
                        </CardTitle>
                        <Button 
                            variant="outline" 
                            size="sm"
                            onClick={handleRefresh}
                            disabled={loading}
                        >
                            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                            Refresh
                        </Button>
                    </div>
                </CardHeader>
                <CardContent>
                    <Tabs value={selectedCategory} onValueChange={setSelectedCategory}>
                        <TabsList className="grid w-full grid-cols-5">
                            <TabsTrigger value="all">All</TabsTrigger>
                            <TabsTrigger value="problem">Problems</TabsTrigger>
                            <TabsTrigger value="solution">Solutions</TabsTrigger>
                            <TabsTrigger value="architecture">Architecture</TabsTrigger>
                            <TabsTrigger value="workflow">Workflow</TabsTrigger>
                        </TabsList>

                        <TabsContent value={selectedCategory} className="mt-6">
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                {filteredPatterns.map((pattern) => (
                                    <Card key={pattern.id} className="hover:shadow-md transition-all cursor-pointer">
                                        <CardHeader className="pb-3">
                                            <div className="flex items-start justify-between">
                                                <div className="flex-1">
                                                    <CardTitle className="text-lg mb-2">{pattern.type}</CardTitle>
                                                    <p className="text-sm text-muted-foreground line-clamp-2">
                                                        {pattern.examples[0] || 'Pattern detected in codebase'}
                                                    </p>
                                                </div>
                                                {getTrendIcon(pattern.trend)}
                                            </div>
                                        </CardHeader>
                                        <CardContent className="space-y-3">
                                            <div className="flex items-center justify-between">
                                                <Badge className={getCategoryColor(pattern.category)}>
                                                    {pattern.category}
                                                </Badge>
                                                <div className="text-sm text-muted-foreground">
                                                    {pattern.count} occurrences
                                                </div>
                                            </div>

                                            <div className="space-y-2">
                                                <div className="flex items-center justify-between text-sm">
                                                    <span>Confidence</span>
                                                    <span className="font-medium">{Math.round(pattern.confidence * 100)}%</span>
                                                </div>
                                                <div className="w-full bg-muted rounded-full h-2">
                                                    <div
                                                        className="bg-primary h-2 rounded-full transition-all"
                                                        style={{ width: `${pattern.confidence * 100}%` }}
                                                    />
                                                </div>
                                            </div>

                                            <div className="space-y-1">
                                                <div className="text-sm font-medium">Examples:</div>
                                                <div className="flex flex-wrap gap-1">
                                                    {pattern.examples.slice(0, 2).map((example, idx) => (
                                                        <Badge key={idx} variant="outline" className="text-xs">
                                                            {example}
                                                        </Badge>
                                                    ))}
                                                    {pattern.examples.length > 2 && (
                                                        <Badge variant="outline" className="text-xs">
                                                            +{pattern.examples.length - 2} more
                                                        </Badge>
                                                    )}
                                                </div>
                                            </div>
                                        </CardContent>
                                    </Card>
                                ))}
                            </div>

                            {filteredPatterns.length === 0 && (
                                <div className="text-center py-12">
                                    <Brain className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                                    <h3 className="text-lg font-medium mb-2">No patterns found</h3>
                                    <p className="text-muted-foreground">
                                        No patterns found for the selected category. Try a different filter.
                                    </p>
                                </div>
                            )}
                        </TabsContent>
                    </Tabs>
                </CardContent>
            </Card>
        </div>
    )
} 