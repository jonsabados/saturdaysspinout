resource "aws_sqs_queue" "race_ingestion_requests" {
  name = "${local.workspace_prefix}SaturdaysSpinoutRaceIngestionRequests"

  visibility_timeout_seconds = 300
  message_retention_seconds  = 900 # 15 minutes - best effort, tokens are short-lived
  receive_wait_time_seconds  = 20  # Long polling

  sqs_managed_sse_enabled = true
}

resource "aws_sqs_queue" "race_ingestion_requests_dlq" {
  name = "${local.workspace_prefix}SaturdaysSpinoutRaceIngestionRequestsDLQ"

  message_retention_seconds = 86400 # 1 day - keep failed messages longer for debugging

  sqs_managed_sse_enabled = true
}

resource "aws_sqs_queue_redrive_policy" "race_ingestion_requests" {
  queue_url = aws_sqs_queue.race_ingestion_requests.id

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.race_ingestion_requests_dlq.arn
    maxReceiveCount     = 3
  })
}

resource "aws_iam_role" "race_ingestion_lambda" {
  name               = "${local.workspace_prefix}SaturdaysSpinoutRaceIngestion"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role_policy.json
}

data "aws_iam_policy_document" "race_ingestion_lambda" {
  statement {
    sid    = "AllowLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "${aws_cloudwatch_log_group.race_ingestion_lambda_logs.arn}:*"
    ]
  }

  statement {
    sid    = "AllowXRayWrite"
    effect = "Allow"
    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords",
      "xray:GetSamplingRules",
      "xray:GetSamplingTargets",
      "xray:GetSamplingStatisticSummaries"
    ]
    resources = ["*"]
  }

  statement {
    sid    = "AllowSQSConsume"
    effect = "Allow"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes"
    ]
    resources = [
      aws_sqs_queue.race_ingestion_requests.arn
    ]
  }

  statement {
    sid    = "AllowDynamoDB"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:UpdateItem",
      "dynamodb:PutItem",
      "dynamodb:Query"
    ]
    resources = [
      aws_dynamodb_table.application_store.arn
    ]
  }
}

resource "aws_iam_role_policy" "race_ingestion_lambda" {
  role   = aws_iam_role.race_ingestion_lambda.name
  policy = data.aws_iam_policy_document.race_ingestion_lambda.json
}

resource "aws_lambda_function" "race_ingestion_lambda" {
  filename                       = "../dist/raceIngestionProcessorLambda.zip"
  source_code_hash               = filebase64sha256("../dist/raceIngestionProcessorLambda.zip")
  timeout                        = 300
  reserved_concurrent_executions = var.race_ingestion_processor_concurrency
  runtime                        = "provided.al2"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  function_name                  = "${local.workspace_prefix}SaturdaysSpinoutRaceIngestion"
  role                           = aws_iam_role.race_ingestion_lambda.arn

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      LOG_LEVEL      = "info"
      DYNAMODB_TABLE = aws_dynamodb_table.application_store.name
    }
  }
}

resource "aws_cloudwatch_log_group" "race_ingestion_lambda_logs" {
  name              = "/aws/lambda/${local.workspace_prefix}SaturdaysSpinoutRaceIngestion"
  retention_in_days = 7
}

resource "aws_lambda_event_source_mapping" "race_ingestion_sqs" {
  event_source_arn                   = aws_sqs_queue.race_ingestion_requests.arn
  function_name                      = aws_lambda_function.race_ingestion_lambda.arn
  batch_size                         = 1
  maximum_batching_window_in_seconds = 0
  scaling_config {
    maximum_concurrency = var.race_ingestion_processor_concurrency
  }
}

output "race_ingestion_queue_url" {
  value = aws_sqs_queue.race_ingestion_requests.url
}

output "race_ingestion_queue_arn" {
  value = aws_sqs_queue.race_ingestion_requests.arn
}
