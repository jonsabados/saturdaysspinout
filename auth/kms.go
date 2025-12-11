package auth

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type AWSKMS interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
	GenerateDataKey(ctx context.Context, params *kms.GenerateDataKeyInput, optFns ...func(*kms.Options)) (*kms.GenerateDataKeyOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

type AWSKMSClient struct {
	kms AWSKMS
}

func NewAWSKMSClient(kmsClient AWSKMS) *AWSKMSClient {
	return &AWSKMSClient{kms: kmsClient}
}

func (c *AWSKMSClient) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	result, err := c.kms.Sign(ctx, &kms.SignInput{
		KeyId:            &keyID,
		Message:          digest,
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
	})
	if err != nil {
		return nil, fmt.Errorf("KMS Sign: %w", err)
	}
	return result.Signature, nil
}

func (c *AWSKMSClient) GetPublicKey(ctx context.Context, keyID string) ([]byte, error) {
	result, err := c.kms.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: &keyID,
	})
	if err != nil {
		return nil, fmt.Errorf("KMS GetPublicKey: %w", err)
	}
	return result.PublicKey, nil
}

func (c *AWSKMSClient) GenerateDataKey(ctx context.Context, keyID string, keySpec string) (plaintext []byte, ciphertext []byte, err error) {
	spec := types.DataKeySpecAes256
	if keySpec == "AES_128" {
		spec = types.DataKeySpecAes128
	}

	result, err := c.kms.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   &keyID,
		KeySpec: spec,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("KMS GenerateDataKey: %w", err)
	}
	return result.Plaintext, result.CiphertextBlob, nil
}

func (c *AWSKMSClient) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	result, err := c.kms.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          &keyID,
		CiphertextBlob: ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("KMS Decrypt: %w", err)
	}
	return result.Plaintext, nil
}
