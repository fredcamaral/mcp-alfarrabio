// Service Worker registration and management

interface ServiceWorkerMessage {
  type: string
  payload?: unknown
}

interface CacheSize {
  totalSize: number
  cacheNames: string[]
}

interface ServiceWorkerRegistrationWithSync extends ServiceWorkerRegistration {
  sync: {
    register(tag: string): Promise<void>
  }
}

class ServiceWorkerManager {
  private registration: ServiceWorkerRegistration | null = null
  private isSupported: boolean = false
  private isOnline: boolean = navigator.onLine

  constructor() {
    this.isSupported = 'serviceWorker' in navigator
    this.setupOnlineListener()
  }

  private setupOnlineListener() {
    window.addEventListener('online', () => {
      this.isOnline = true
      this.handleOnlineStatusChange(true)
    })

    window.addEventListener('offline', () => {
      this.isOnline = false
      this.handleOnlineStatusChange(false)
    })
  }

  private handleOnlineStatusChange(isOnline: boolean) {
    // Notify the application about online status changes
    window.dispatchEvent(new CustomEvent('sw:online-status', { 
      detail: { isOnline } 
    }))

    if (isOnline && this.registration) {
      // Trigger background sync when coming back online
      this.triggerBackgroundSync()
    }
  }

  async register(): Promise<ServiceWorkerRegistration | null> {
    if (!this.isSupported) {
      console.warn('Service Worker not supported')
      return null
    }

    try {
      console.log('🚀 Registering Service Worker...')
      
      this.registration = await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
        updateViaCache: 'none', // Always check for updates
      })

      console.log('✅ Service Worker registered:', this.registration.scope)

      // Handle updates
      this.setupUpdateHandling()

      // Handle messages from service worker
      this.setupMessageHandling()

      // Check for updates periodically
      this.setupPeriodicUpdateCheck()

      return this.registration
    } catch (error) {
      console.error('❌ Service Worker registration failed:', error)
      return null
    }
  }

  private setupUpdateHandling() {
    if (!this.registration) return

    this.registration.addEventListener('updatefound', () => {
      const newWorker = this.registration!.installing
      
      if (newWorker) {
        console.log('🔄 New Service Worker found, installing...')
        
        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            // New version available
            this.notifyUpdateAvailable()
          }
        })
      }
    })

    // Listen for controlling service worker changes
    navigator.serviceWorker.addEventListener('controllerchange', () => {
      console.log('🔄 Service Worker controller changed')
      window.location.reload()
    })
  }

  private setupMessageHandling() {
    navigator.serviceWorker.addEventListener('message', (event) => {
      const { type, data } = event.data

      switch (type) {
        case 'PERFORMANCE_METRIC':
          this.handlePerformanceMetric(data)
          break
        case 'CACHE_SIZE':
          this.handleCacheSize(data)
          break
        default:
          console.log('SW Message:', event.data)
      }
    })
  }

  private setupPeriodicUpdateCheck() {
    // Check for updates every hour
    setInterval(() => {
      this.checkForUpdates()
    }, 60 * 60 * 1000)
  }

  private handlePerformanceMetric(data: unknown) {
    // Forward performance metrics to analytics
    window.dispatchEvent(new CustomEvent('sw:performance-metric', { 
      detail: data 
    }))
  }

  private handleCacheSize(data: CacheSize) {
    // Report cache size for monitoring
    window.dispatchEvent(new CustomEvent('sw:cache-size', { 
      detail: data 
    }))
  }

  private notifyUpdateAvailable() {
    // Notify the application that an update is available
    window.dispatchEvent(new CustomEvent('sw:update-available'))
    
    if (process.env.NODE_ENV === 'development') {
      console.log('🆕 New version available! Reload to update.')
    }
  }

  async checkForUpdates(): Promise<void> {
    if (!this.registration) return

    try {
      await this.registration.update()
      console.log('🔍 Checked for Service Worker updates')
    } catch (error) {
      console.error('Failed to check for updates:', error)
    }
  }

  async skipWaiting(): Promise<void> {
    if (!this.registration || !this.registration.waiting) return

    this.sendMessage({ type: 'SKIP_WAITING' })
  }

  async clearCache(cacheNames?: string[]): Promise<void> {
    this.sendMessage({ 
      type: 'CLEAR_CACHE', 
      payload: { cacheNames } 
    })
  }

  async prefetchResources(urls: string[]): Promise<void> {
    this.sendMessage({ 
      type: 'PREFETCH_RESOURCES', 
      payload: { urls } 
    })
  }

  async getCacheSize(): Promise<number> {
    return new Promise((resolve) => {
      const channel = new MessageChannel()
      
      channel.port1.onmessage = (event) => {
        if (event.data.type === 'CACHE_SIZE') {
          resolve(event.data.size)
        }
      }

      this.sendMessage({ type: 'GET_CACHE_SIZE' }, [channel.port2])
    })
  }

  async triggerBackgroundSync(): Promise<void> {
    if (!this.registration || !('sync' in this.registration)) return

    try {
      await (this.registration as ServiceWorkerRegistrationWithSync).sync.register('background-sync')
      console.log('🔄 Background sync registered')
    } catch (error) {
      console.error('Background sync registration failed:', error)
    }
  }

  private sendMessage(message: ServiceWorkerMessage, transfer?: Transferable[]): void {
    if (!navigator.serviceWorker.controller) return

    if (transfer && transfer.length > 0) {
      navigator.serviceWorker.controller.postMessage(message, { transfer })
    } else {
      navigator.serviceWorker.controller.postMessage(message)
    }
  }

  async unregister(): Promise<boolean> {
    if (!this.registration) return false

    try {
      const result = await this.registration.unregister()
      console.log('🗑️ Service Worker unregistered')
      return result
    } catch (error) {
      console.error('Failed to unregister Service Worker:', error)
      return false
    }
  }

  getRegistration(): ServiceWorkerRegistration | null {
    return this.registration
  }

  isServiceWorkerSupported(): boolean {
    return this.isSupported
  }

  isServiceWorkerActive(): boolean {
    return !!(this.registration && this.registration.active)
  }

  getOnlineStatus(): boolean {
    return this.isOnline
  }

  // Prefetch critical resources for better performance
  async prefetchCriticalResources(): Promise<void> {
    const criticalUrls = [
      '/api/config',
      '/api/user/profile',
      '/_next/static/css/app.css', // Adjust based on your build output
    ]

    await this.prefetchResources(criticalUrls)
  }

  // Cache management utilities
  async warmupCache(): Promise<void> {
    console.log('🔥 Warming up cache...')
    
    const urls = [
      '/api/memories?limit=10',
      '/api/patterns',
      '/api/repositories',
    ]

    await this.prefetchResources(urls)
  }

  // Performance monitoring
  monitorCachePerformance(): void {
    // Monitor cache hit rates and performance
    let cacheHits = 0
    let cacheMisses = 0

    window.addEventListener('sw:performance-metric', ((event: CustomEvent) => {
      const { duration, status } = event.detail
      
      if (status === 200 && duration < 100) {
        cacheHits++
      } else {
        cacheMisses++
      }

      // Log cache hit rate every 100 requests
      if ((cacheHits + cacheMisses) % 100 === 0) {
        const hitRate = (cacheHits / (cacheHits + cacheMisses)) * 100
        console.log(`📊 Cache hit rate: ${hitRate.toFixed(2)}%`)
      }
    }) as EventListener)
  }
}

