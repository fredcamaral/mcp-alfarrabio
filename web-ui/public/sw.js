// Service Worker for caching and performance optimization
const CACHE_NAME = 'lerian-mcp-memory-v1.0.3'
const STATIC_CACHE_NAME = 'lerian-static-v1.0.3'
const DYNAMIC_CACHE_NAME = 'lerian-dynamic-v1.0.3'

// Resources to cache immediately
const STATIC_ASSETS = [
  '/',
  '/manifest.json',
  // Remove directories and non-existent files to prevent cache errors
]

// API routes to cache with different strategies
const API_ROUTES = {
  CACHE_FIRST: [
    '/api/config',
    '/api/user/profile',
  ],
  NETWORK_FIRST: [
    '/api/memories',
    '/api/search',
    '/graphql',
  ],
  STALE_WHILE_REVALIDATE: [
    '/api/patterns',
    '/api/repositories',
  ]
}

// Cache duration configurations
const CACHE_DURATIONS = {
  STATIC: 365 * 24 * 60 * 60 * 1000, // 1 year
  API: 24 * 60 * 60 * 1000, // 1 day
  DYNAMIC: 7 * 24 * 60 * 60 * 1000, // 1 week
}

// Install event - cache static assets
self.addEventListener('install', (event) => {
  console.log('🚀 Service Worker installing...')
  
  event.waitUntil(
    Promise.all([
      caches.open(STATIC_CACHE_NAME).then((cache) => {
        // Try to cache each asset individually to handle failures gracefully
        return Promise.all(
          STATIC_ASSETS.map(asset => 
            cache.add(asset).catch(err => {
              console.warn(`Failed to cache ${asset}:`, err)
              // Continue with other assets even if one fails
            })
          )
        )
      }),
      self.skipWaiting() // Force activation
    ])
  )
})

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('✅ Service Worker activated')
  
  event.waitUntil(
    Promise.all([
      // Clean up old caches
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter((cacheName) => {
              return cacheName !== STATIC_CACHE_NAME && 
                     cacheName !== DYNAMIC_CACHE_NAME &&
                     cacheName !== CACHE_NAME
            })
            .map((cacheName) => caches.delete(cacheName))
        )
      }),
      self.clients.claim() // Take control of all pages
    ])
  )
})

// Fetch event - implement caching strategies
self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // Skip non-GET requests
  if (request.method !== 'GET') {
    return
  }

  // Skip chrome-extension and dev server requests
  if (url.protocol === 'chrome-extension:' || url.hostname === 'localhost' && url.port === '24678') {
    return
  }

  // Handle different types of requests
  if (isStaticAsset(url)) {
    event.respondWith(handleStaticAsset(request))
  } else if (isApiRoute(url)) {
    event.respondWith(handleApiRoute(request, url))
  } else if (isNavigationRequest(request)) {
    event.respondWith(handleNavigationRequest(request))
  } else {
    event.respondWith(handleDynamicRequest(request))
  }
})

// Check if request is for static asset
function isStaticAsset(url) {
  return url.pathname.startsWith('/_next/static/') ||
         url.pathname.startsWith('/static/') ||
         url.pathname.match(/\.(js|css|png|jpg|jpeg|gif|webp|svg|ico|woff|woff2|ttf|otf)$/)
}

// Check if request is for API route
function isApiRoute(url) {
  return url.pathname.startsWith('/api/') || 
         url.pathname.startsWith('/graphql')
}

// Check if request is navigation request
function isNavigationRequest(request) {
  return request.mode === 'navigate'
}

// Handle static assets with cache-first strategy
async function handleStaticAsset(request) {
  try {
    const cache = await caches.open(STATIC_CACHE_NAME)
    const cachedResponse = await cache.match(request)
    
    if (cachedResponse) {
      // Check if cache is still valid
      const cacheTime = new Date(cachedResponse.headers.get('sw-cache-time') || 0)
      const now = new Date()
      
      if (now - cacheTime < CACHE_DURATIONS.STATIC) {
        return cachedResponse
      }
    }
    
    // Fetch from network and update cache
    const response = await fetch(request)
    
    if (response.ok) {
      // Clone the response to read the body
      const responseToCache = response.clone()
      
      // Create a new response with additional headers for caching
      const responseBody = await responseToCache.arrayBuffer()
      const newHeaders = new Headers(response.headers)
      newHeaders.set('sw-cache-time', new Date().toISOString())
      
      const cachedResponse = new Response(responseBody, {
        status: response.status,
        statusText: response.statusText,
        headers: newHeaders
      })
      
      // Store in cache asynchronously
      cache.put(request, cachedResponse).catch(err => {
        console.warn('Failed to cache response:', err)
      })
    }
    
    return response
  } catch (error) {
    console.error('Static asset fetch failed:', error)
    
    // Return cached version if available
    const cache = await caches.open(STATIC_CACHE_NAME)
    const cachedResponse = await cache.match(request)
    
    if (cachedResponse) {
      return cachedResponse
    }
    
    // Return fallback
    return new Response('Asset not available offline', { status: 503 })
  }
}

