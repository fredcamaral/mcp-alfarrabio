// Package security provides security validation rules and enforcement
package security

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Validator provides comprehensive security validation
type Validator struct {
	config ValidationConfig
}

// ValidationConfig configures security validation rules
type ValidationConfig struct {
	// String validation
	MaxStringLength     int      `json:"max_string_length"`
	MinStringLength     int      `json:"min_string_length"`
	AllowedCharsets     []string `json:"allowed_charsets"`
	DisallowedPatterns  []string `json:"disallowed_patterns"`
	RequiredPatterns    []string `json:"required_patterns"`
	
	// Numeric validation
	MaxNumericValue     float64 `json:"max_numeric_value"`
	MinNumericValue     float64 `json:"min_numeric_value"`
	AllowNegativeNumbers bool   `json:"allow_negative_numbers"`
	AllowFloatingPoint  bool    `json:"allow_floating_point"`
	
	// URL validation
	AllowedSchemes      []string `json:"allowed_schemes"`
	AllowedDomains      []string `json:"allowed_domains"`
	DisallowedDomains   []string `json:"disallowed_domains"`
	RequireHTTPS        bool     `json:"require_https"`
	AllowIPAddresses    bool     `json:"allow_ip_addresses"`
	AllowLocalhost      bool     `json:"allow_localhost"`
	
	// Email validation
	AllowedEmailDomains []string `json:"allowed_email_domains"`
	RequireEmailVerification bool `json:"require_email_verification"`
	
	// File validation
	AllowedFileExtensions []string `json:"allowed_file_extensions"`
	DisallowedFileExtensions []string `json:"disallowed_file_extensions"`
	MaxFileSize          int64    `json:"max_file_size"`
	
	// Security patterns
	CheckForSQLInjection    bool `json:"check_for_sql_injection"`
	CheckForXSS             bool `json:"check_for_xss"`
	CheckForPathTraversal   bool `json:"check_for_path_traversal"`
	CheckForCommandInjection bool `json:"check_for_command_injection"`
	
	// Custom validation rules
	CustomRules map[string]ValidationRule `json:"custom_rules"`
}

