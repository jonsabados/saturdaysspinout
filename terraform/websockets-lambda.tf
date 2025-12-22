locals {
  ws_env_vars = {
    LOG_LEVEL                 = "info"
    JWT_SIGNING_KEY_SECRET    = aws_secretsmanager_secret.jwt_signing_key.arn
    JWT_ENCRYPTION_KEY_SECRET = aws_secretsmanager_secret.jwt_encryption_key.arn
    DYNAMODB_TABLE            = aws_dynamodb_table.application_store.name
    WS_MANAGEMENT_ENDPOINT    = "https://${aws_apigatewayv2_api.websockets.id}.execute-api.us-east-1.amazonaws.com/${aws_apigatewayv2_stage.ws.name}"
  }
}

resource "aws_iam_role" "ws_lambda" {
  name               = "${local.workspace_prefix}SaturdaysSpinoutWS"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role_policy.json
}

data "aws_iam_policy_document" "ws_lambda" {
  statement {
    sid    = "AllowLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "${aws_cloudwatch_log_group.ws_lambda_logs.arn}:*"
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
    sid    = "AllowDynamoDB"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:DeleteItem",
      "dynamodb:Query"
    ]
    resources = [
      aws_dynamodb_table.application_store.arn
    ]
  }

  statement {
    sid    = "AllowSecretsManager"
    effect = "Allow"
    actions = [
      "secretsmanager:GetSecretValue"
    ]
    resources = [
      aws_secretsmanager_secret.jwt_signing_key.arn,
      aws_secretsmanager_secret.jwt_encryption_key.arn,
    ]
  }

  statement {
    sid    = "AllowAPIGatewayManagement"
    effect = "Allow"
    actions = [
      "execute-api:ManageConnections"
    ]
    resources = [
      "arn:aws:execute-api:us-east-1:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.websockets.id}/*"
    ]
  }
}

resource "aws_iam_role_policy" "ws_lambda" {
  role   = aws_iam_role.ws_lambda.name
  policy = data.aws_iam_policy_document.ws_lambda.json
}

resource "aws_lambda_function" "ws_lambda" {
  filename                       = "../dist/websocketLambda.zip"
  source_code_hash               = filebase64sha256("../dist/websocketLambda.zip")
  timeout                        = 15
  # reserved_concurrent_executions = 15
  runtime                        = "provided.al2"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  function_name                  = "${local.workspace_prefix}SaturdaysSpinoutWS"
  role                           = aws_iam_role.ws_lambda.arn

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = local.ws_env_vars
  }
}

resource "aws_cloudwatch_log_group" "ws_lambda_logs" {
  name              = "/aws/lambda/${local.workspace_prefix}SaturdaysSpinoutWS"
  retention_in_days = 7
}

resource "aws_lambda_permission" "ws_api_gateway" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ws_lambda.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:us-east-1:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.websockets.id}/*/*"
}

output "ws_env_vars" {
  value = join(" ", [for k, v in local.ws_env_vars : "${k}=${v}"])
}
