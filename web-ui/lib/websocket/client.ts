import { useEffect, useRef, useState, useCallback } from 'react'
import { useAppDispatch } from '@/store/store'
import { addNotification } from '@/store/slices/uiSlice'
import { addMemories, updateMemory, removeMemory } from '@/store/slices/memoriesSlice'
import { logger } from '@/lib/logger'
import type { ConversationChunk } from '@/types/memory'

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export interface WebSocketMessage {
    type: 'memory_created' | 'memory_updated' | 'memory_deleted' | 'pattern_detected' | 'system_status' | 'error'
    data: unknown
    timestamp: number
    id?: string
}

export interface ConnectionStats {
    uptime: number
    messagesReceived: number
    messagesSent: number
    lastActivity: number
    reconnectAttempts: number
}

interface WebSocketConfig {
    url?: string
    reconnectInterval?: number
    maxReconnectAttempts?: number
    heartbeatInterval?: number
    enableLogging?: boolean
}

// URL validation and construction utilities
export function isValidWebSocketURL(url: string): boolean {
    try {
        const parsed = new URL(url)
        return parsed.protocol === 'ws:' || parsed.protocol === 'wss:'
    } catch {
        return false
    }
}

export function constructWebSocketURL(): string {
    // In server environment, use environment variable
    if (typeof window === 'undefined') {
        const envUrl = process.env.NEXT_PUBLIC_WS_URL
        if (envUrl && isValidWebSocketURL(envUrl)) {
            return envUrl
        }
        return 'ws://localhost:9080/ws'
    }

    // In browser environment, construct from current location
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const hostname = window.location.hostname
    
    // Try environment variable first
    const envUrl = process.env.NEXT_PUBLIC_WS_URL
    if (envUrl && isValidWebSocketURL(envUrl)) {
        return envUrl
    }

    // Default ports based on environment
    const defaultPort = process.env.NODE_ENV === 'production' ? '443' : '9080'
    const port = process.env.NEXT_PUBLIC_WS_PORT || defaultPort
    
    // Don't include port for standard ports
    const needsPort = !(protocol === 'ws:' && port === '80') && !(protocol === 'wss:' && port === '443')
    const portPart = needsPort ? `:${port}` : ''
    
    return `${protocol}//${hostname}${portPart}/ws`
}

const DEFAULT_CONFIG: Required<WebSocketConfig> = {
    url: constructWebSocketURL(),
    reconnectInterval: 3000,
    maxReconnectAttempts: 10,
    heartbeatInterval: 30000,
    enableLogging: process.env.NODE_ENV === 'development'
}

export class WebSocketClient {
    private ws: WebSocket | null = null
    private config: Required<WebSocketConfig>
    private reconnectAttempts = 0
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null
    private heartbeatTimer: ReturnType<typeof setInterval> | null = null
    private isIntentionallyClosed = false

    // Event handlers
    private onStatusChange: (status: ConnectionStatus) => void = () => { }
    private onMessage: (message: WebSocketMessage) => void = () => { }
    private onStatsUpdate: (stats: ConnectionStats) => void = () => { }

    // Stats tracking
    private stats: ConnectionStats = {
        uptime: 0,
        messagesReceived: 0,
        messagesSent: 0,
        lastActivity: Date.now(),
        reconnectAttempts: 0
    }
    private startTime = Date.now()
    private statsTimer: ReturnType<typeof setInterval> | null = null

    constructor(config: WebSocketConfig = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config }
        
        // Validate and potentially correct the URL
        if (config.url && !isValidWebSocketURL(config.url)) {
            logger.warn(`Invalid WebSocket URL provided: ${config.url}. Falling back to default.`)
            this.config.url = DEFAULT_CONFIG.url
        }
        
