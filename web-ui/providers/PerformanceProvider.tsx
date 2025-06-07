'use client'

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { initializePerformanceMonitoring } from '@/lib/performance/web-vitals'
import { registerServiceWorker } from '@/lib/sw-registration'
import { markPerformance } from '@/lib/performance/web-vitals'

interface PerformanceContextType {
  isInitialized: boolean
  performanceScore: number | null
  serviceWorkerReady: boolean
}

const PerformanceContext = createContext<PerformanceContextType>({
  isInitialized: false,
  performanceScore: null,
  serviceWorkerReady: false,
})

export const usePerformance = () => {
  const context = useContext(PerformanceContext)
  if (!context) {
    throw new Error('usePerformance must be used within a PerformanceProvider')
  }
  return context
}

interface PerformanceProviderProps {
  children: ReactNode
}

export function PerformanceProvider({ children }: PerformanceProviderProps) {
  const [isInitialized, setIsInitialized] = useState(false)
  const [performanceScore, setPerformanceScore] = useState<number | null>(null)
  const [serviceWorkerReady, setServiceWorkerReady] = useState(false)

  useEffect(() => {
    let mounted = true

    const initializePerformance = async () => {
      try {
        markPerformance('performance-provider-init-start')

        // Initialize performance monitoring
        const monitor = initializePerformanceMonitoring('/api/performance/metrics')
        
        // Register service worker
        const registration = await registerServiceWorker()
        
        if (mounted) {
          setServiceWorkerReady(!!registration)
          setIsInitialized(true)
          markPerformance('performance-provider-init-complete')
        }

        // Calculate initial performance score
        setTimeout(() => {
          if (mounted && monitor) {
            const report = monitor.generateReport()
            const score = calculatePerformanceScore(report.summary)
            setPerformanceScore(score)
          }
        }, 5000) // Wait 5 seconds for initial metrics

      } catch (error) {
        console.error('Failed to initialize performance monitoring:', error)
        if (mounted) {
          setIsInitialized(true) // Set as initialized even if some parts failed
        }
      }
    }

    initializePerformance()

    return () => {
      mounted = false
    }
  }, [])

  // Listen for performance updates
  useEffect(() => {
    const handlePerformanceUpdate = () => {
      // Recalculate performance score when metrics update
      const monitor = initializePerformanceMonitoring()
      if (monitor) {
        const report = monitor.generateReport()
        const score = calculatePerformanceScore(report.summary)
        setPerformanceScore(score)
      }
    }

    // Listen for web vitals updates
    window.addEventListener('performance:update', handlePerformanceUpdate)
    
    return () => {
      window.removeEventListener('performance:update', handlePerformanceUpdate)
    }
  }, [])

  // Performance score calculation
  const calculatePerformanceScore = (summary: Record<string, any>): number => {
    const weights = {
      FCP: 0.15,
      LCP: 0.25,
      FID: 0.25,
      CLS: 0.25,
      TTFB: 0.10,
    }

    let totalScore = 0
    let totalWeight = 0

    Object.entries(weights).forEach(([metric, weight]) => {
      const data = summary[metric]
      if (data) {
        let score = 0
        const value = data.average

        // Score based on Google's thresholds
        switch (metric) {
          case 'FCP':
            score = value < 1800 ? 100 : value < 3000 ? 50 : 0
            break
          case 'LCP':
            score = value < 2500 ? 100 : value < 4000 ? 50 : 0
            break
          case 'FID':
            score = value < 100 ? 100 : value < 300 ? 50 : 0
            break
          case 'CLS':
            score = value < 0.1 ? 100 : value < 0.25 ? 50 : 0
            break
          case 'TTFB':
            score = value < 800 ? 100 : value < 1800 ? 50 : 0
            break
        }

        totalScore += score * weight
        totalWeight += weight
      }
    })

    return totalWeight > 0 ? Math.round(totalScore / totalWeight) : 0
  }

  const value: PerformanceContextType = {
    isInitialized,
    performanceScore,
    serviceWorkerReady,
  }

  return (
    <PerformanceContext.Provider value={value}>
      {children}
    </PerformanceContext.Provider>
  )
}

// Performance indicator component
export function PerformanceIndicator() {
  const { performanceScore, serviceWorkerReady } = usePerformance()

  if (process.env.NODE_ENV !== 'development') {
    return null // Only show in development
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2">
      {performanceScore !== null && (
        <div className={`px-3 py-2 rounded-lg shadow-lg text-sm font-medium ${
          performanceScore >= 80 
            ? 'bg-green-100 text-green-800 border border-green-200'
            : performanceScore >= 60
            ? 'bg-yellow-100 text-yellow-800 border border-yellow-200'
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          Performance: {performanceScore}/100
        </div>
      )}
      
      {serviceWorkerReady && (
        <div className="px-3 py-2 rounded-lg shadow-lg text-sm font-medium bg-blue-100 text-blue-800 border border-blue-200">
          SW: Active
        </div>
      )}
    </div>
  )
}

// Performance metrics hook for components
export function usePerformanceMetrics() {
  const [metrics, setMetrics] = useState<any[]>([])

  useEffect(() => {
    const monitor = initializePerformanceMonitoring()
    if (monitor) {
      setMetrics(monitor.getMetrics())
    }

    const handleUpdate = () => {
      if (monitor) {
        setMetrics(monitor.getMetrics())
      }
    }

    window.addEventListener('performance:update', handleUpdate)
    
    return () => {
      window.removeEventListener('performance:update', handleUpdate)
    }
  }, [])

  return metrics
}

export default PerformanceProvider