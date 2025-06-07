import { useCallback, useEffect, useRef, useState } from 'react'
import { markPerformance, measurePerformance } from '@/lib/performance/web-vitals'

/**
 * Hook for optimized component state management with performance tracking
 */
export function useOptimizedState<T>(
  initialValue: T,
  componentName?: string
): [T, (value: T | ((prev: T) => T)) => void] {
  const [state, setState] = useState(initialValue)
  const renderCount = useRef(0)
  const stateUpdateCount = useRef(0)

  const optimizedSetState = useCallback((value: T | ((prev: T) => T)) => {
    stateUpdateCount.current++
    
    if (componentName && process.env.NODE_ENV === 'development') {
      markPerformance(`${componentName}-state-update-${stateUpdateCount.current}`)
    }

    setState(value)
  }, [componentName])

  useEffect(() => {
    renderCount.current++
    
    if (componentName && process.env.NODE_ENV === 'development') {
      markPerformance(`${componentName}-render-${renderCount.current}`)
      
      if (renderCount.current > 1) {
        measurePerformance(
          `${componentName}-render-duration`,
          `${componentName}-render-${renderCount.current - 1}`,
          `${componentName}-render-${renderCount.current}`
        )
      }
    }
  })

  return [state, optimizedSetState]
}

/**
 * Hook for debounced values with performance optimization
 */
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value)
  const timeoutRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    timeoutRef.current = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [value, delay])

  return debouncedValue
}

/**
 * Hook for throttled callbacks
 */
export function useThrottle<T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): T {
  const lastCall = useRef<number>(0)
  const timeoutRef = useRef<NodeJS.Timeout | null>(null)

  return useCallback((...args: Parameters<T>) => {
    const now = Date.now()
    
    if (now - lastCall.current >= delay) {
      lastCall.current = now
      return callback(...args)
    } else {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      
      timeoutRef.current = setTimeout(() => {
        lastCall.current = Date.now()
        callback(...args)
      }, delay - (now - lastCall.current))
    }
  }, [callback, delay]) as T
}

/**
 * Hook for intersection observer with performance optimization
 */
export function useIntersectionObserver(
  options?: IntersectionObserverInit
): [React.RefCallback<Element>, boolean, IntersectionObserverEntry | null] {
  const [isIntersecting, setIsIntersecting] = useState(false)
  const [entry, setEntry] = useState<IntersectionObserverEntry | null>(null)
  const observerRef = useRef<IntersectionObserver | null>(null)

  const ref = useCallback((element: Element | null) => {
    if (observerRef.current) {
      observerRef.current.disconnect()
    }

    if (element) {
      observerRef.current = new IntersectionObserver(
        ([entry]) => {
          setIsIntersecting(entry.isIntersecting)
          setEntry(entry)
        },
        {
          threshold: 0.1,
          rootMargin: '50px',
          ...options,
        }
      )

      observerRef.current.observe(element)
    }
  }, [options])

  useEffect(() => {
    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect()
      }
    }
  }, [])

  return [ref, isIntersecting, entry]
}

/**
 * Hook for virtual scrolling optimization
 */
export function useVirtualList<T>({
  items,
  itemHeight,
  containerHeight,
  overscan = 5,
}: {
  items: T[]
  itemHeight: number
  containerHeight: number
  overscan?: number
}) {
  const [scrollTop, setScrollTop] = useState(0)

  const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan)
  const endIndex = Math.min(
    items.length - 1,
    Math.ceil((scrollTop + containerHeight) / itemHeight) + overscan
  )

  const visibleItems = items.slice(startIndex, endIndex + 1).map((item, index) => ({
    item,
    index: startIndex + index,
  }))

  const totalHeight = items.length * itemHeight

  const onScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop)
  }, [])

  return {
    visibleItems,
    totalHeight,
    startIndex,
    endIndex,
    onScroll,
  }
}

/**
 * Hook for memory leak prevention in async operations
 */