// ValidationRule represents a custom validation rule
type ValidationRule struct {
	Pattern     string `json:"pattern"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	ErrorMessage string `json:"error_message"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid       bool                `json:"valid"`
	Errors      []ValidationError   `json:"errors"`
	Warnings    []ValidationWarning `json:"warnings"`
	Sanitized   interface{}         `json:"sanitized"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string `json:"field"`
	Value       string `json:"value"`
	Rule        string `json:"rule"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Code        string `json:"code"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// DefaultValidationConfig returns secure validation defaults
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxStringLength:    10000,
		MinStringLength:    0,
		AllowedCharsets:    []string{"utf-8", "ascii"},
		DisallowedPatterns: []string{
			// SQL injection patterns
			`(?i)(union|select|insert|update|delete|drop|create|alter)\s+`,
			`(?i)'.*or.*'`,
			`(?i);.*--`,
			
			// XSS patterns
			`<script[^>]*>`,
			`javascript:`,
			`on\w+\s*=`,
			
			// Path traversal
			`\.\.\/`,
			`\.\.\\`,
			
			// Command injection
			`[;&|]`,
			`\$\(`,
			` \|\| `,
			` && `,
		},
		MaxNumericValue:        1e10,
		MinNumericValue:        -1e10,
		AllowNegativeNumbers:   true,
		AllowFloatingPoint:     true,
		AllowedSchemes:        []string{"https", "http"},
		AllowedDomains:        []string{},
		DisallowedDomains:     []string{"localhost", "127.0.0.1", "0.0.0.0"},
		RequireHTTPS:          false,
		AllowIPAddresses:      false,
		AllowLocalhost:        true,
		AllowedEmailDomains:   []string{},
		RequireEmailVerification: false,
		AllowedFileExtensions: []string{".txt", ".md", ".json", ".yaml", ".yml"},
		DisallowedFileExtensions: []string{".exe", ".bat", ".sh", ".ps1", ".php", ".jsp"},
		MaxFileSize:           10 * 1024 * 1024, // 10MB
		CheckForSQLInjection:  true,
		CheckForXSS:           true,
		CheckForPathTraversal: true,
		CheckForCommandInjection: true,
		CustomRules:           make(map[string]ValidationRule),
	}
}

// NewValidator creates a new security validator
func NewValidator(config ValidationConfig) *Validator {
	return &Validator{
		config: config,
	}
}

// NewDefaultValidator creates a validator with secure defaults
func NewDefaultValidator() *Validator {
	return NewValidator(DefaultValidationConfig())
}

// ValidateString validates a string value against security rules
func (v *Validator) ValidateString(field, value string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Metadata: make(map[string]interface{}),
	}
	
	// Length validation
	if len(value) > v.config.MaxStringLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    value,
			Rule:     "max_length",
			Message:  fmt.Sprintf("String length %d exceeds maximum %d", len(value), v.config.MaxStringLength),
			Severity: "high",
			Code:     "STRING_TOO_LONG",
		})
	}
	
	if len(value) < v.config.MinStringLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    value,
			Rule:     "min_length",
			Message:  fmt.Sprintf("String length %d below minimum %d", len(value), v.config.MinStringLength),
			Severity: "medium",
			Code:     "STRING_TOO_SHORT",
		})
	}
	
	// Character validation
	if !v.isValidCharset(value) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    value,
			Rule:     "charset",
			Message:  "String contains invalid characters",
			Severity: "medium",
			Code:     "INVALID_CHARSET",
		})
	}
	
	// Security pattern validation
	v.checkSecurityPatterns(field, value, &result)
	
	// Custom pattern validation
	v.checkCustomPatterns(field, value, &result)
	
	// Sanitize value
	result.Sanitized = v.sanitizeString(value)
	
	return result
}

// ValidateURL validates a URL against security rules
func (v *Validator) ValidateURL(field, urlStr string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Metadata: make(map[string]interface{}),
	}
	
	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    urlStr,
			Rule:     "url_format",
			Message:  fmt.Sprintf("Invalid URL format: %v", err),
			Severity: "high",
			Code:     "INVALID_URL",
		})
		return result
	}
	
	// Validate scheme
	if !v.isAllowedScheme(parsedURL.Scheme) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    urlStr,
			Rule:     "url_scheme",
			Message:  fmt.Sprintf("URL scheme '%s' not allowed", parsedURL.Scheme),
			Severity: "high",
			Code:     "DISALLOWED_SCHEME",
		})
	}
	
	// HTTPS requirement
	if v.config.RequireHTTPS && parsedURL.Scheme != "https" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    urlStr,
			Rule:     "require_https",
			Message:  "HTTPS is required",
			Severity: "high",
			Code:     "HTTPS_REQUIRED",
		})
	}
	
	// Validate host
	v.validateHost(field, parsedURL.Host, urlStr, &result)
	
	// Check for dangerous URL patterns
	v.checkURLPatterns(field, urlStr, &result)
	
	result.Sanitized = parsedURL.String()
	result.Metadata["parsed_url"] = map[string]string{
		"scheme": parsedURL.Scheme,
		"host":   parsedURL.Host,
		"path":   parsedURL.Path,
	}
	
	return result
}

// ValidateEmail validates an email address
func (v *Validator) ValidateEmail(field, email string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Metadata: make(map[string]interface{}),
	}
	
	// Basic email format validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    email,
			Rule:     "email_format",
			Message:  "Invalid email format",
			Severity: "medium",
			Code:     "INVALID_EMAIL",
		})
		return result
	}
	
	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    email,
			Rule:     "email_structure",
			Message:  "Invalid email structure",
			Severity: "medium",
			Code:     "INVALID_EMAIL_STRUCTURE",
		})
		return result
	}
	
	domain := parts[1]
	
	// Domain whitelist validation
	if len(v.config.AllowedEmailDomains) > 0 {
		allowed := false
		for _, allowedDomain := range v.config.AllowedEmailDomains {
			if strings.EqualFold(domain, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    email,
				Rule:     "email_domain",
				Message:  fmt.Sprintf("Email domain '%s' not allowed", domain),
				Severity: "medium",
				Code:     "DISALLOWED_EMAIL_DOMAIN",
			})
		}
	}
	
	result.Sanitized = strings.ToLower(email)
	result.Metadata["domain"] = domain
	
	return result
}

// ValidateFilename validates a filename for security
func (v *Validator) ValidateFilename(field, filename string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Metadata: make(map[string]interface{}),
	}
	
	// Check for path traversal in filename
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    filename,
			Rule:     "path_traversal",
			Message:  "Filename contains path traversal characters",
			Severity: "critical",
			Code:     "PATH_TRAVERSAL",
		})
	}
	
	// Extract file extension
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	
	// Check allowed extensions
	if len(v.config.AllowedFileExtensions) > 0 {
		allowed := false
		for _, allowedExt := range v.config.AllowedFileExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    filename,
				Rule:     "file_extension",
				Message:  fmt.Sprintf("File extension '%s' not allowed", ext),
				Severity: "high",
				Code:     "DISALLOWED_EXTENSION",
			})
		}
	}
	
	// Check disallowed extensions
	for _, disallowedExt := range v.config.DisallowedFileExtensions {
		if ext == disallowedExt {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    filename,
				Rule:     "file_extension",
				Message:  fmt.Sprintf("File extension '%s' is forbidden", ext),
				Severity: "critical",
				Code:     "FORBIDDEN_EXTENSION",
			})
			break
		}
	}
	
	// Check for dangerous filenames
	dangerousNames := []string{"con", "prn", "aux", "nul", "com1", "com2", "lpt1", "lpt2"}
	baseFilename := strings.ToLower(filename[:strings.LastIndex(filename, ".")])
	for _, dangerous := range dangerousNames {
		if baseFilename == dangerous {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    filename,
				Rule:     "dangerous_filename",
				Message:  fmt.Sprintf("Filename '%s' is a reserved system name", baseFilename),
				Severity: "high",
				Code:     "RESERVED_FILENAME",
			})
			break
		}
	}
	
	result.Sanitized = filename
	result.Metadata["extension"] = ext
	result.Metadata["base_name"] = baseFilename
	
	return result
}

// isValidCharset checks if string uses valid charset
func (v *Validator) isValidCharset(value string) bool {
	if len(v.config.AllowedCharsets) == 0 {
		return true // No restrictions
	}
	
	for _, charset := range v.config.AllowedCharsets {
		switch charset {
		case "ascii":
			if v.isASCII(value) {
				return true
			}
		case "utf-8":
			if v.isValidUTF8(value) {
				return true
			}
		}
	}
	
	return false
}

// isASCII checks if string contains only ASCII characters
func (v *Validator) isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// isValidUTF8 checks if string is valid UTF-8
func (v *Validator) isValidUTF8(s string) bool {
	for _, r := range s {
		if r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

// isAllowedScheme checks if URL scheme is allowed
func (v *Validator) isAllowedScheme(scheme string) bool {
	if len(v.config.AllowedSchemes) == 0 {
		return true
	}
	
	for _, allowed := range v.config.AllowedSchemes {
		if strings.EqualFold(scheme, allowed) {
			return true
		}
	}
	
	return false
}

// validateHost validates URL host
func (v *Validator) validateHost(field, host, fullURL string, result *ValidationResult) {
	// Check for IP addresses
	if ip := net.ParseIP(host); ip != nil {
		if !v.config.AllowIPAddresses {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    fullURL,
				Rule:     "ip_addresses",
				Message:  "IP addresses not allowed in URLs",
				Severity: "medium",
				Code:     "IP_ADDRESS_FORBIDDEN",
			})
		}
		
		// Check for localhost IP
		if !v.config.AllowLocalhost && (ip.IsLoopback() || ip.IsPrivate()) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    fullURL,
				Rule:     "localhost",
				Message:  "Localhost/private IP addresses not allowed",
				Severity: "medium",
				Code:     "LOCALHOST_FORBIDDEN",
			})
		}
		return
	}
	
	// Check localhost domain
	if !v.config.AllowLocalhost && (strings.EqualFold(host, "localhost") || strings.HasSuffix(host, ".local")) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    field,
			Value:    fullURL,
			Rule:     "localhost",
			Message:  "Localhost domains not allowed",
			Severity: "medium",
			Code:     "LOCALHOST_DOMAIN_FORBIDDEN",
		})
	}
	
	// Check domain whitelist
	if len(v.config.AllowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range v.config.AllowedDomains {
			if strings.EqualFold(host, allowedDomain) || strings.HasSuffix(host, "."+allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    fullURL,
				Rule:     "domain_whitelist",
				Message:  fmt.Sprintf("Domain '%s' not in allowed list", host),
				Severity: "high",
				Code:     "DOMAIN_NOT_ALLOWED",
			})
		}
	}
	
	// Check domain blacklist
	for _, disallowed := range v.config.DisallowedDomains {
		if strings.EqualFold(host, disallowed) || strings.HasSuffix(host, "."+disallowed) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    fullURL,
				Rule:     "domain_blacklist",
				Message:  fmt.Sprintf("Domain '%s' is blacklisted", host),
				Severity: "critical",
				Code:     "DOMAIN_BLACKLISTED",
			})
			break
		}
	}
}

// checkSecurityPatterns checks for common security threats
func (v *Validator) checkSecurityPatterns(field, value string, result *ValidationResult) {
	// SQL injection check
	if v.config.CheckForSQLInjection {
		sqlPatterns := []string{
			`(?i)(union|select|insert|update|delete|drop|create|alter|exec)\s+`,
			`(?i)'.*or.*'`,
			`(?i)'.*and.*'`,
			`(?i);.*--`,
			`(?i)/\*.*\*/`,
		}
		
		for _, pattern := range sqlPatterns {
			if matched, _ := regexp.MatchString(pattern, value); matched {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    field,
					Value:    value,
					Rule:     "sql_injection",
					Message:  "Potential SQL injection detected",
					Severity: "critical",
					Code:     "SQL_INJECTION",
				})
				break
			}
		}
	}
	
	// XSS check
	if v.config.CheckForXSS {
		xssPatterns := []string{
			`<script[^>]*>`,
			`javascript:`,
			`on\w+\s*=`,
			`<iframe[^>]*>`,
			`<object[^>]*>`,
			`<embed[^>]*>`,
		}
		
		for _, pattern := range xssPatterns {
			if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    field,
					Value:    value,
					Rule:     "xss",
					Message:  "Potential XSS attack detected",
					Severity: "critical",
					Code:     "XSS_ATTACK",
				})
				break
			}
		}
	}
	
	// Path traversal check
	if v.config.CheckForPathTraversal {
		pathPatterns := []string{`\.\.\/`, `\.\.\\`, `%2e%2e%2f`, `%2e%2e%5c`}
		
		for _, pattern := range pathPatterns {
			if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    field,
					Value:    value,
					Rule:     "path_traversal",
					Message:  "Potential path traversal attack detected",
					Severity: "critical",
					Code:     "PATH_TRAVERSAL",
				})
				break
			}
		}
	}
	
	// Command injection check
	if v.config.CheckForCommandInjection {
		cmdPatterns := []string{`[;&|]`, `\$\(`, ` \|\| `, ` && `, "`"}
		
		for _, pattern := range cmdPatterns {
			if matched, _ := regexp.MatchString(pattern, value); matched {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:    field,
					Value:    value,
					Rule:     "command_injection",
					Message:  "Potential command injection detected",
					Severity: "critical",
					Code:     "COMMAND_INJECTION",
				})
				break
			}
		}
	}
}