// Handle API routes with different strategies
async function handleApiRoute(request, url) {
  const route = url.pathname
  
  // Determine caching strategy
  if (API_ROUTES.CACHE_FIRST.some(pattern => route.includes(pattern))) {
    return handleCacheFirst(request, DYNAMIC_CACHE_NAME)
  } else if (API_ROUTES.NETWORK_FIRST.some(pattern => route.includes(pattern))) {
    return handleNetworkFirst(request, DYNAMIC_CACHE_NAME)
  } else if (API_ROUTES.STALE_WHILE_REVALIDATE.some(pattern => route.includes(pattern))) {
    return handleStaleWhileRevalidate(request, DYNAMIC_CACHE_NAME)
  } else {
    // Default to network first for API routes
    return handleNetworkFirst(request, DYNAMIC_CACHE_NAME)
  }
}

// Handle navigation requests
async function handleNavigationRequest(request) {
  try {
    // Try network first for navigation
    const response = await fetch(request)
    
    if (response.ok) {
      // Cache successful navigation responses
      const cache = await caches.open(DYNAMIC_CACHE_NAME)
      cache.put(request, response.clone())
    }
    
    return response
  } catch (error) {
    console.error('Navigation fetch failed:', error)
    
    // Try cache
    const cache = await caches.open(DYNAMIC_CACHE_NAME)
    const cachedResponse = await cache.match(request)
    
    if (cachedResponse) {
      return cachedResponse
    }
    
    // Return offline page
    return caches.match('/offline') || new Response('Offline', { status: 503 })
  }
}

// Handle dynamic requests
async function handleDynamicRequest(request) {
  return handleStaleWhileRevalidate(request, DYNAMIC_CACHE_NAME)
}

// Cache-first strategy
async function handleCacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cachedResponse = await cache.match(request)
  
  if (cachedResponse) {
    // Check cache validity
    const cacheTime = new Date(cachedResponse.headers.get('sw-cache-time') || 0)
    const now = new Date()
    
    if (now - cacheTime < CACHE_DURATIONS.API) {
      return cachedResponse
    }
  }
  
  try {
    const response = await fetch(request)
    
    if (response.ok) {
      // Create a new response with cache time header
      const responseBody = await response.clone().arrayBuffer()
      const newHeaders = new Headers(response.headers)
      newHeaders.set('sw-cache-time', new Date().toISOString())
      
      const cachedResponse = new Response(responseBody, {
        status: response.status,
        statusText: response.statusText,
        headers: newHeaders
      })
      
      cache.put(request, cachedResponse).catch(err => {
        console.warn('Failed to cache response:', err)
      })
    }
    
    return response
  } catch (error) {
    if (cachedResponse) {
      return cachedResponse
    }
    throw error
  }
}

// Network-first strategy
async function handleNetworkFirst(request, cacheName) {
  try {
    const response = await fetch(request)
    
    if (response.ok) {
      const cache = await caches.open(cacheName)
      
      // Create a new response with cache time header
      const responseBody = await response.clone().arrayBuffer()
      const newHeaders = new Headers(response.headers)
      newHeaders.set('sw-cache-time', new Date().toISOString())
      
      const cachedResponse = new Response(responseBody, {
        status: response.status,
        statusText: response.statusText,
        headers: newHeaders
      })
      
      cache.put(request, cachedResponse).catch(err => {
        console.warn('Failed to cache response:', err)
      })
    }
    
    return response
  } catch (error) {
    const cache = await caches.open(cacheName)
    const cachedResponse = await cache.match(request)
    
    if (cachedResponse) {
      return cachedResponse
    }
    
    throw error
  }
}