export function useAsyncOperation() {
  const isMountedRef = useRef(true)

  useEffect(() => {
    return () => {
      isMountedRef.current = false
    }
  }, [])

  const executeAsync = useCallback(async <T>(
    asyncFn: () => Promise<T>,
    onSuccess?: (result: T) => void,
    onError?: (error: Error) => void
  ) => {
    try {
      const result = await asyncFn()
      if (isMountedRef.current && onSuccess) {
        onSuccess(result)
      }
      return result
    } catch (error) {
      if (isMountedRef.current && onError) {
        onError(error as Error)
      }
      throw error
    }
  }, [])

  return { executeAsync, isMounted: () => isMountedRef.current }
}

/**
 * Hook for performance monitoring of component lifecycle
 */
export function useComponentPerformance(componentName: string) {
  const mountTime = useRef<number>(Date.now())
  const renderCount = useRef<number>(0)
  const updateCount = useRef<number>(0)

  useEffect(() => {
    markPerformance(`${componentName}-mount`)
    
    return () => {
      const lifetimeDuration = Date.now() - mountTime.current
      markPerformance(`${componentName}-unmount`, {
        lifetimeDuration,
        renderCount: renderCount.current,
        updateCount: updateCount.current,
      })
    }
  }, [componentName])

  useEffect(() => {
    renderCount.current++
    markPerformance(`${componentName}-render-${renderCount.current}`)
  })

  const trackUpdate = useCallback((updateType: string) => {
    updateCount.current++
    markPerformance(`${componentName}-update-${updateType}`)
  }, [componentName])

  return { trackUpdate }
}

/**
 * Hook for optimized event listeners
 */
export function useOptimizedEventListener<K extends keyof WindowEventMap>(
  eventName: K,
  handler: (event: WindowEventMap[K]) => void,
  options?: AddEventListenerOptions
) {
  const savedHandler = useRef(handler)

  useEffect(() => {
    savedHandler.current = handler
  }, [handler])

  useEffect(() => {
    const eventListener = (event: WindowEventMap[K]) => {
      savedHandler.current(event)
    }

    const optimizedOptions = {
      passive: true,
      ...options,
    }

    window.addEventListener(eventName, eventListener, optimizedOptions)

    return () => {
      window.removeEventListener(eventName, eventListener, optimizedOptions)
    }
  }, [eventName, options])
}

/**
 * Hook for RAF-based animations with performance optimization
 */
export function useAnimationFrame(callback: (deltaTime: number) => void, deps: any[] = []) {
  const requestRef = useRef<number>()
  const previousTimeRef = useRef<number>()
  const callbackRef = useRef(callback)

  useEffect(() => {
    callbackRef.current = callback
  }, [callback])

  const animate = useCallback((time: number) => {
    if (previousTimeRef.current !== undefined) {
      const deltaTime = time - previousTimeRef.current
      callbackRef.current(deltaTime)
    }
    previousTimeRef.current = time
    requestRef.current = requestAnimationFrame(animate)
  }, [])

  useEffect(() => {
    requestRef.current = requestAnimationFrame(animate)
    return () => {
      if (requestRef.current) {
        cancelAnimationFrame(requestRef.current)
      }
    }
  }, deps)
}

/**
 * Hook for image lazy loading with performance optimization
 */
export function useLazyImage(src: string, placeholder?: string) {
  const [imageSrc, setImageSrc] = useState(placeholder || '')
  const [isLoaded, setIsLoaded] = useState(false)
  const [isError, setIsError] = useState(false)
  const [ref, isIntersecting] = useIntersectionObserver({
    threshold: 0.1,
    rootMargin: '50px',
  })

  useEffect(() => {
    if (isIntersecting && src && !isLoaded && !isError) {
      const img = new Image()
      
      img.onload = () => {
        setImageSrc(src)
        setIsLoaded(true)
      }
      
      img.onerror = () => {
        setIsError(true)
      }
      
      img.src = src
    }
  }, [isIntersecting, src, isLoaded, isError])

  return { ref, imageSrc, isLoaded, isError }
}