'use client'

import React, { useState, useEffect, useMemo, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Activity,
  Database,
  Clock,
  TrendingUp,
  TrendingDown,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  XCircle
} from 'lucide-react'
import { 
  getPerformanceMonitor, 
  analyzeResourceTiming,
  markPerformance 
} from '@/lib/performance/web-vitals'
import { useServiceWorker } from '@/lib/sw-registration'
import { logCacheSize, cacheUtils } from '@/lib/apollo-client'
import { cn } from '@/lib/utils'

interface PerformanceData {
  webVitals: {
    fcp: number
    lcp: number
    fid: number
    cls: number
    ttfb: number
  }
  resourceTiming: {
    slowResources: Array<{
      name: string
      duration: number
    }>
    totalResources: number
    averageLoadTime: number
  }
  cacheMetrics: {
    apolloCacheSize: number
    serviceWorkerCacheSize: number
    cacheHitRate: number
  }
  networkMetrics: {
    requestCount: number
    errorRate: number
    averageResponseTime: number
  }
}

// Performance score calculation
function calculatePerformanceScore(data: PerformanceData): number {
  const scores = {
    fcp: data.webVitals.fcp < 1800 ? 90 : data.webVitals.fcp < 3000 ? 50 : 10,
    lcp: data.webVitals.lcp < 2500 ? 90 : data.webVitals.lcp < 4000 ? 50 : 10,
    fid: data.webVitals.fid < 100 ? 90 : data.webVitals.fid < 300 ? 50 : 10,
    cls: data.webVitals.cls < 0.1 ? 90 : data.webVitals.cls < 0.25 ? 50 : 10,
    ttfb: data.webVitals.ttfb < 800 ? 90 : data.webVitals.ttfb < 1800 ? 50 : 10,
  }

  return Math.round(Object.values(scores).reduce((sum, score) => sum + score, 0) / 5)
}

// Performance rating
function getPerformanceRating(score: number): { rating: string; color: string; icon: React.ReactNode } {
  if (score >= 80) {
    return { 
      rating: 'Excellent', 
      color: 'text-green-600', 
      icon: <CheckCircle className="h-5 w-5 text-green-600" /> 
    }
  } else if (score >= 60) {
    return { 
      rating: 'Good', 
      color: 'text-yellow-600', 
      icon: <AlertTriangle className="h-5 w-5 text-yellow-600" /> 
    }
  } else {
    return { 
      rating: 'Needs Improvement', 
      color: 'text-red-600', 
      icon: <XCircle className="h-5 w-5 text-red-600" /> 
    }
  }
}