// checkCustomPatterns checks custom validation patterns
func (v *Validator) checkCustomPatterns(field, value string, result *ValidationResult) {
	// Check disallowed patterns
	for _, pattern := range v.config.DisallowedPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    value,
				Rule:     "disallowed_pattern",
				Message:  "Value matches disallowed pattern",
				Severity: "medium",
				Code:     "DISALLOWED_PATTERN",
			})
		}
	}
	
	// Check required patterns
	for _, pattern := range v.config.RequiredPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, value); !matched {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    value,
				Rule:     "required_pattern",
				Message:  "Value doesn't match required pattern",
				Severity: "medium",
				Code:     "MISSING_REQUIRED_PATTERN",
			})
		}
	}
	
	// Check custom rules
	for ruleName, rule := range v.config.CustomRules {
		if matched, _ := regexp.MatchString(`(?i)`+rule.Pattern, value); rule.Required && !matched {
			result.Valid = false
			message := rule.ErrorMessage
			if message == "" {
				message = fmt.Sprintf("Value doesn't match rule '%s'", ruleName)
			}
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    value,
				Rule:     ruleName,
				Message:  message,
				Severity: "medium",
				Code:     "CUSTOM_RULE_VIOLATION",
			})
		} else if !rule.Required && matched {
			// Optional pattern that should trigger a warning if matched
			message := rule.ErrorMessage
			if message == "" {
				message = fmt.Sprintf("Value matches warning pattern '%s'", ruleName)
			}
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   field,
				Value:   value,
				Rule:    ruleName,
				Message: message,
			})
		}
	}
}

// checkURLPatterns checks for dangerous URL patterns
func (v *Validator) checkURLPatterns(field, urlStr string, result *ValidationResult) {
	dangerousPatterns := []string{
		`file://`,
		`ftp://`,
		`ldap://`,
		`jar:`,
		`data:`,
		`vbscript:`,
		`javascript:`,
	}
	
	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, urlStr); matched {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    field,
				Value:    urlStr,
				Rule:     "dangerous_url",
				Message:  "URL contains dangerous protocol or pattern",
				Severity: "critical",
				Code:     "DANGEROUS_URL_PATTERN",
			})
			break
		}
	}
}

// sanitizeString performs basic string sanitization
func (v *Validator) sanitizeString(value string) string {
	// Remove null bytes
	value = strings.ReplaceAll(value, "\x00", "")
	
	// Normalize whitespace
	value = strings.TrimSpace(value)
	
	// Remove control characters except tab, newline, carriage return
	var result strings.Builder
	for _, r := range value {
		if r == '\t' || r == '\n' || r == '\r' || !unicode.IsControl(r) {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}