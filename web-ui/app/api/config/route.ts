import { NextRequest, NextResponse } from 'next/server'
import { configSchemas, validateRequest } from '@/lib/validation-schemas'
import { logger } from '@/lib/logger'

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080'

export async function GET() {
    try {
        // Fetch current configuration from backend
        const response = await fetch(`${BACKEND_URL}/api/config`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        })

        if (!response.ok) {
            throw new Error(`Backend responded with ${response.status}`)
        }

        const config = await response.json()
        return NextResponse.json(config)
    } catch (error) {
        logger.error('Error fetching configuration', error as Error, { 
            component: 'ConfigAPI',
            action: 'GET' 
        })

        // Return default configuration if backend is not available
        const defaultConfig = {
            host: 'localhost',
            port: 9080,
            protocol: 'http' as const,
            transportProtocols: {
                http: true,
                websocket: true,
                grpc: false,
            },
            vectorDb: {
                provider: 'qdrant' as const,
                host: 'localhost',
                port: 6333,
                collection: 'claude_memory',
                dimension: 1536,
            },
            openai: {
                // NEVER expose API keys to frontend
                model: 'text-embedding-ada-002',
                maxTokens: 8192,
                temperature: 0.1,
                timeout: 30000,
            },
            cacheEnabled: true,
            realtimeEnabled: true,
            analyticsEnabled: true,
            debugMode: false,
            authEnabled: false,
        }

        return NextResponse.json(defaultConfig)
    }
}

export async function POST(request: NextRequest) {
    try {
        const body = await request.json()
        
        // Validate request body
        const validation = validateRequest(configSchemas.update, body)
        if (!validation.success) {
            return NextResponse.json(
                { error: 'Validation failed', details: validation.errors },
                { status: 400 }
            )
        }

        // Send configuration to backend
        const response = await fetch(`${BACKEND_URL}/api/config`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(validation.data),
        })

        if (!response.ok) {
            throw new Error(`Backend responded with ${response.status}`)
        }

        const savedConfig = await response.json()
        return NextResponse.json(savedConfig)
    } catch (error) {
        logger.error('Error saving configuration', error as Error, {
            component: 'ConfigAPI',
            action: 'POST'
        })
        return NextResponse.json(
            { error: 'Failed to save configuration' },
            { status: 500 }
        )
    }
}

export async function PUT(request: NextRequest) {
    try {
        const body = await request.json()
        
        // Validate request body
        const validation = validateRequest(configSchemas.update, body)
        if (!validation.success) {
            return NextResponse.json(
                { error: 'Validation failed', details: validation.errors },
                { status: 400 }
            )
        }

        // Send configuration update to backend
        const response = await fetch(`${BACKEND_URL}/api/config`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(validation.data),
        })

        if (!response.ok) {
            throw new Error(`Backend responded with ${response.status}`)
        }

        const updatedConfig = await response.json()
        
        logger.info('Configuration updated', {
            component: 'ConfigAPI',
            action: 'PUT',
            updatedFields: Object.keys(body || {}).join(', ')
        })
        
        return NextResponse.json(updatedConfig)
    } catch (error) {
        logger.error('Error updating configuration', error as Error, {
            component: 'ConfigAPI',
            action: 'PUT'
        })
        return NextResponse.json(
            { error: 'Failed to update configuration' },
            { status: 500 }
        )
    }
}