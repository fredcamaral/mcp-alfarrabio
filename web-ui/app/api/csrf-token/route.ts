/**
 * CSRF Token API Endpoint
 * 
 * Provides CSRF tokens for client-side form protection.
 * Follows double-submit cookie pattern for security.
 */

import { NextRequest, NextResponse } from 'next/server'
import { setCSRFToken, getCSRFToken } from '@/lib/csrf'

/**
 * GET /api/csrf-token
 * 
 * Returns a CSRF token for the current session.
 * Sets the token in an httpOnly cookie and returns it in the response.
 */
export async function GET(request: NextRequest): Promise<NextResponse> {
  try {
    // Check if token already exists
    let token = await getCSRFToken()
    
    // Generate new token if none exists
    if (!token) {
      token = await setCSRFToken()
    }

    return NextResponse.json(
      { 
        token,
        expires: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString() // 24 hours
      },
      { 
        status: 200,
        headers: {
          'Cache-Control': 'no-cache, no-store, must-revalidate',
          'Pragma': 'no-cache',
          'Expires': '0'
        }
      }
    )
  } catch (error) {
    console.error('Error generating CSRF token:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to generate CSRF token',
        code: 'CSRF_TOKEN_GENERATION_FAILED'
      },
      { status: 500 }
    )
  }
}

/**
 * POST /api/csrf-token
 * 
 * Refreshes the CSRF token for the current session.
 * Useful for long-running sessions or after token expiration.
 */
export async function POST(request: NextRequest): Promise<NextResponse> {
  try {
    // Always generate a new token for POST requests
    const token = await setCSRFToken()

    return NextResponse.json(
      { 
        token,
        message: 'CSRF token refreshed successfully',
        expires: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
      },
      { 
        status: 200,
        headers: {
          'Cache-Control': 'no-cache, no-store, must-revalidate',
          'Pragma': 'no-cache',
          'Expires': '0'
        }
      }
    )
  } catch (error) {
    console.error('Error refreshing CSRF token:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to refresh CSRF token',
        code: 'CSRF_TOKEN_REFRESH_FAILED'
      },
      { status: 500 }
    )
  }
}