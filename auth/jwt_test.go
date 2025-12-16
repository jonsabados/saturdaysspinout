package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_CreateToken(t *testing.T) {
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	idGenerator := func() string { return "test-session-id" }
	service, err := NewJWTService(privateKey, encryptionKey, idGenerator, "test-issuer", time.Hour)
	require.NoError(t, err)

	tokenExpiry := time.Now().Add(time.Hour)
	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token-123", "refresh-token-456", tokenExpiry)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// JWT should have 3 parts separated by dots
	assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
}

func TestJWTService_CreateAndValidateToken(t *testing.T) {
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	idGenerator := func() string { return "test-session-id" }
	service, err := NewJWTService(privateKey, encryptionKey, idGenerator, "test-issuer", time.Hour)
	require.NoError(t, err)

	// Create token
	tokenExpiry := time.Now().Add(time.Hour)
	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token-123", "refresh-token-456", tokenExpiry)
	require.NoError(t, err)

	// Validate token
	sessionClaims, sensitiveClaims, err := service.ValidateToken(ctx, token)
	require.NoError(t, err)

	// Verify session claims
	assert.Equal(t, "test-session-id", sessionClaims.SessionID)
	assert.Equal(t, int64(12345), sessionClaims.IRacingUserID)
	assert.Equal(t, "TestDriver", sessionClaims.IRacingUserName)
	assert.Equal(t, "test-issuer", sessionClaims.Issuer)
	assert.Equal(t, "12345", sessionClaims.Subject)

	// Verify sensitive claims were decrypted correctly
	assert.Equal(t, "access-token-123", sensitiveClaims.IRacingAccessToken)
	assert.Equal(t, "refresh-token-456", sensitiveClaims.IRacingRefreshToken)
	assert.Equal(t, tokenExpiry.Unix(), sensitiveClaims.IRacingTokenExpiry)
}

func TestJWTService_ValidateToken_InvalidSignature(t *testing.T) {
	ctx := context.Background()

	// Generate two different key pairs
	privateKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	idGenerator := func() string { return "test-session-id" }

	// Create token with key1
	service1, err := NewJWTService(privateKey1, encryptionKey, idGenerator, "test-issuer", time.Hour)
	require.NoError(t, err)

	token, err := service1.CreateToken(ctx, 12345, "TestDriver", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	// Try to validate with key2 (should fail)
	service2, err := NewJWTService(privateKey2, encryptionKey, idGenerator, "test-issuer", time.Hour)
	require.NoError(t, err)

	_, _, err = service2.ValidateToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing token")
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	// Create service with very short token expiry (negative = already expired)
	idGenerator := func() string { return "test-session-id" }
	service, err := NewJWTService(privateKey, encryptionKey, idGenerator, "test-issuer", -time.Hour)
	require.NoError(t, err)

	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	_, _, err = service.ValidateToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestJWTService_InvalidEncryptionKeyLength(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 16 bytes is too short for AES-256
	shortKey := make([]byte, 16)
	_, err = rand.Read(shortKey)
	require.NoError(t, err)

	idGenerator := func() string { return "test-session-id" }
	_, err = NewJWTService(privateKey, shortKey, idGenerator, "test-issuer", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryption key must be 32 bytes")
}

func TestParseSigningKeyPEM(t *testing.T) {
	// Generate a key and convert to PEM (same as Terraform's tls_private_key does)
	originalKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	derBytes, err := x509.MarshalECPrivateKey(originalKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)

	// Test successful parsing
	parsedKey, err := ParseSigningKeyPEM(pemData)
	require.NoError(t, err)
	assert.Equal(t, originalKey.D, parsedKey.D)
	assert.Equal(t, originalKey.PublicKey.X, parsedKey.PublicKey.X)
	assert.Equal(t, originalKey.PublicKey.Y, parsedKey.PublicKey.Y)

	// Test with invalid PEM
	_, err = ParseSigningKeyPEM([]byte("not valid pem"))
	assert.Error(t, err)

	// Test with empty input
	_, err = ParseSigningKeyPEM([]byte{})
	assert.Error(t, err)
}

func TestParseEncryptionKeyBase64(t *testing.T) {
	// Valid 32-byte key in base64
	validKey := "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=" // "01234567890123456789012345678901"

	key, err := ParseEncryptionKeyBase64(validKey)
	require.NoError(t, err)
	assert.Len(t, key, 32)

	// Invalid base64
	_, err = ParseEncryptionKeyBase64("not valid base64!!!")
	assert.Error(t, err)

	// Valid base64 but wrong length (16 bytes)
	shortKey := "MDEyMzQ1Njc4OTAxMjM0NQ==" // "0123456789012345"
	_, err = ParseEncryptionKeyBase64(shortKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be 32 bytes")
}