package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionManager handles data encryption and decryption
type EncryptionManager struct {
	enabled    bool
	masterKey  []byte
	saltLength int
	keyLength  int
	iterations int
}

// EncryptedData represents encrypted data with metadata
type EncryptedData struct {
	Algorithm string `json:"algorithm"`
	Salt      string `json:"salt"`
	IV        string `json:"iv"`
	Data      string `json:"data"`
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager(masterPassword string) *EncryptionManager {
	em := &EncryptionManager{
		enabled:    masterPassword != "",
		saltLength: 32,
		keyLength:  32,
		iterations: 100000,
	}

	if em.enabled {
		// Derive master key from password
		salt := make([]byte, em.saltLength)
		if _, err := rand.Read(salt); err != nil {
			panic("failed to read random bytes: " + err.Error())
		}
		em.masterKey = pbkdf2.Key([]byte(masterPassword), salt, em.iterations, em.keyLength, sha256.New)
	}

	return em
}

// EncryptString encrypts a string using AES-GCM
func (em *EncryptionManager) EncryptString(plaintext string) (*EncryptedData, error) {
	if !em.enabled {
		return &EncryptedData{
			Algorithm: "none",
			Data:      plaintext,
		}, nil
	}

	if plaintext == "" {
		return &EncryptedData{
			Algorithm: "aes-gcm",
			Data:      "",
		}, nil
	}

	// Generate salt for this encryption
	salt := make([]byte, em.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key for this encryption
	key := pbkdf2.Key(em.masterKey, salt, em.iterations, em.keyLength, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate IV
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	return &EncryptedData{
		Algorithm: "aes-gcm",
		Salt:      base64.StdEncoding.EncodeToString(salt),
		IV:        base64.StdEncoding.EncodeToString(iv),
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// DecryptString decrypts a string using AES-GCM
func (em *EncryptionManager) DecryptString(encrypted *EncryptedData) (string, error) {
	if encrypted.Algorithm == "none" {
		return encrypted.Data, nil
	}

	if !em.enabled {
		return "", errors.New("encryption is not enabled")
	}

	if encrypted.Data == "" {
		return "", nil
	}

	if encrypted.Algorithm != "aes-gcm" {
		return "", errors.New("unsupported encryption algorithm: " + encrypted.Algorithm)
	}

	// Decode components
	salt, err := base64.StdEncoding.DecodeString(encrypted.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(encrypted.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Derive key
	key := pbkdf2.Key(em.masterKey, salt, em.iterations, em.keyLength, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	return string(plaintext), nil
}

// EncryptSensitiveFields encrypts sensitive fields in chunk content
func (em *EncryptionManager) EncryptSensitiveFields(content string) (string, error) {
	if !em.enabled {
		return content, nil
	}

	// Define patterns for sensitive data
	sensitivePatterns := map[string]*regexp.Regexp{
		"api_key":    regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?([a-zA-Z0-9\-_]+)['"]?`),
		"password":   regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['"]?([^\s'"]+)['"]?`),
		"token":      regexp.MustCompile(`(?i)(token|access[_-]?token)\s*[:=]\s*['"]?([a-zA-Z0-9\-._]+)['"]?`),
		"secret":     regexp.MustCompile(`(?i)(secret|secret[_-]?key)\s*[:=]\s*['"]?([a-zA-Z0-9\-_]+)['"]?`),
		"connection": regexp.MustCompile(`(?i)(connection[_-]?string|conn[_-]?str)\s*[:=]\s*['"]?([^'"]+)['"]?`),
	}

	result := content

	for fieldType, pattern := range sensitivePatterns {
		matches := pattern.FindAllStringSubmatch(result, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			original := match[0]
			sensitiveValue := match[2]

			// Encrypt the sensitive value
			encrypted, err := em.EncryptString(sensitiveValue)
			if err != nil {
				return "", fmt.Errorf("failed to encrypt %s: %w", fieldType, err)
			}

			// Replace with encrypted placeholder
			encryptedPlaceholder := "[ENCRYPTED:" + fieldType + ":" + encrypted.Data + "]"
			result = strings.ReplaceAll(result, original,
				strings.Replace(original, sensitiveValue, encryptedPlaceholder, 1))
		}
	}

	return result, nil
}

// DecryptSensitiveFields decrypts sensitive fields in chunk content
func (em *EncryptionManager) DecryptSensitiveFields(content string) (string, error) {
	if !em.enabled {
		return content, nil
	}

	// Find encrypted placeholders
	encryptedPattern := regexp.MustCompile(`\[ENCRYPTED:([^:]+):([^\]]+)\]`)
	matches := encryptedPattern.FindAllStringSubmatch(content, -1)

	result := content

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		placeholder := match[0]
		_ = match[1] // fieldType
		encryptedData := match[2]

		// Decrypt the value
		encrypted := &EncryptedData{
			Algorithm: "aes-gcm",
			Data:      encryptedData,
		}

		decrypted, err := em.DecryptString(encrypted)
		if err != nil {
			// If decryption fails, leave placeholder as is
			continue
		}

		// Replace placeholder with decrypted value
		result = strings.ReplaceAll(result, placeholder, decrypted)
	}

	return result, nil
}

// AnonymizeData anonymizes personally identifiable information
func (em *EncryptionManager) AnonymizeData(content string) string {
	// Define anonymization patterns - order matters! More specific patterns first
	anonymizationPatterns := []struct {
		pattern     string
		replacement string
	}{
		// SSN pattern (must come before phone)
		{`\b\d{3}-\d{2}-\d{4}\b`, "[SSN_REDACTED]"},
		// Credit card numbers (must come before phone)
		{`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`, "[CARD_REDACTED]"},
		// Email addresses
		{`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "[EMAIL_REDACTED]"},
		// IP addresses
		{`\b(?:\d{1,3}\.){3}\d{1,3}\b`, "[IP_REDACTED]"},
		// Phone numbers (simple pattern - less specific, so last)
		{`\b\+?[\d\s\-\(\)]{10,}\b`, "[PHONE_REDACTED]"},
	}

	result := content

	for _, rule := range anonymizationPatterns {
		re := regexp.MustCompile(rule.pattern)
		result = re.ReplaceAllString(result, rule.replacement)
	}

	return result
}

// HashSensitiveData creates a one-way hash of sensitive data for comparison
func (em *EncryptionManager) HashSensitiveData(data string) string {
	if data == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// ValidateEncryption validates that encryption is working correctly
func (em *EncryptionManager) ValidateEncryption() error {
	if !em.enabled {
		return nil
	}

	testData := "test encryption validation"

	// Test encryption
	encrypted, err := em.EncryptString(testData)
	if err != nil {
		return fmt.Errorf("encryption test failed: %w", err)
	}

	// Test decryption
	decrypted, err := em.DecryptString(encrypted)
	if err != nil {
		return fmt.Errorf("decryption test failed: %w", err)
	}

	if decrypted != testData {
		return errors.New("encryption validation failed: data mismatch")
	}

	return nil
}

// RotateKey rotates the encryption key (in a real implementation)
func (em *EncryptionManager) RotateKey(newPassword string) error {
	if !em.enabled {
		return errors.New("encryption is not enabled")
	}

	// In a real implementation, this would:
	// 1. Decrypt all data with old key
	// 2. Generate new key from new password
	// 3. Re-encrypt all data with new key
	// 4. Update the master key

	// Generate new master key
	salt := make([]byte, em.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	newKey := pbkdf2.Key([]byte(newPassword), salt, em.iterations, em.keyLength, sha256.New)

	// In production, you'd need to re-encrypt all existing data here
	em.masterKey = newKey

	return nil
}

// IsEnabled returns whether encryption is enabled
func (em *EncryptionManager) IsEnabled() bool {
	return em.enabled
}

// Enable enables encryption with a master password
func (em *EncryptionManager) Enable(masterPassword string) error {
	if masterPassword == "" {
		return errors.New("master password is required")
	}

	// Generate master key
	salt := make([]byte, em.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	em.masterKey = pbkdf2.Key([]byte(masterPassword), salt, em.iterations, em.keyLength, sha256.New)
	em.enabled = true

	return em.ValidateEncryption()
}

// Disable disables encryption
func (em *EncryptionManager) Disable() {
	em.enabled = false
	em.masterKey = nil
}
