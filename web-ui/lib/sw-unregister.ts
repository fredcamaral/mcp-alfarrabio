/**
 * Service Worker Unregister Utility
 * 
 * Helper to forcefully unregister service workers and clear caches
 * Useful for development and debugging
 */

export async function unregisterServiceWorker(): Promise<void> {
  if ('serviceWorker' in navigator) {
    try {
      // Get all service worker registrations
      const registrations = await navigator.serviceWorker.getRegistrations()
      
      // Unregister all service workers
      for (const registration of registrations) {
        const success = await registration.unregister()
        console.log(`Service Worker unregistered: ${success}`)
      }
      
      // Clear all caches
      if ('caches' in window) {
        const cacheNames = await caches.keys()
        await Promise.all(
          cacheNames.map(cacheName => {
            console.log(`Deleting cache: ${cacheName}`)
            return caches.delete(cacheName)
          })
        )
      }
      
      console.log('✅ All service workers unregistered and caches cleared')
    } catch (error) {
      console.error('Failed to unregister service worker:', error)
    }
  }
}

// Add to window for easy console access in development
if (typeof window !== 'undefined' && process.env.NODE_ENV === 'development') {
  (window as Window & { unregisterSW?: typeof unregisterServiceWorker }).unregisterSW = unregisterServiceWorker
}