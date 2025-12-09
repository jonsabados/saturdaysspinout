package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// KMSSigner handles cryptographic operations via KMS
type KMSSigner interface {
	Sign(ctx context.Context, digest []byte) ([]byte, error)
	GetPublicKey(ctx context.Context) (*ecdsa.PublicKey, error)
}

// KMSEncryptor handles envelope encryption via KMS
type KMSEncryptor interface {
	GenerateDataKey(ctx context.Context) (plaintext []byte, ciphertext []byte, err error)
	DecryptDataKey(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// KMSClient is the interface for AWS KMS operations
type KMSClient interface {
	Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error)
	GetPublicKey(ctx context.Context, keyID string) ([]byte, error)
	GenerateDataKey(ctx context.Context, keyID string, keySpec string) (plaintext []byte, ciphertext []byte, err error)
	Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}

// EncryptedClaims contains the iRacing tokens in encrypted form
type EncryptedClaims struct {
	EncryptedData string `json:"enc"`
	EncryptedKey  string `json:"key"`
	Nonce         string `json:"nonce"`
}

// SensitiveClaims contains the iRacing tokens that get encrypted
type SensitiveClaims struct {
	IRacingAccessToken  string `json:"irt"`
	IRacingRefreshToken string `json:"irrt"`
	IRacingTokenExpiry  int64  `json:"irte"`
}

// SessionClaims represents the JWT claims for a user session
type SessionClaims struct {
	jwt.RegisteredClaims
	SessionID       string          `json:"sid"`
	IRacingUserID   int             `json:"ir_uid"`
	IRacingUserName string          `json:"ir_name"`
	Encrypted       EncryptedClaims `json:"encrypted"`
}

// IDGenerator is a function that generates unique IDs
type IDGenerator func() string

// JWTService handles creating and validating JWTs
type JWTService struct {
	signer      KMSSigner
	encryptor   KMSEncryptor
	idGenerator IDGenerator
	issuer      string
	tokenExpiry time.Duration
}

// NewJWTService creates a new JWTService
func NewJWTService(signer KMSSigner, encryptor KMSEncryptor, idGenerator IDGenerator, issuer string, tokenExpiry time.Duration) *JWTService {
	return &JWTService{
		signer:      signer,
		encryptor:   encryptor,
		idGenerator: idGenerator,
		issuer:      issuer,
		tokenExpiry: tokenExpiry,
	}
}

func (s *JWTService) CreateToken(ctx context.Context, userID int, userName string, accessToken, refreshToken string, tokenExpiry time.Time) (string, error) {
	encryptedClaims, err := s.encryptSensitiveClaims(ctx, &SensitiveClaims{
		IRacingAccessToken:  accessToken,
		IRacingRefreshToken: refreshToken,
		IRacingTokenExpiry:  tokenExpiry.Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("encrypting sensitive claims: %w", err)
	}

	now := time.Now()
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
		SessionID:       s.idGenerator(),
		IRacingUserID:   userID,
		IRacingUserName: userName,
		Encrypted:       *encryptedClaims,
	}

	token := jwt.NewWithClaims(&kmsSigningMethod{signer: s.signer, ctx: ctx}, claims)

	signedString, err := token.SignedString(nil)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signedString, nil
}

func (s *JWTService) ValidateToken(ctx context.Context, tokenString string) (*SessionClaims, *SensitiveClaims, error) {
	pubKey, err := s.signer.GetPublicKey(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting public key: %w", err)
	}

	claims := &SessionClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parsing token: %w", err)
	}

	if !token.Valid {
		return nil, nil, errors.New("invalid token")
	}

	sensitiveClaims, err := s.decryptSensitiveClaims(ctx, &claims.Encrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypting sensitive claims: %w", err)
	}

	return claims, sensitiveClaims, nil
}

func (s *JWTService) encryptSensitiveClaims(ctx context.Context, claims *SensitiveClaims) (*EncryptedClaims, error) {
	plaintext, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshaling claims: %w", err)
	}

	dataKey, encryptedDataKey, err := s.encryptor.GenerateDataKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating data key: %w", err)
	}

	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedClaims{
		EncryptedData: base64.RawURLEncoding.EncodeToString(ciphertext),
		EncryptedKey:  base64.RawURLEncoding.EncodeToString(encryptedDataKey),
		Nonce:         base64.RawURLEncoding.EncodeToString(nonce),
	}, nil
}

func (s *JWTService) decryptSensitiveClaims(ctx context.Context, encrypted *EncryptedClaims) (*SensitiveClaims, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	encryptedKey, err := base64.RawURLEncoding.DecodeString(encrypted.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decoding encrypted key: %w", err)
	}

	nonce, err := base64.RawURLEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}

	dataKey, err := s.encryptor.DecryptDataKey(ctx, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting data key: %w", err)
	}

	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintextBytes, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	var result SensitiveClaims
	if err := json.Unmarshal(plaintextBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling claims: %w", err)
	}

	return &result, nil
}

// kmsSigningMethod implements jwt.SigningMethod using KMS
type kmsSigningMethod struct {
	signer KMSSigner
	ctx    context.Context
}

func (m *kmsSigningMethod) Alg() string {
	return "ES256"
}

func (m *kmsSigningMethod) Verify(signingString string, sig []byte, key interface{}) error {
	return jwt.ErrSignatureInvalid
}

func (m *kmsSigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	hasher := jwt.SigningMethodES256.Hash.New()
	hasher.Write([]byte(signingString))
	digest := hasher.Sum(nil)

	signature, err := m.signer.Sign(m.ctx, digest)
	if err != nil {
		return nil, fmt.Errorf("KMS signing: %w", err)
	}

	return signature, nil
}

// KMSSignerAdapter adapts a KMSClient to the KMSSigner interface
type KMSSignerAdapter struct {
	client    KMSClient
	keyID     string
	publicKey *ecdsa.PublicKey
}

// NewKMSSignerAdapter creates a new KMSSignerAdapter
func NewKMSSignerAdapter(client KMSClient, keyID string) *KMSSignerAdapter {
	return &KMSSignerAdapter{
		client: client,
		keyID:  keyID,
	}
}

func (s *KMSSignerAdapter) Sign(ctx context.Context, digest []byte) ([]byte, error) {
	return s.client.Sign(ctx, s.keyID, digest)
}

func (s *KMSSignerAdapter) GetPublicKey(ctx context.Context) (*ecdsa.PublicKey, error) {
	if s.publicKey != nil {
		return s.publicKey, nil
	}

	pubKeyBytes, err := s.client.GetPublicKey(ctx, s.keyID)
	if err != nil {
		return nil, err
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not ECDSA")
	}

	s.publicKey = ecdsaKey
	return ecdsaKey, nil
}

// KMSEncryptorAdapter adapts a KMSClient to the KMSEncryptor interface
type KMSEncryptorAdapter struct {
	client KMSClient
	keyID  string
}

// NewKMSEncryptorAdapter creates a new KMSEncryptorAdapter
func NewKMSEncryptorAdapter(client KMSClient, keyID string) *KMSEncryptorAdapter {
	return &KMSEncryptorAdapter{
		client: client,
		keyID:  keyID,
	}
}

func (e *KMSEncryptorAdapter) GenerateDataKey(ctx context.Context) (plaintext []byte, ciphertext []byte, err error) {
	return e.client.GenerateDataKey(ctx, e.keyID, "AES_256")
}

func (e *KMSEncryptorAdapter) DecryptDataKey(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return e.client.Decrypt(ctx, e.keyID, ciphertext)
}