resource "aws_s3_bucket" "iracing_cache" {
  bucket = "${local.workspace_prefix}iracing-cache-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_public_access_block" "iracing_cache" {
  bucket = aws_s3_bucket.iracing_cache.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "iracing_cache" {
  bucket = aws_s3_bucket.iracing_cache.id

  rule {
    id     = "expire-old-cache"
    status = "Enabled"

    expiration {
      days = 7
    }
  }
}