// Metric card component
const MetricCard = ({ 
  title, 
  value, 
  unit, 
  target, 
  description,
  trend
}: {
  title: string
  value: number
  unit: string
  target: number
  description: string
  trend?: 'up' | 'down' | 'stable'
}) => {
  const isGood = value <= target
  const percentage = Math.min((value / target) * 100, 100)

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {trend && (
            <div className="flex items-center space-x-1">
              {trend === 'up' && <TrendingUp className="h-4 w-4 text-red-500" />}
              {trend === 'down' && <TrendingDown className="h-4 w-4 text-green-500" />}
              {trend === 'stable' && <Activity className="h-4 w-4 text-blue-500" />}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-baseline space-x-2">
          <span className={cn(
            "text-2xl font-bold",
            isGood ? "text-green-600" : "text-red-600"
          )}>
            {Math.round(value)}
          </span>
          <span className="text-sm text-muted-foreground">{unit}</span>
        </div>
        
        <Progress 
          value={percentage} 
          className={cn(
            "h-2",
            isGood ? "bg-green-100" : "bg-red-100"
          )}
        />
        
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>{description}</span>
          <span>Target: {target}{unit}</span>
        </div>
      </CardContent>
    </Card>
  )
}

// Resource timing table
const ResourceTimingTable = ({ resources }: { resources: Array<{ name: string; duration: number }> }) => {
  const sortedResources = useMemo(() => {
    return resources
      .sort((a, b) => b.duration - a.duration)
      .slice(0, 10) // Top 10 slowest resources
  }, [resources])

  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium">Slowest Resources</h4>
      <div className="space-y-1">
        {sortedResources.map((resource, index) => (
          <div 
            key={index}
            className="flex items-center justify-between p-2 bg-muted/50 rounded text-sm"
          >
            <span className="truncate flex-1" title={resource.name}>
              {resource.name.split('/').pop() || resource.name}
            </span>
            <span className="ml-2 font-mono">
              {Math.round(resource.duration)}ms
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export function PerformanceDashboard() {
  const [performanceData, setPerformanceData] = useState<PerformanceData | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const { cacheSize, clearCache } = useServiceWorker()

  const refreshData = useCallback(async () => {
    setIsRefreshing(true)
    markPerformance('performance-dashboard-refresh')

    try {
      const monitor = getPerformanceMonitor()
      const resourceTiming = analyzeResourceTiming()
      
      // Get Apollo cache size
      const apolloCacheSize = cacheUtils.getCacheSize()
      
      // Get performance metrics
      const metrics = monitor?.getMetrics() || []
      
      // Calculate web vitals averages
      const webVitals = {
        fcp: monitor?.getAverageMetric('FCP') || 0,
        lcp: monitor?.getAverageMetric('LCP') || 0,
        fid: monitor?.getAverageMetric('FID') || 0,
        cls: monitor?.getAverageMetric('CLS') || 0,
        ttfb: monitor?.getAverageMetric('TTFB') || 0,
      }

      // Calculate network metrics
      const networkMetrics = {
        requestCount: metrics.length,
        errorRate: metrics.filter(m => m.rating === 'poor').length / metrics.length * 100,
        averageResponseTime: metrics.reduce((sum, m) => sum + m.value, 0) / metrics.length || 0,
      }

      setPerformanceData({
        webVitals,
        resourceTiming,
        cacheMetrics: {
          apolloCacheSize,
          serviceWorkerCacheSize: cacheSize,
          cacheHitRate: 0, // This would need to be tracked separately
        },
        networkMetrics,
      })
    } catch (error) {
      console.error('Failed to refresh performance data:', error)
    } finally {
      setIsRefreshing(false)
    }
  }, [cacheSize])

  const handleClearCache = async () => {
    await Promise.all([
      clearCache(),
      cacheUtils.clearCache(),
    ])
    
    // Refresh data after clearing cache
    setTimeout(refreshData, 1000)
  }

  const forceGarbageCollection = () => {
    cacheUtils.forceGC()
    
    // Force browser garbage collection if available
    if ('gc' in window && typeof window.gc === 'function') {
      window.gc()
    }
    
    // Refresh data
    setTimeout(refreshData, 500)
  }

  useEffect(() => {
    refreshData()
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(refreshData, 30000)
    
    return () => clearInterval(interval)
  }, [refreshData])

  if (!performanceData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Performance Dashboard</CardTitle>
          <CardDescription>Loading performance metrics...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-32">
            <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    )
  }

  const performanceScore = calculatePerformanceScore(performanceData)
  const { rating, color, icon } = getPerformanceRating(performanceScore)

  return (
    <div className="space-y-6">
      {/* Performance Overview */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Performance Dashboard</CardTitle>
              <CardDescription>Real-time application performance metrics</CardDescription>
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={refreshData}
                disabled={isRefreshing}
              >
                <RefreshCw className={cn("h-4 w-4 mr-2", isRefreshing && "animate-spin")} />
                Refresh
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-4">
            {icon}
            <div>
              <div className="flex items-center space-x-2">
                <span className="text-3xl font-bold">{performanceScore}</span>
                <span className="text-muted-foreground">/100</span>
              </div>
              <p className={cn("text-sm font-medium", color)}>{rating}</p>
            </div>
            <div className="flex-1">
              <Progress value={performanceScore} className="h-2" />
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="vitals" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="vitals">Web Vitals</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
          <TabsTrigger value="cache">Cache</TabsTrigger>
          <TabsTrigger value="network">Network</TabsTrigger>
        </TabsList>

        {/* Web Vitals Tab */}
        <TabsContent value="vitals" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <MetricCard
              title="First Contentful Paint"
              value={performanceData.webVitals.fcp}
              unit="ms"
              target={1800}
              description="Time to first content"
            />
            <MetricCard
              title="Largest Contentful Paint"
              value={performanceData.webVitals.lcp}
              unit="ms"
              target={2500}
              description="Time to largest content"
            />
            <MetricCard
              title="First Input Delay"
              value={performanceData.webVitals.fid}
              unit="ms"
              target={100}
              description="Time to first interaction"
            />
            <MetricCard
              title="Cumulative Layout Shift"
              value={performanceData.webVitals.cls}
              unit=""
              target={0.1}
              description="Visual stability"
            />
            <MetricCard
              title="Time to First Byte"
              value={performanceData.webVitals.ttfb}
              unit="ms"
              target={800}
              description="Server response time"
            />
          </div>
        </TabsContent>

        {/* Resources Tab */}
        <TabsContent value="resources" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Clock className="h-5 w-5" />
                  <span>Resource Timing</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Total Resources</p>
                    <p className="text-2xl font-bold">
                      {performanceData.resourceTiming.totalResources}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Avg Load Time</p>
                    <p className="text-2xl font-bold">
                      {Math.round(performanceData.resourceTiming.averageLoadTime)}ms
                    </p>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Slow Resources</p>
                  <p className="text-2xl font-bold text-red-600">
                    {performanceData.resourceTiming.slowResources.length}
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Resource Details</CardTitle>
              </CardHeader>
              <CardContent>
                <ResourceTimingTable resources={performanceData.resourceTiming.slowResources} />
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Cache Tab */}
        <TabsContent value="cache" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Database className="h-5 w-5" />
                  <span>Cache Metrics</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="text-sm text-muted-foreground">Apollo Cache Size</p>
                  <p className="text-2xl font-bold">
                    {(performanceData.cacheMetrics.apolloCacheSize / 1024).toFixed(2)} KB
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Service Worker Cache</p>
                  <p className="text-2xl font-bold">
                    {(performanceData.cacheMetrics.serviceWorkerCacheSize / 1024 / 1024).toFixed(2)} MB
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Cache Hit Rate</p>
                  <p className="text-2xl font-bold">
                    {performanceData.cacheMetrics.cacheHitRate.toFixed(1)}%
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Cache Management</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <Button
                  variant="outline"
                  onClick={handleClearCache}
                  className="w-full"
                >
                  Clear All Caches
                </Button>
                <Button
                  variant="outline"
                  onClick={forceGarbageCollection}
                  className="w-full"
                >
                  Force Garbage Collection
                </Button>
                <Button
                  variant="outline"
                  onClick={logCacheSize}
                  className="w-full"
                >
                  Log Cache Details
                </Button>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Network Tab */}
        <TabsContent value="network" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <MetricCard
              title="Request Count"
              value={performanceData.networkMetrics.requestCount}
              unit=""
              target={100}
              description="Total requests made"
            />
            <MetricCard
              title="Error Rate"
              value={performanceData.networkMetrics.errorRate}
              unit="%"
              target={5}
              description="Failed requests percentage"
            />
            <MetricCard
              title="Avg Response Time"
              value={performanceData.networkMetrics.averageResponseTime}
              unit="ms"
              target={1000}
              description="Average API response time"
            />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}