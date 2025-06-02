# CSRF Protection Documentation

This document describes the CSRF (Cross-Site Request Forgery) protection implementation for the MCP Memory WebUI.

## Overview

The CSRF protection system implements the **double-submit cookie pattern** with secure token validation to prevent CSRF attacks against the WebUI forms and API endpoints.

## Architecture

### Components

1. **CSRF Token Management** (`lib/csrf.ts`)
   - Cryptographically secure token generation
   - Cookie-based token storage with security flags
   - Token validation utilities

2. **Middleware Protection** (`middleware.ts`)
   - Automatic CSRF validation for protected routes
   - Content-type aware validation
   - Route-based exemption system

3. **React Integration** (`hooks/useCSRFProtection.ts`)
   - React hooks for token management
   - Automatic token refresh and validation
   - Protected request utilities

4. **Form Components** (`components/shared/ProtectedForm.tsx`)
   - Ready-to-use protected form wrapper
   - Built-in error handling and loading states
   - Automatic token injection

5. **Provider System** (`providers/CSRFProvider.tsx`)
   - Application-wide CSRF context
   - Automatic initialization and management
   - Status monitoring components

## Security Features

### Token Security
- **Cryptographically secure**: Uses `crypto.getRandomValues()` or Node.js `crypto.randomBytes()`
- **64-character hex tokens**: 32 bytes of entropy (256 bits)
- **HttpOnly cookies**: Prevents XSS-based token theft
- **SameSite=Strict**: Prevents cross-site cookie transmission
- **Secure flag**: HTTPS-only transmission in production

### Validation Methods
- **Header validation**: `X-CSRF-Token` header for AJAX requests
- **Form validation**: Hidden form field for traditional submissions
- **Double-submit verification**: Cookie value must match submitted token

### Route Protection
- **API endpoints**: All `/api/*` routes except health/status
- **Content-type aware**: Validates based on request content type
- **Method filtering**: Only validates unsafe methods (POST, PUT, PATCH, DELETE)

## Usage

### Basic Form Protection

```tsx
import { ProtectedForm } from '@/components/shared/ProtectedForm'

function MyForm() {
  const handleSubmit = async (formData, submitProtected) => {
    const response = await submitProtected('/api/endpoint', {
      field1: formData.get('field1'),
      field2: formData.get('field2')
    })
    
    if (!response.ok) {
      throw new Error('Submission failed')
    }
  }

  return (
    <ProtectedForm onSubmit={handleSubmit}>
      <input name="field1" type="text" />
      <input name="field2" type="text" />
      <button type="submit">Submit</button>
    </ProtectedForm>
  )
}
```

### Manual CSRF Protection

```tsx
import { useCSRFForm } from '@/hooks/useCSRFProtection'

function CustomForm() {
  const { submitForm, getCSRFInput, isTokenValid } = useCSRFForm()

  const handleSubmit = async (e) => {
    e.preventDefault()
    const formData = new FormData(e.target)
    
    const response = await submitForm('/api/endpoint', formData)
    // Handle response...
  }

  return (
    <form onSubmit={handleSubmit}>
      {getCSRFInput()}
      <input name="data" type="text" />
      <button type="submit" disabled={!isTokenValid}>
        Submit
      </button>
    </form>
  )
}
```

### Protected API Requests

```tsx
import { useCSRF } from '@/providers/CSRFProvider'

function ApiComponent() {
  const { makeProtectedRequest } = useCSRF()

  const createMemory = async (data) => {
    const response = await makeProtectedRequest('/api/memories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
    
    return response.json()
  }
}
```

## Configuration

### Environment Variables
- `NODE_ENV`: Affects cookie security flags
- `NEXT_PUBLIC_API_URL`: Base URL for API requests

### Route Configuration
Edit `middleware.ts` to customize protected routes:

```typescript
// Add routes that require CSRF protection
const PROTECTED_ROUTES = [
  '/api/memories',
  '/api/search',
  '/api/custom-endpoint'
]

// Add routes exempt from CSRF protection
const EXEMPT_ROUTES = [
  '/api/csrf-token',
  '/api/health',
  '/api/public-data'
]
```

### Token Refresh
Configure automatic token refresh in the provider:

```tsx
<CSRFProvider autoRefreshInterval={30 * 60 * 1000}> {/* 30 minutes */}
  <App />
</CSRFProvider>
```

