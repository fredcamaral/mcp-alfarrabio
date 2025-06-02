/**
 * Content Sanitization Utilities
 * 
 * Provides comprehensive content sanitization to prevent XSS attacks and ensure
 * safe handling of user-generated content across the WebUI.
 */

// HTML entities for escaping
const HTML_ENTITIES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#x27;',
  '/': '&#x2F;',
  '`': '&#x60;',
  '=': '&#x3D;'
}

// URL protocol whitelist
const SAFE_URL_PROTOCOLS = ['http:', 'https:', 'mailto:', 'tel:', 'ftp:']

// Dangerous HTML tags to completely remove
const DANGEROUS_TAGS = [
  'script', 'iframe', 'object', 'embed', 'form', 'input', 'textarea', 
  'select', 'button', 'link', 'meta', 'style', 'title', 'base'
]

// Dangerous HTML attributes to remove
const DANGEROUS_ATTRIBUTES = [
  'onload', 'onerror', 'onclick', 'onmouseover', 'onmouseout', 'onkeydown',
  'onkeyup', 'onkeypress', 'onfocus', 'onblur', 'onchange', 'onsubmit',
  'onreset', 'onresize', 'onscroll', 'onunload', 'onbeforeunload',
  'javascript:', 'vbscript:', 'data:', 'livescript:', 'mocha:',
  'expression', 'eval', 'script'
]

export interface SanitizationOptions {
  allowHtml?: boolean
  allowLinks?: boolean
  allowImages?: boolean
  maxLength?: number
  preserveWhitespace?: boolean
  allowedTags?: string[]
  allowedAttributes?: string[]
  removeEmptyLines?: boolean
  convertNewlines?: boolean
}

export interface SanitizationResult {
  sanitized: string
  violations: string[]
  truncated: boolean
  originalLength: number
  sanitizedLength: number
}

/**
 * Sanitize content with comprehensive security measures
 */
export function sanitizeContent(
  content: string,
  options: SanitizationOptions = {}
): SanitizationResult {
  if (!content || typeof content !== 'string') {
    return {
      sanitized: '',
      violations: ['Invalid content type'],
      truncated: false,
      originalLength: 0,
      sanitizedLength: 0
    }
  }

  const {
    allowHtml = false,
    allowLinks = true,
    allowImages = false,
    maxLength = 10000,
    preserveWhitespace = false,
    allowedTags = ['p', 'br', 'strong', 'em', 'code', 'pre'],
    allowedAttributes = ['href', 'title', 'alt'],
    removeEmptyLines = true,
    convertNewlines = true
  } = options

  const violations: string[] = []
  let sanitized = content
  const originalLength = content.length

  // 1. Basic length validation and truncation
  let truncated = false
  if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength)
    truncated = true
    violations.push(`Content truncated from ${originalLength} to ${maxLength} characters`)
  }

  // 2. Remove or escape dangerous patterns
  sanitized = removeDangerousPatterns(sanitized, violations)

  // 3. Handle HTML content
  if (allowHtml) {
    sanitized = sanitizeHTML(sanitized, allowedTags, allowedAttributes, violations)
  } else {
    sanitized = escapeHTML(sanitized)
    if (convertNewlines) {
      sanitized = sanitized.replace(/\n/g, '<br>')
    }
  }

  // 4. Handle URLs and links
  if (allowLinks) {
    sanitized = sanitizeURLs(sanitized, violations)
  } else {
    sanitized = removeURLs(sanitized, violations)
  }

  // 5. Handle images
  if (!allowImages) {
    sanitized = removeImages(sanitized, violations)
  }

  // 6. Clean up whitespace
  if (!preserveWhitespace) {
    sanitized = cleanWhitespace(sanitized, removeEmptyLines)
  }

  // 7. Final security check
  sanitized = finalSecurityScan(sanitized, violations)

  return {
    sanitized: sanitized.trim(),
    violations,
    truncated,
    originalLength,
    sanitizedLength: sanitized.length
  }
}

/**
 * Escape HTML characters to prevent XSS
 */
