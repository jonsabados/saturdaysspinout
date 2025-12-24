package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type EncryptedClaims struct {
	EncryptedData string `json:"enc"`
	Nonce         string `json:"nonce"`
}

type SensitiveClaims struct {
	IRacingAccessToken  string `json:"irt"`
	IRacingRefreshToken string `json:"irrt"`
	IRacingTokenExpiry  int64  `json:"irte"`
}

type SessionClaims struct {
	jwt.RegisteredClaims
	SessionID       string          `json:"sid"`
	IRacingUserID   int64           `json:"ir_uid"`
	IRacingUserName string          `json:"ir_name"`
	Entitlements    []string        `json:"ent,omitempty"`
	Encrypted       EncryptedClaims `json:"encrypted"`
}

type IDGenerator func() string

type JWTService struct {
	signingKey    *ecdsa.PrivateKey
	encryptionKey []byte
	gcm           cipher.AEAD
	idGenerator   IDGenerator
	issuer        string
	tokenExpiry   time.Duration
}

func NewJWTService(signingKey *ecdsa.PrivateKey, encryptionKey []byte, idGenerator IDGenerator, issuer string, tokenExpiry time.Duration) (*JWTService, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(encryptionKey))
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &JWTService{
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
		gcm:           gcm,
		idGenerator:   idGenerator,
		issuer:        issuer,
		tokenExpiry:   tokenExpiry,
	}, nil
}

func (s *JWTService) CreateToken(_ context.Context, userID int64, userName string, entitlements []string, accessToken, refreshToken string, tokenExpiry time.Time) (string, error) {
	encryptedClaims, err := s.encryptSensitiveClaims(&SensitiveClaims{
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
		Entitlements:    entitlements,
		Encrypted:       *encryptedClaims,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	signedString, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signedString, nil
}

func (s *JWTService) ValidateToken(_ context.Context, tokenString string) (*SessionClaims, *SensitiveClaims, error) {
	claims := &SessionClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &s.signingKey.PublicKey, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parsing token: %w", err)
	}

	if !token.Valid {
		return nil, nil, errors.New("invalid token")
	}

	sensitiveClaims, err := s.decryptSensitiveClaims(&claims.Encrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypting sensitive claims: %w", err)
	}

	return claims, sensitiveClaims, nil
}

func (s *JWTService) encryptSensitiveClaims(claims *SensitiveClaims) (*EncryptedClaims, error) {
	plaintext, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshaling claims: %w", err)
	}

	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := s.gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedClaims{
		EncryptedData: base64.RawURLEncoding.EncodeToString(ciphertext),
		Nonce:         base64.RawURLEncoding.EncodeToString(nonce),
	}, nil
}

func (s *JWTService) decryptSensitiveClaims(encrypted *EncryptedClaims) (*SensitiveClaims, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	nonce, err := base64.RawURLEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}

	plaintextBytes, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	var result SensitiveClaims
	if err := json.Unmarshal(plaintextBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling claims: %w", err)
	}

	return &result, nil
}