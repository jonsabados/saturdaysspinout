package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jonsabados/saturdaysspinout/auth/mocks"
)

func TestJWTService_CreateToken(t *testing.T) {
	ctx := context.Background()

	// Generate a real ECDSA key for signing (we'll use it to create valid signatures)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 32-byte key for AES-256
	dataKey := make([]byte, 32)
	_, err = rand.Read(dataKey)
	require.NoError(t, err)

	encryptedDataKey := []byte("encrypted-data-key")

	mockSigner := mocks.NewMockKMSSigner(t)
	mockEncryptor := mocks.NewMockKMSEncryptor(t)

	// Set up encryptor expectations
	mockEncryptor.EXPECT().
		GenerateDataKey(mock.Anything).
		Return(dataKey, encryptedDataKey, nil)

	// Set up signer to use the real private key for signing
	mockSigner.EXPECT().
		Sign(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, digest []byte) ([]byte, error) {
			r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
			if err != nil {
				return nil, err
			}
			// Convert to fixed-size format for ES256
			curveBits := privateKey.Curve.Params().BitSize
			keyBytes := curveBits / 8
			if curveBits%8 > 0 {
				keyBytes++
			}
			sig := make([]byte, 2*keyBytes)
			r.FillBytes(sig[:keyBytes])
			s.FillBytes(sig[keyBytes:])
			return sig, nil
		})

	idGenerator := func() string { return "test-session-id" }
	service := NewJWTService(mockSigner, mockEncryptor, idGenerator, "test-issuer", time.Hour)

	tokenExpiry := time.Now().Add(time.Hour)
	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token-123", "refresh-token-456", tokenExpiry)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// JWT should have 3 parts separated by dots
	assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
}