## Security Considerations

### Token Lifecycle
- **24-hour expiration**: Tokens automatically expire
- **Automatic refresh**: Tokens refresh on page visibility change
- **Session invalidation**: Tokens clear on logout

### Error Handling
- **Graceful degradation**: Forms disable when tokens unavailable
- **Retry logic**: Automatic token refresh on validation failures
- **User feedback**: Clear error messages for CSRF failures

### Best Practices
1. **Always use ProtectedForm**: For consistent CSRF protection
2. **Validate server-side**: Never rely solely on client-side validation
3. **Monitor token status**: Use CSRFStatusIndicator for debugging
4. **Handle errors gracefully**: Provide clear user feedback

## API Reference

### CSRF Token Endpoint

#### GET /api/csrf-token
Returns a CSRF token for the current session.

**Response:**
```json
{
  "token": "abc123...",
  "expires": "2024-01-01T12:00:00.000Z"
}
```

#### POST /api/csrf-token
Refreshes the CSRF token for the current session.

**Response:**
```json
{
  "token": "def456...",
  "message": "CSRF token refreshed successfully",
  "expires": "2024-01-01T12:00:00.000Z"
}
```

### Error Responses

#### 403 Forbidden - Missing Token
```json
{
  "error": "CSRF token required",
  "code": "CSRF_TOKEN_MISSING"
}
```

#### 403 Forbidden - Invalid Token
```json
{
  "error": "CSRF token validation failed",
  "code": "CSRF_TOKEN_INVALID",
  "details": {
    "method": "POST",
    "pathname": "/api/endpoint",
    "contentType": "application/json",
    "hasToken": true,
    "hasHeader": false
  }
}
```

## Troubleshooting

### Common Issues

1. **"CSRF token required" error**
   - Ensure CSRFProvider wraps your app
   - Check that tokens are being fetched on initialization

2. **"CSRF token validation failed" error**
   - Verify token is included in requests (header or form field)
   - Check token hasn't expired (24-hour limit)
   - Ensure cookies are enabled

3. **Forms not submitting**
   - Check isTokenValid before enabling submit buttons
   - Verify ProtectedForm is being used correctly
   - Look for network errors preventing token fetch

### Debug Mode
In development, use the CSRFTokenDisplay component to monitor token status:

```tsx
import { CSRFTokenDisplay } from '@/components/shared/ProtectedForm'

function DebugForm() {
  return (
    <div>
      <MyProtectedForm />
      <CSRFTokenDisplay /> {/* Shows token status */}
    </div>
  )
}
```

### Network Debugging
Enable detailed CSRF logging in the browser console:

```javascript
// Enable CSRF debug logging
localStorage.setItem('csrf-debug', 'true')
```

## Testing

### Unit Tests
Test CSRF protection components:

```javascript
import { render, fireEvent } from '@testing-library/react'
import { CSRFProvider } from '@/providers/CSRFProvider'
import { ProtectedForm } from '@/components/shared/ProtectedForm'

test('form includes CSRF token', async () => {
  const { getByRole } = render(
    <CSRFProvider>
      <ProtectedForm onSubmit={jest.fn()}>
        <button type="submit">Submit</button>
      </ProtectedForm>
    </CSRFProvider>
  )
  
  // Test implementation...
})
```

### Integration Tests
Test end-to-end CSRF protection:

```javascript
// Test form submission with CSRF protection
await page.goto('/memory-form')
await page.fill('[name="content"]', 'Test memory')
await page.click('[type="submit"]')
await expect(page.locator('.success-message')).toBeVisible()
```

## Performance Considerations

### Token Caching
- Tokens cached in memory to avoid repeated API calls
- Automatic refresh prevents unnecessary requests
- Efficient cookie-based storage

### Request Overhead
- Minimal overhead: single header per request
- No additional round trips for token validation
- Efficient cryptographic operations

### Bundle Size
- Tree-shakable utilities minimize bundle impact
- Optional components reduce unused code
- TypeScript provides zero-runtime overhead

## Future Enhancements

1. **CSP Integration**: Content Security Policy headers
2. **Rate Limiting**: Request throttling for token endpoints
3. **Audit Logging**: CSRF attack attempt logging
4. **Multi-device Support**: Device-specific token management
5. **GraphQL Integration**: CSRF protection for GraphQL mutations