// Singleton instance
let swManager: ServiceWorkerManager | null = null

export function getServiceWorkerManager(): ServiceWorkerManager {
  if (!swManager) {
    swManager = new ServiceWorkerManager()
  }
  return swManager
}

// Convenience functions
export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  const manager = getServiceWorkerManager()
  const registration = await manager.register()
  
  if (registration) {
    // Start performance monitoring
    manager.monitorCachePerformance()
    
    // Prefetch critical resources
    await manager.prefetchCriticalResources()
    
    // Warm up cache in background
    setTimeout(() => {
      manager.warmupCache()
    }, 5000) // Wait 5 seconds before warming up cache
  }
  
  return registration
}

export async function updateServiceWorker(): Promise<void> {
  const manager = getServiceWorkerManager()
  await manager.skipWaiting()
}

export async function clearServiceWorkerCache(): Promise<void> {
  const manager = getServiceWorkerManager()
  await manager.clearCache()
}

export async function getServiceWorkerCacheSize(): Promise<number> {
  const manager = getServiceWorkerManager()
  return await manager.getCacheSize()
}

// React Hook for Service Worker integration
export function useServiceWorker() {
  const [isSupported, setIsSupported] = useState(false)
  const [isActive, setIsActive] = useState(false)
  const [isOnline, setIsOnline] = useState(navigator.onLine)
  const [updateAvailable, setUpdateAvailable] = useState(false)
  const [cacheSize, setCacheSize] = useState<number>(0)

  useEffect(() => {
    const manager = getServiceWorkerManager()
    
    setIsSupported(manager.isServiceWorkerSupported())
    setIsActive(manager.isServiceWorkerActive())
    setIsOnline(manager.getOnlineStatus())

    // Listen for Service Worker events
    const handleUpdateAvailable = () => setUpdateAvailable(true)
    const handleOnlineStatus = (event: CustomEvent) => setIsOnline(event.detail.isOnline)
    const handleCacheSize = (event: CustomEvent) => setCacheSize(event.detail.totalSize)

    window.addEventListener('sw:update-available', handleUpdateAvailable)
    window.addEventListener('sw:online-status', handleOnlineStatus as EventListener)
    window.addEventListener('sw:cache-size', handleCacheSize as EventListener)

    // Get initial cache size
    manager.getCacheSize().then(setCacheSize)

    return () => {
      window.removeEventListener('sw:update-available', handleUpdateAvailable)
      window.removeEventListener('sw:online-status', handleOnlineStatus as EventListener)
      window.removeEventListener('sw:cache-size', handleCacheSize as EventListener)
    }
  }, [])

  const applyUpdate = useCallback(async () => {
    await updateServiceWorker()
    setUpdateAvailable(false)
  }, [])

  const clearCache = useCallback(async () => {
    await clearServiceWorkerCache()
    const newSize = await getServiceWorkerCacheSize()
    setCacheSize(newSize)
  }, [])

  return {
    isSupported,
    isActive,
    isOnline,
    updateAvailable,
    cacheSize,
    applyUpdate,
    clearCache,
  }
}

// Type exports
export type { ServiceWorkerMessage, CacheSize }
export { ServiceWorkerManager }

// Import React hooks
import { useState, useEffect, useCallback } from 'react'