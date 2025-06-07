/**
 * CSRF Token API Endpoint
 * 
 * Provides CSRF tokens for client-side form protection.
 * Follows double-submit cookie pattern for security.
 */

import { NextRequest, NextResponse } from 'next/server'
import { generateCSRFToken, validateCSRFToken } from '@/lib/csrf'
import { logger } from '@/lib/logger'

/**
 * GET /api/csrf-token
 * 
 * Generates and returns a new CSRF token for client-side requests
 */
export async function GET() {
  try {
    const token = generateCSRFToken()

    const response = NextResponse.json({
      token,
      expires: Date.now() + (24 * 60 * 60 * 1000) // 24 hours
    })

    // Set CSRF token in httpOnly cookie for additional security
    response.cookies.set('csrf-token', token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      maxAge: 24 * 60 * 60 // 24 hours
    })

    logger.info('CSRF token generated', {
      component: 'CSRF',
      action: 'generate'
    })

    return response
  } catch (error) {
    logger.error('Error generating CSRF token', error as Error, {
      component: 'CSRF',
      action: 'generate'
    })
    return NextResponse.json(
      { error: 'Failed to generate CSRF token' },
      { status: 500 }
    )
  }
}

/**
 * POST /api/csrf-token/validate
 * 
 * Validates a CSRF token
 */
export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const { token } = body || {}

    if (!token) {
      return NextResponse.json(
        { error: 'Token is required' },
        { status: 400 }
      )
    }

    // Validate the token
    const isValid = validateCSRFToken(token)

    if (!isValid) {
      logger.warn('Invalid CSRF token attempted', {
        component: 'CSRF',
        action: 'validate'
      })
      return NextResponse.json({
        valid: false,
        message: 'CSRF token is invalid or expired'
      })
    }

    return NextResponse.json({
      valid: true,
      message: 'CSRF token is valid'
    })
  } catch (error) {
    logger.error('Error validating CSRF token', error as Error, {
      component: 'CSRF',
      action: 'validate'
    })
    return NextResponse.json(
      { error: 'Failed to validate CSRF token' },
      { status: 500 }
    )
  }
}