func TestJWTService_CreateAndValidateToken(t *testing.T) {
	ctx := context.Background()

	// Generate a real ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 32-byte key for AES-256
	dataKey := make([]byte, 32)
	_, err = rand.Read(dataKey)
	require.NoError(t, err)

	encryptedDataKey := []byte("encrypted-data-key")

	mockSigner := mocks.NewMockKMSSigner(t)
	mockEncryptor := mocks.NewMockKMSEncryptor(t)

	// Set up encryptor for token creation
	mockEncryptor.EXPECT().
		GenerateDataKey(mock.Anything).
		Return(dataKey, encryptedDataKey, nil)

	// Set up signer for token creation
	mockSigner.EXPECT().
		Sign(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, digest []byte) ([]byte, error) {
			r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
			if err != nil {
				return nil, err
			}
			curveBits := privateKey.Curve.Params().BitSize
			keyBytes := curveBits / 8
			if curveBits%8 > 0 {
				keyBytes++
			}
			sig := make([]byte, 2*keyBytes)
			r.FillBytes(sig[:keyBytes])
			s.FillBytes(sig[keyBytes:])
			return sig, nil
		})

	// Set up signer for token validation (returns public key)
	mockSigner.EXPECT().
		GetPublicKey(mock.Anything).
		Return(&privateKey.PublicKey, nil)

	// Set up encryptor for token validation (decrypt the data key)
	mockEncryptor.EXPECT().
		DecryptDataKey(mock.Anything, encryptedDataKey).
		Return(dataKey, nil)

	idGenerator := func() string { return "test-session-id" }
	service := NewJWTService(mockSigner, mockEncryptor, idGenerator, "test-issuer", time.Hour)

	// Create token
	tokenExpiry := time.Now().Add(time.Hour)
	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token-123", "refresh-token-456", tokenExpiry)
	require.NoError(t, err)

	// Validate token
	sessionClaims, sensitiveClaims, err := service.ValidateToken(ctx, token)
	require.NoError(t, err)

	// Verify session claims
	assert.Equal(t, "test-session-id", sessionClaims.SessionID)
	assert.Equal(t, 12345, sessionClaims.IRacingUserID)
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

	dataKey := make([]byte, 32)
	_, err = rand.Read(dataKey)
	require.NoError(t, err)

	encryptedDataKey := []byte("encrypted-data-key")

	mockSigner := mocks.NewMockKMSSigner(t)
	mockEncryptor := mocks.NewMockKMSEncryptor(t)

	// Create token signed with key1
	mockEncryptor.EXPECT().
		GenerateDataKey(mock.Anything).
		Return(dataKey, encryptedDataKey, nil)

	mockSigner.EXPECT().
		Sign(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, digest []byte) ([]byte, error) {
			r, s, err := ecdsa.Sign(rand.Reader, privateKey1, digest)
			if err != nil {
				return nil, err
			}
			curveBits := privateKey1.Curve.Params().BitSize
			keyBytes := curveBits / 8
			if curveBits%8 > 0 {
				keyBytes++
			}
			sig := make([]byte, 2*keyBytes)
			r.FillBytes(sig[:keyBytes])
			s.FillBytes(sig[keyBytes:])
			return sig, nil
		})

	// Validate with key2's public key (should fail)
	mockSigner.EXPECT().
		GetPublicKey(mock.Anything).
		Return(&privateKey2.PublicKey, nil)

	idGenerator := func() string { return "test-session-id" }
	service := NewJWTService(mockSigner, mockEncryptor, idGenerator, "test-issuer", time.Hour)

	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	_, _, err = service.ValidateToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing token")
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	dataKey := make([]byte, 32)
	_, err = rand.Read(dataKey)
	require.NoError(t, err)

	encryptedDataKey := []byte("encrypted-data-key")

	mockSigner := mocks.NewMockKMSSigner(t)
	mockEncryptor := mocks.NewMockKMSEncryptor(t)

	mockEncryptor.EXPECT().
		GenerateDataKey(mock.Anything).
		Return(dataKey, encryptedDataKey, nil)

	mockSigner.EXPECT().
		Sign(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, digest []byte) ([]byte, error) {
			r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest)
			if err != nil {
				return nil, err
			}
			curveBits := privateKey.Curve.Params().BitSize
			keyBytes := curveBits / 8
			if curveBits%8 > 0 {
				keyBytes++
			}
			sig := make([]byte, 2*keyBytes)
			r.FillBytes(sig[:keyBytes])
			s.FillBytes(sig[keyBytes:])
			return sig, nil
		})

	mockSigner.EXPECT().
		GetPublicKey(mock.Anything).
		Return(&privateKey.PublicKey, nil)

	// Create service with very short token expiry
	idGenerator := func() string { return "test-session-id" }
	service := NewJWTService(mockSigner, mockEncryptor, idGenerator, "test-issuer", -time.Hour)

	token, err := service.CreateToken(ctx, 12345, "TestDriver", "access-token", "refresh-token", time.Now().Add(time.Hour))
	require.NoError(t, err)

	_, _, err = service.ValidateToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestKMSSignerAdapter(t *testing.T) {
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	mockClient := mocks.NewMockKMSClient(t)

	testDigest := []byte("test-digest")
	testSignature := []byte("test-signature")
	testKeyID := "test-key-id"

	mockClient.EXPECT().
		Sign(ctx, testKeyID, testDigest).
		Return(testSignature, nil)

	adapter := NewKMSSignerAdapter(mockClient, testKeyID)
	sig, err := adapter.Sign(ctx, testDigest)

	require.NoError(t, err)
	assert.Equal(t, testSignature, sig)

	// Test GetPublicKey
	// Marshal the public key to DER format
	pubKeyBytes, err := ecdsaPublicKeyToDER(&privateKey.PublicKey)
	require.NoError(t, err)

	mockClient.EXPECT().
		GetPublicKey(ctx, testKeyID).
		Return(pubKeyBytes, nil)

	pubKey, err := adapter.GetPublicKey(ctx)
	require.NoError(t, err)
	assert.Equal(t, privateKey.PublicKey.X, pubKey.X)
	assert.Equal(t, privateKey.PublicKey.Y, pubKey.Y)

	// Test caching - GetPublicKey should not call client again
	pubKey2, err := adapter.GetPublicKey(ctx)
	require.NoError(t, err)
	assert.Equal(t, pubKey, pubKey2)
}

func TestKMSEncryptorAdapter(t *testing.T) {
	ctx := context.Background()

	mockClient := mocks.NewMockKMSClient(t)
	testKeyID := "test-key-id"

	plaintext := []byte("plaintext-key")
	ciphertext := []byte("ciphertext-key")

	mockClient.EXPECT().
		GenerateDataKey(ctx, testKeyID, "AES_256").
		Return(plaintext, ciphertext, nil)

	adapter := NewKMSEncryptorAdapter(mockClient, testKeyID)
	gotPlaintext, gotCiphertext, err := adapter.GenerateDataKey(ctx)

	require.NoError(t, err)
	assert.Equal(t, plaintext, gotPlaintext)
	assert.Equal(t, ciphertext, gotCiphertext)

	// Test DecryptDataKey
	mockClient.EXPECT().
		Decrypt(ctx, testKeyID, ciphertext).
		Return(plaintext, nil)

	decrypted, err := adapter.DecryptDataKey(ctx, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// Helper to convert ECDSA public key to DER format
func ecdsaPublicKeyToDER(pub *ecdsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(pub)
}