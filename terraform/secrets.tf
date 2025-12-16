data "aws_secretsmanager_secret" "iracing_credentials" {
  name = "iracing_credentials"
}

# JWT Signing Key (ECDSA P-256)
resource "tls_private_key" "jwt_signing" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "aws_secretsmanager_secret" "jwt_signing_key" {
  name                    = "${local.workspace_prefix}jwt-signing-key"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "jwt_signing_key" {
  secret_id     = aws_secretsmanager_secret.jwt_signing_key.id
  secret_string = tls_private_key.jwt_signing.private_key_pem
}

# JWT Encryption Key (AES-256, 32 bytes)
resource "random_bytes" "jwt_encryption" {
  length = 32
}

resource "aws_secretsmanager_secret" "jwt_encryption_key" {
  name                    = "${local.workspace_prefix}jwt-encryption-key"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "jwt_encryption_key" {
  secret_id     = aws_secretsmanager_secret.jwt_encryption_key.id
  secret_string = random_bytes.jwt_encryption.base64
}