// Stale-while-revalidate strategy
async function handleStaleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cachedResponse = await cache.match(request)
  
  // Start fetch in background
  const fetchPromise = fetch(request).then(async (response) => {
    if (response.ok) {
      // Create a new response with cache time header
      const responseBody = await response.clone().arrayBuffer()
      const newHeaders = new Headers(response.headers)
      newHeaders.set('sw-cache-time', new Date().toISOString())
      
      const cachedResponse = new Response(responseBody, {
        status: response.status,
        statusText: response.statusText,
        headers: newHeaders
      })
      
      cache.put(request, cachedResponse).catch(err => {
        console.warn('Failed to cache response:', err)
      })
    }
    return response
  }).catch(() => {
    // Fail silently for background updates
  })
  
  // Return cached version immediately if available
  if (cachedResponse) {
    // Check if cache is stale
    const cacheTime = new Date(cachedResponse.headers.get('sw-cache-time') || 0)
    const now = new Date()
    
    if (now - cacheTime < CACHE_DURATIONS.DYNAMIC) {
      return cachedResponse
    }
  }
  
  // Wait for network response if no cache or cache is stale
  return await fetchPromise || cachedResponse || new Response('Service unavailable', { status: 503 })
}

// Background sync for offline actions
self.addEventListener('sync', (event) => {
  if (event.tag === 'background-sync') {
    event.waitUntil(handleBackgroundSync())
  }
})

async function handleBackgroundSync() {
  console.log('🔄 Background sync triggered')
  
  // Handle any pending offline actions
  // This could include syncing memory updates, search queries, etc.
  
  try {
    // Example: Sync pending memory updates
    const pendingUpdates = await getStoredPendingUpdates()
    
    for (const update of pendingUpdates) {
      try {
        await fetch(update.url, {
          method: update.method,
          headers: update.headers,
          body: update.body,
        })
        
        // Remove from pending updates on success
        await removePendingUpdate(update.id)
      } catch (error) {
        console.error('Failed to sync update:', error)
      }
    }
  } catch (error) {
    console.error('Background sync failed:', error)
  }
}

// Message handling for cache management
self.addEventListener('message', (event) => {
  const { type, payload } = event.data
  
  switch (type) {
    case 'SKIP_WAITING':
      self.skipWaiting()
      break
      
    case 'CLEAR_CACHE':
      handleClearCache(payload.cacheNames)
      break
      
    case 'PREFETCH_RESOURCES':
      handlePrefetchResources(payload.urls)
      break
      
    case 'GET_CACHE_SIZE':
      handleGetCacheSize()
        .then(size => event.ports[0].postMessage({ type: 'CACHE_SIZE', size }))
      break
  }
})

async function handleClearCache(cacheNames = []) {
  if (cacheNames.length === 0) {
    cacheNames = await caches.keys()
  }
  
  await Promise.all(
    cacheNames.map(cacheName => caches.delete(cacheName))
  )
  
  console.log('🗑️ Cache cleared:', cacheNames)
}

async function handlePrefetchResources(urls = []) {
  const cache = await caches.open(DYNAMIC_CACHE_NAME)
  
  await Promise.all(
    urls.map(async (url) => {
      try {
        const response = await fetch(url)
        if (response.ok) {
          await cache.put(url, response)
        }
      } catch (error) {
        console.warn('Failed to prefetch:', url, error)
      }
    })
  )
  
  console.log('🔥 Resources prefetched:', urls.length)
}

async function handleGetCacheSize() {
  const cacheNames = await caches.keys()
  let totalSize = 0
  
  for (const cacheName of cacheNames) {
    const cache = await caches.open(cacheName)
    const requests = await cache.keys()
    
    for (const request of requests) {
      const response = await cache.match(request)
      if (response) {
        const blob = await response.blob()
        totalSize += blob.size
      }
    }
  }
  
  return totalSize
}

// Utility functions for offline storage
async function getStoredPendingUpdates() {
  // This would typically use IndexedDB for persistence
  // For now, return empty array
  return []
}

async function removePendingUpdate(id) {
  // Remove update from IndexedDB
  console.log('Removing pending update:', id)
}

// Performance monitoring
self.addEventListener('fetch', (event) => {
  // Track performance metrics for important requests
  if (event.request.url.includes('/api/') || event.request.url.includes('/graphql')) {
    const startTime = performance.now()
    
    event.respondWith(
      fetch(event.request).then((response) => {
        const duration = performance.now() - startTime
        
        // Report metrics to analytics
        self.clients.matchAll().then((clients) => {
          clients.forEach((client) => {
            client.postMessage({
              type: 'PERFORMANCE_METRIC',
              data: {
                url: event.request.url,
                method: event.request.method,
                duration,
                status: response.status,
                timestamp: Date.now(),
              }
            })
          })
        })
        
        return response
      })
    )
  }
})

console.log('🚀 Service Worker loaded successfully')