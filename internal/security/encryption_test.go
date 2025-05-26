package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptionManager(t *testing.T) {
	tests := []struct {
		name          string
		masterPassword string
		wantEnabled   bool
	}{
		{
			name:          "With password",
			masterPassword: "test-password-123",
			wantEnabled:   true,
		},
		{
			name:          "Without password",
			masterPassword: "",
			wantEnabled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := NewEncryptionManager(tt.masterPassword)
			assert.Equal(t, tt.wantEnabled, em.IsEnabled())
		})
	}
}

func TestEncryptionManager_EncryptDecryptString(t *testing.T) {
	em := NewEncryptionManager("test-password")

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "Normal text",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "Empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "Special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:   false,
		},
		{
			name:      "Unicode text",
			plaintext: "Hello 世界 🌍",
			wantErr:   false,
		},
		{
			name:      "Large text",
			plaintext: strings.Repeat("A", 10000),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := em.EncryptString(tt.plaintext)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, encrypted)
			assert.Equal(t, "aes-gcm", encrypted.Algorithm)
			
			// Ensure encrypted data is different from plaintext (unless empty)
			if tt.plaintext != "" {
				assert.NotEqual(t, tt.plaintext, encrypted.Data)
			}

			// Decrypt
			decrypted, err := em.DecryptString(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptionManager_DisabledEncryption(t *testing.T) {
	em := NewEncryptionManager("") // Disabled

	plaintext := "test data"
	
	// Encrypt with disabled encryption
	encrypted, err := em.EncryptString(plaintext)
	require.NoError(t, err)
	assert.Equal(t, "none", encrypted.Algorithm)
	assert.Equal(t, plaintext, encrypted.Data)

	// Decrypt with disabled encryption
	decrypted, err := em.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptionManager_EncryptSensitiveFields(t *testing.T) {
	em := NewEncryptionManager("test-password")

	tests := []struct {
		name     string
		content  string
		contains []string
		excludes []string
	}{
		{
			name:     "API key",
			content:  "api_key: sk-1234567890abcdef",
			contains: []string{"[ENCRYPTED:api_key:"},
			excludes: []string{"sk-1234567890abcdef"},
		},
		{
			name:     "Password",
			content:  "password: mysecretpass123",
			contains: []string{"[ENCRYPTED:password:"},
			excludes: []string{"mysecretpass123"},
		},
		{
			name:     "Multiple secrets",
			content:  "api_key=key123\npassword=pass456\ntoken=tok789",
			contains: []string{"[ENCRYPTED:api_key:", "[ENCRYPTED:password:", "[ENCRYPTED:token:"},
			excludes: []string{"key123", "pass456", "tok789"},
		},
		{
			name:     "No sensitive data",
			content:  "This is just normal text without secrets",
			contains: []string{"This is just normal text without secrets"},
			excludes: []string{"[ENCRYPTED:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := em.EncryptSensitiveFields(tt.content)
			require.NoError(t, err)

			for _, expected := range tt.contains {
				assert.Contains(t, encrypted, expected)
			}

			for _, unexpected := range tt.excludes {
				assert.NotContains(t, encrypted, unexpected)
			}
		})
	}
}

func TestEncryptionManager_AnonymizeData(t *testing.T) {
	em := NewEncryptionManager("")

	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:    "Email address",
			content: "Contact me at john.doe@example.com",
			expected: map[string]string{
				"john.doe@example.com": "[EMAIL_REDACTED]",
			},
		},
		{
			name:    "IP address",
			content: "Server IP: 192.168.1.100",
			expected: map[string]string{
				"192.168.1.100": "[IP_REDACTED]",
			},
		},
		{
			name:    "Phone number",
			content: "Call me at +1-555-123-4567",
			expected: map[string]string{
				"+1-555-123-4567": "[PHONE_REDACTED]",
			},
		},
		{
			name:    "Credit card",
			content: "Card: 1234 5678 9012 3456",
			expected: map[string]string{
				"1234 5678 9012 3456": "[CARD_REDACTED]",
			},
		},
		{
			name:    "SSN",
			content: "SSN: 123-45-6789",
			expected: map[string]string{
				"123-45-6789": "[SSN_REDACTED]",
			},
		},
		{
			name:    "Multiple PII",
			content: "Email: test@example.com, IP: 10.0.0.1",
			expected: map[string]string{
				"test@example.com": "[EMAIL_REDACTED]",
				"10.0.0.1":        "[IP_REDACTED]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := em.AnonymizeData(tt.content)

			for original, redacted := range tt.expected {
				assert.NotContains(t, result, original)
				assert.Contains(t, result, redacted)
			}
		})
	}
}

