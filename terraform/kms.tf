resource "aws_kms_key" "jwt" {
  description              = "${local.workspace_prefix}JWT signing key"
  deletion_window_in_days  = 7
  key_usage                = "SIGN_VERIFY"
  customer_master_key_spec = "ECC_NIST_P256"

  tags = {
    Name      = "${local.workspace_prefix}jwt-key"
    Workspace = terraform.workspace
  }
}

resource "aws_kms_alias" "jwt" {
  name          = "alias/${local.workspace_prefix}jwt-key"
  target_key_id = aws_kms_key.jwt.key_id
}

resource "aws_kms_key" "jwt_encryption" {
  description             = "${local.workspace_prefix}JWT claims encryption key"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  key_usage               = "ENCRYPT_DECRYPT"

  tags = {
    Name      = "${local.workspace_prefix}jwt-encryption-key"
    Workspace = terraform.workspace
  }
}

resource "aws_kms_alias" "jwt_encryption" {
  name          = "alias/${local.workspace_prefix}jwt-encryption-key"
  target_key_id = aws_kms_key.jwt_encryption.key_id
}