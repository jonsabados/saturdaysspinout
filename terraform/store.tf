resource "aws_dynamodb_table" "application_store" {
  name         = "${local.workspace_prefix}SaturdaysSpinoutAppData"
  billing_mode = "PAY_PER_REQUEST"

  hash_key  = "partition_key"
  range_key = "sort_key"

  attribute {
    name = "partition_key"
    type = "S"
  }

  attribute {
    name = "sort_key"
    type = "S"
  }
}