func TestEncryptionManager_HashSensitiveData(t *testing.T) {
	em := NewEncryptionManager("")

	tests := []struct {
		name string
		data string
	}{
		{
			name: "Normal string",
			data: "test data",
		},
		{
			name: "Empty string",
			data: "",
		},
		{
			name: "Special characters",
			data: "!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := em.HashSensitiveData(tt.data)
			hash2 := em.HashSensitiveData(tt.data)

			if tt.data == "" {
				assert.Empty(t, hash1)
			} else {
				assert.NotEmpty(t, hash1)
				assert.Equal(t, hash1, hash2) // Same input produces same hash
				assert.NotEqual(t, tt.data, hash1) // Hash is different from input
			}
		})
	}

	// Different inputs produce different hashes
	hash1 := em.HashSensitiveData("data1")
	hash2 := em.HashSensitiveData("data2")
	assert.NotEqual(t, hash1, hash2)
}

func TestEncryptionManager_ValidateEncryption(t *testing.T) {
	// Enabled encryption
	em1 := NewEncryptionManager("test-password")
	err := em1.ValidateEncryption()
	assert.NoError(t, err)

	// Disabled encryption
	em2 := NewEncryptionManager("")
	err = em2.ValidateEncryption()
	assert.NoError(t, err)
}

func TestEncryptionManager_EnableDisable(t *testing.T) {
	em := NewEncryptionManager("")
	
	// Initially disabled
	assert.False(t, em.IsEnabled())

	// Enable
	err := em.Enable("new-password")
	require.NoError(t, err)
	assert.True(t, em.IsEnabled())

	// Test encryption works after enabling
	plaintext := "test data"
	encrypted, err := em.EncryptString(plaintext)
	require.NoError(t, err)
	assert.Equal(t, "aes-gcm", encrypted.Algorithm)

	decrypted, err := em.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Disable
	em.Disable()
	assert.False(t, em.IsEnabled())

	// Test encryption is disabled
	encrypted2, err := em.EncryptString(plaintext)
	require.NoError(t, err)
	assert.Equal(t, "none", encrypted2.Algorithm)
	assert.Equal(t, plaintext, encrypted2.Data)
}

func TestEncryptionManager_RotateKey(t *testing.T) {
	em := NewEncryptionManager("old-password")

	// Encrypt with old key
	plaintext := "sensitive data"
	encrypted, err := em.EncryptString(plaintext)
	require.NoError(t, err)

	// Rotate key
	err = em.RotateKey("new-password")
	require.NoError(t, err)

	// Note: In a real implementation, old encrypted data would need to be re-encrypted
	// For now, we just test that new encryption works with new key
	newEncrypted, err := em.EncryptString(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, encrypted.Data, newEncrypted.Data) // Different encryption

	// Can decrypt new data
	decrypted, err := em.DecryptString(newEncrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptionManager_EdgeCases(t *testing.T) {
	em := NewEncryptionManager("test-password")

	t.Run("Decrypt with wrong algorithm", func(t *testing.T) {
		encrypted := &EncryptedData{
			Algorithm: "unsupported-algo",
			Data:      "some-data",
		}
		_, err := em.DecryptString(encrypted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported encryption algorithm")
	})

	t.Run("Decrypt with disabled encryption", func(t *testing.T) {
		emDisabled := NewEncryptionManager("")
		encrypted := &EncryptedData{
			Algorithm: "aes-gcm",
			Data:      "encrypted-data",
		}
		_, err := emDisabled.DecryptString(encrypted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "encryption is not enabled")
	})

	t.Run("Enable with empty password", func(t *testing.T) {
		em := NewEncryptionManager("")
		err := em.Enable("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master password is required")
	})

	t.Run("Rotate key with disabled encryption", func(t *testing.T) {
		emDisabled := NewEncryptionManager("")
		err := emDisabled.RotateKey("new-password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "encryption is not enabled")
	})
}

// Benchmark tests
func BenchmarkEncryptString(b *testing.B) {
	em := NewEncryptionManager("test-password")
	plaintext := "This is a test message for benchmarking encryption performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := em.EncryptString(plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptString(b *testing.B) {
	em := NewEncryptionManager("test-password")
	plaintext := "This is a test message for benchmarking decryption performance"
	encrypted, _ := em.EncryptString(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := em.DecryptString(encrypted)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnonymizeData(b *testing.B) {
	em := NewEncryptionManager("")
	content := "Contact john.doe@example.com at 192.168.1.1 or call +1-555-123-4567"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = em.AnonymizeData(content)
	}
}