export function escapeHTML(text: string): string {
  return text.replace(/[&<>"'`=\/]/g, (match) => HTML_ENTITIES[match] || match)
}

/**
 * Unescape HTML entities
 */
export function unescapeHTML(text: string): string {
  const entityMap = Object.fromEntries(
    Object.entries(HTML_ENTITIES).map(([char, entity]) => [entity, char])
  )
  
  return text.replace(/&(?:amp|lt|gt|quot|#x27|#x2F|#x60|#x3D);/g, (match) => 
    entityMap[match] || match
  )
}

/**
 * Remove dangerous patterns like script injections
 */
function removeDangerousPatterns(content: string, violations: string[]): string {
  let cleaned = content

  // Remove script tags and content
  const scriptRegex = /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi
  if (scriptRegex.test(cleaned)) {
    violations.push('Script tags removed')
    cleaned = cleaned.replace(scriptRegex, '')
  }

  // Remove javascript: URLs
  const jsUrlRegex = /javascript\s*:/gi
  if (jsUrlRegex.test(cleaned)) {
    violations.push('JavaScript URLs removed')
    cleaned = cleaned.replace(jsUrlRegex, 'removed:')
  }

  // Remove data: URLs with suspicious content
  const dataUrlRegex = /data:(?!image\/)[^;,]*[;,]/gi
  if (dataUrlRegex.test(cleaned)) {
    violations.push('Suspicious data URLs removed')
    cleaned = cleaned.replace(dataUrlRegex, 'removed:')
  }

  // Remove vbscript: URLs
  const vbsRegex = /vbscript\s*:/gi
  if (vbsRegex.test(cleaned)) {
    violations.push('VBScript URLs removed')
    cleaned = cleaned.replace(vbsRegex, 'removed:')
  }

  // Remove expression() calls
  const exprRegex = /expression\s*\(/gi
  if (exprRegex.test(cleaned)) {
    violations.push('CSS expression() calls removed')
    cleaned = cleaned.replace(exprRegex, 'removed(')
  }

  return cleaned
}

/**
 * Sanitize HTML tags and attributes
 */
function sanitizeHTML(
  content: string, 
  allowedTags: string[], 
  allowedAttributes: string[],
  violations: string[]
): string {
  let sanitized = content

  // Remove dangerous tags entirely
  DANGEROUS_TAGS.forEach(tag => {
    const regex = new RegExp(`<${tag}\\b[^>]*>.*?<\/${tag}>`, 'gi')
    if (regex.test(sanitized)) {
      violations.push(`Dangerous tag <${tag}> removed`)
      sanitized = sanitized.replace(regex, '')
    }
  })

  // Remove dangerous attributes
  DANGEROUS_ATTRIBUTES.forEach(attr => {
    const regex = new RegExp(`\\s${attr}\\s*=\\s*['""][^'"]*['"]`, 'gi')
    if (regex.test(sanitized)) {
      violations.push(`Dangerous attribute ${attr} removed`)
      sanitized = sanitized.replace(regex, '')
    }
  })

  // Filter allowed tags
  const tagRegex = /<\/?([a-z][a-z0-9]*)\b[^>]*>/gi
  sanitized = sanitized.replace(tagRegex, (match, tagName) => {
    if (!allowedTags.includes(tagName.toLowerCase())) {
      violations.push(`Disallowed tag <${tagName}> removed`)
      return ''
    }
    return match
  })

  return sanitized
}

/**
 * Sanitize URLs to ensure they're safe
 */
function sanitizeURLs(content: string, violations: string[]): string {
  // URL regex pattern
  const urlRegex = /(https?:\/\/[^\s<>"']+)/gi
  
  return content.replace(urlRegex, (match) => {
    try {
      const url = new URL(match)
      
      // Check protocol
      if (!SAFE_URL_PROTOCOLS.includes(url.protocol)) {
        violations.push(`Unsafe URL protocol ${url.protocol} removed`)
        return '[unsafe URL removed]'
      }
      
      // Check for suspicious patterns
      if (url.pathname.includes('..') || url.search.includes('<script')) {
        violations.push('Suspicious URL pattern removed')
        return '[suspicious URL removed]'
      }
      
      return match
    } catch (error) {
      violations.push('Invalid URL removed')
      return '[invalid URL removed]'
    }
  })
}

/**
 * Remove all URLs from content
 */
function removeURLs(content: string, violations: string[]): string {
  const urlRegex = /(https?:\/\/[^\s<>"']+)/gi
  
  if (urlRegex.test(content)) {
    violations.push('URLs removed')
    return content.replace(urlRegex, '[URL removed]')
  }
  
  return content
}

/**
 * Remove image tags and references
 */
function removeImages(content: string, violations: string[]): string {
  let cleaned = content

  // Remove img tags
  const imgRegex = /<img\b[^>]*>/gi
  if (imgRegex.test(cleaned)) {
    violations.push('Image tags removed')
    cleaned = cleaned.replace(imgRegex, '[image removed]')
  }

  // Remove image URLs in markdown format
  const mdImgRegex = /!\[[^\]]*\]\([^)]+\)/gi
  if (mdImgRegex.test(cleaned)) {
    violations.push('Markdown images removed')
    cleaned = cleaned.replace(mdImgRegex, '[image removed]')
  }

  return cleaned
}

/**
 * Clean up whitespace and formatting
 */
function cleanWhitespace(content: string, removeEmptyLines: boolean): string {
  let cleaned = content

  // Normalize line endings
  cleaned = cleaned.replace(/\r\n/g, '\n').replace(/\r/g, '\n')

  // Remove excessive whitespace
  cleaned = cleaned.replace(/[ \t]+/g, ' ')

  // Remove empty lines if requested
  if (removeEmptyLines) {
    cleaned = cleaned.replace(/\n\s*\n/g, '\n')
  }

  // Remove leading/trailing whitespace from lines
  cleaned = cleaned.replace(/^[ \t]+|[ \t]+$/gm, '')

  return cleaned
}

/**
 * Final security scan for remaining threats
 */
function finalSecurityScan(content: string, violations: string[]): string {
  let secured = content

  // Check for encoded script attempts
  const encodedScriptRegex = /&#x6A;&#x61;&#x76;&#x61;&#x73;&#x63;&#x72;&#x69;&#x70;&#x74;/gi
  if (encodedScriptRegex.test(secured)) {
    violations.push('Encoded script attempts removed')
    secured = secured.replace(encodedScriptRegex, '[encoded script removed]')
  }

  // Check for unicode script attempts
  const unicodeScriptRegex = /\u006A\u0061\u0076\u0061\u0073\u0063\u0072\u0069\u0070\u0074/gi
  if (unicodeScriptRegex.test(secured)) {
    violations.push('Unicode script attempts removed')
    secured = secured.replace(unicodeScriptRegex, '[unicode script removed]')
  }

  return secured
}

/**
 * Sanitize filename for safe file operations
 */
export function sanitizeFilename(filename: string): string {
  return filename
    .replace(/[<>:"/\\|?*]/g, '_') // Replace invalid chars
    .replace(/^\.+/, '') // Remove leading dots
    .replace(/\.+$/, '') // Remove trailing dots
    .replace(/\s+/g, '_') // Replace spaces with underscores
    .substring(0, 255) // Limit length
}

/**
 * Sanitize memory chunk content
 */
export function sanitizeMemoryContent(content: string): SanitizationResult {
  return sanitizeContent(content, {
    allowHtml: false,
    allowLinks: true,
    allowImages: false,
    maxLength: 50000, // Larger limit for memory content
    preserveWhitespace: true,
    removeEmptyLines: false,
    convertNewlines: false
  })
}

/**
 * Sanitize search query
 */
export function sanitizeSearchQuery(query: string): string {
  const result = sanitizeContent(query, {
    allowHtml: false,
    allowLinks: false,
    allowImages: false,
    maxLength: 1000,
    preserveWhitespace: false,
    removeEmptyLines: true,
    convertNewlines: false
  })
  
  return result.sanitized
}

/**
 * Sanitize user input for forms
 */
export function sanitizeFormInput(input: string, maxLength: number = 1000): string {
  const result = sanitizeContent(input, {
    allowHtml: false,
    allowLinks: false,
    allowImages: false,
    maxLength,
    preserveWhitespace: false,
    removeEmptyLines: true,
    convertNewlines: false
  })
  
  return result.sanitized
}