        this.startStatsTracking()
    }

    connect(): void {
        if (this.ws?.readyState === WebSocket.OPEN) {
            return
        }

        // Validate URL before attempting connection
        if (!isValidWebSocketURL(this.config.url)) {
            this.log('Invalid WebSocket URL:', this.config.url)
            this.onStatusChange('error')
            return
        }

        this.isIntentionallyClosed = false
        this.onStatusChange('connecting')

        try {
            this.log('Attempting to connect to:', this.config.url)
            this.ws = new WebSocket(this.config.url)
            this.setupEventHandlers()
        } catch (error) {
            this.log('Connection failed:', error)
            this.onStatusChange('error')
            this.scheduleReconnect()
        }
    }

    disconnect(): void {
        this.isIntentionallyClosed = true
        this.clearTimers()

        if (this.ws) {
            this.ws.close(1000, 'Client disconnect')
            this.ws = null
        }

        this.onStatusChange('disconnected')
    }

    updateURL(newUrl: string): boolean {
        if (!isValidWebSocketURL(newUrl)) {
            this.log('Invalid URL provided for update:', newUrl)
            return false
        }

        const wasConnected = this.ws?.readyState === WebSocket.OPEN
        
        // Disconnect if currently connected
        if (wasConnected) {
            this.disconnect()
        }

        // Update the URL
        this.config.url = newUrl
        this.log('Updated WebSocket URL to:', newUrl)

        // Reconnect if we were previously connected
        if (wasConnected) {
            setTimeout(() => this.connect(), 100)
        }

        return true
    }

    send(message: Omit<WebSocketMessage, 'timestamp'>): boolean {
        if (this.ws?.readyState !== WebSocket.OPEN) {
            this.log('Cannot send message: WebSocket not connected')
            return false
        }

        try {
            const fullMessage: WebSocketMessage = {
                ...message,
                timestamp: Date.now()
            }

            this.ws.send(JSON.stringify(fullMessage))
            this.stats.messagesSent++
            this.stats.lastActivity = Date.now()
            this.log('Sent message:', fullMessage)
            return true
        } catch (error) {
            this.log('Failed to send message:', error)
            return false
        }
    }

    // Event handler setters
    setStatusHandler(handler: (status: ConnectionStatus) => void): void {
        this.onStatusChange = handler
    }

    setMessageHandler(handler: (message: WebSocketMessage) => void): void {
        this.onMessage = handler
    }

    setStatsHandler(handler: (stats: ConnectionStats) => void): void {
        this.onStatsUpdate = handler
    }

    getStats(): ConnectionStats {
        return {
            ...this.stats,
            uptime: Date.now() - this.startTime,
            reconnectAttempts: this.reconnectAttempts
        }
    }

    getConfig(): Required<WebSocketConfig> {
        return { ...this.config }
    }

    private setupEventHandlers(): void {
        if (!this.ws) return

        this.ws.onopen = () => {
            this.log('WebSocket connected')
            this.onStatusChange('connected')
            this.reconnectAttempts = 0
            this.startHeartbeat()

            // Send initial handshake
            this.send({
                type: 'system_status',
                data: { action: 'handshake', clientId: this.generateClientId() }
            })
        }

        this.ws.onmessage = (event) => {
            try {
                const message: WebSocketMessage = JSON.parse(event.data)
                this.stats.messagesReceived++
                this.stats.lastActivity = Date.now()

                this.log('Received message:', message)
                this.onMessage(message)
            } catch (error) {
                this.log('Failed to parse message:', error)
            }
        }

        this.ws.onclose = (event) => {
            this.log('WebSocket closed:', event.code, event.reason)
            this.clearTimers()

            if (!this.isIntentionallyClosed) {
                this.onStatusChange('disconnected')
                this.scheduleReconnect()
            }
        }

        this.ws.onerror = (error) => {
            this.log('WebSocket error:', error)
            this.onStatusChange('error')
        }
    }

    private scheduleReconnect(): void {
        if (this.isIntentionallyClosed || this.reconnectAttempts >= this.config.maxReconnectAttempts) {
            this.log('Max reconnect attempts reached or intentionally closed')
            return
        }

        this.reconnectAttempts++
        this.stats.reconnectAttempts = this.reconnectAttempts

        const delay = Math.min(this.config.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1), 30000)

        this.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`)

        this.reconnectTimer = setTimeout(() => {
            this.connect()
        }, delay)
    }

    private startHeartbeat(): void {
        this.heartbeatTimer = setInterval(() => {
            this.send({
                type: 'system_status',
                data: { action: 'ping' }
            })
        }, this.config.heartbeatInterval)
    }

    private startStatsTracking(): void {
        this.statsTimer = setInterval(() => {
            this.onStatsUpdate(this.getStats())
        }, 1000)
    }

    private clearTimers(): void {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
            this.reconnectTimer = null
        }

        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer)
            this.heartbeatTimer = null
        }
    }

    private generateClientId(): string {
        return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    }

    private log(...args: unknown[]): void {
        if (this.config.enableLogging) {
            logger.debug(`[WebSocket] ${args.join(' ')}`)
        }
    }

    destroy(): void {
        this.disconnect()
        if (this.statsTimer) {
            clearInterval(this.statsTimer)
            this.statsTimer = null
        }
    }
}

// React hook for WebSocket connection
export function useWebSocket(config: WebSocketConfig = {}) {
    const dispatch = useAppDispatch()
    const [status, setStatus] = useState<ConnectionStatus>('disconnected')
    const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
    const [stats, setStats] = useState<ConnectionStats>({
        uptime: 0,
        messagesReceived: 0,
        messagesSent: 0,
        lastActivity: Date.now(),
        reconnectAttempts: 0
    })

    const clientRef = useRef<WebSocketClient | null>(null)

    const handleMessage = useCallback((message: WebSocketMessage) => {
        setLastMessage(message)

        // Handle different message types
        switch (message.type) {
            case 'memory_created':
                dispatch(addMemories([message.data as ConversationChunk]))
                dispatch(addNotification({
                    type: 'success',
                    title: 'Memory Created',
                    message: 'A new memory has been added to your collection',
                    duration: 3000
                }))
                break

            case 'memory_updated':
                dispatch(updateMemory(message.data as ConversationChunk))
                dispatch(addNotification({
                    type: 'info',
                    title: 'Memory Updated',
                    message: 'A memory has been updated',
                    duration: 3000
                }))
                break

            case 'memory_deleted':
                dispatch(removeMemory((message.data as { id: string }).id))
                dispatch(addNotification({
                    type: 'warning',
                    title: 'Memory Deleted',
                    message: 'A memory has been removed',
                    duration: 3000
                }))
                break

            case 'pattern_detected':
                const patternData = message.data as { type?: string; pattern?: string }
                dispatch(addNotification({
                    type: 'info',
                    title: 'Pattern Detected',
                    message: `New pattern found: ${patternData.type || patternData.pattern || 'Unknown pattern'}`,
                    duration: 5000
                }))
                break

            case 'error':
                const errorData = message.data as { message?: string }
                dispatch(addNotification({
                    type: 'error',
                    title: 'WebSocket Error',
                    message: errorData.message || 'An error occurred',
                    duration: 5000
                }))
                break
        }
    }, [dispatch])

    useEffect(() => {
        clientRef.current = new WebSocketClient(config)

        clientRef.current.setStatusHandler(setStatus)
        clientRef.current.setMessageHandler(handleMessage)
        clientRef.current.setStatsHandler(setStats)

        // Auto-connect
        clientRef.current.connect()

        return () => {
            clientRef.current?.destroy()
        }
    }, [config, handleMessage])

    const send = useCallback((message: Omit<WebSocketMessage, 'timestamp'>) => {
        return clientRef.current?.send(message) || false
    }, [])

    const connect = useCallback(() => {
        clientRef.current?.connect()
    }, [])

    const disconnect = useCallback(() => {
        clientRef.current?.disconnect()
    }, [])

    const updateURL = useCallback((newUrl: string) => {
        return clientRef.current?.updateURL(newUrl) || false
    }, [])

    return {
        status,
        lastMessage,
        stats,
        send,
        connect,
        disconnect,
        updateURL,
        isConnected: status === 'connected',
        currentURL: clientRef.current?.getConfig().url || config.url || DEFAULT_CONFIG.url
    }
} 