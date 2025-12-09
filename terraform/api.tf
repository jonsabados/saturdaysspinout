locals {
  api_host_name   = "${local.workspace_prefix}api"
  api_domain_name = "${local.api_host_name}.${data.aws_route53_zone.route53_zone.name}"

  app_env_vars = {
    LOG_LEVEL                  = "info"
    CORS_ALLOWED_ORIGINS       = "https://${local.frontend_domain_name},http://127.0.0.1:5173"
    IRACING_CREDENTIALS_SECRET = data.aws_secretsmanager_secret.iracing_credentials.arn
    JWT_SIGNING_KEY_ARN        = aws_kms_key.jwt.arn
    JWT_ENCRYPTION_KEY_ARN     = aws_kms_key.jwt_encryption.arn
  }
}

resource "aws_iam_role" "api_lambda" {
  name               = "${local.workspace_prefix}SaturdaysSpinoutAPI"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role_policy.json
}

data "aws_iam_policy_document" "api_lambda" {
  statement {
    sid    = "AllowLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "${aws_cloudwatch_log_group.api_lambda_logs.arn}:*"
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
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:TransactWriteItems",
      "dynamodb:TransactGetItems"
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
      data.aws_secretsmanager_secret.iracing_credentials.arn
    ]
  }

  statement {
    sid    = "AllowKMSJWTSigning"
    effect = "Allow"
    actions = [
      "kms:Sign",
      "kms:Verify",
      "kms:GetPublicKey"
    ]
    resources = [
      aws_kms_key.jwt.arn
    ]
  }

  statement {
    sid    = "AllowKMSJWTEncryption"
    effect = "Allow"
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:GenerateDataKey"
    ]
    resources = [
      aws_kms_key.jwt_encryption.arn
    ]
  }
}

resource "aws_iam_role_policy" "api_lambda" {
  role   = aws_iam_role.api_lambda.name
  policy = data.aws_iam_policy_document.api_lambda.json
}

resource "aws_lambda_function" "api_lambda" {
  filename         = "../dist/apiLambda.zip"
  source_code_hash = filebase64sha256("../dist/apiLambda.zip")
  timeout          = 15
  // setting reserved concurrent executions super low cause personal account & don't want to make it too easy for someone to grief my wallet by pounding on things
  reserved_concurrent_executions = 15
  runtime                        = "provided.al2"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  function_name                  = "${local.workspace_prefix}SaturdaysSpinoutAPI"
  role                           = aws_iam_role.api_lambda.arn

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = local.app_env_vars
  }
}

resource "aws_cloudwatch_log_group" "api_lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.api_lambda.function_name}"
  retention_in_days = 7
}

data "aws_route53_zone" "route53_zone" {
  name = var.route53_domain
}

module "api_cert" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~> 4.0"

  domain_name = local.api_domain_name
  zone_id     = data.aws_route53_zone.route53_zone.id

  validation_method = "DNS"

  wait_for_validation = true
}

resource "aws_api_gateway_rest_api" "api" {
  name = "${local.workspace_prefix}SaturdaysSpinoutAPI"

  tags = {
    Workspace = terraform.workspace
  }
}

resource "aws_api_gateway_domain_name" "api" {
  domain_name     = local.api_domain_name
  certificate_arn = module.api_cert.acm_certificate_arn
}

resource "aws_route53_record" "api" {
  name    = local.api_host_name
  type    = "CNAME"
  zone_id = data.aws_route53_zone.route53_zone.id
  records = [aws_api_gateway_domain_name.api.cloudfront_domain_name]
  ttl     = 300
}

resource "aws_api_gateway_resource" "health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "health"
}

resource "aws_api_gateway_resource" "health_ping" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.health.id
  path_part   = "ping"
}

resource "aws_api_gateway_method" "health_ping_get" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.health_ping.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_method" "health_ping_options" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.health_ping.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_lambda_permission" "api_gateway_api_lambda" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api_lambda.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "arn:aws:execute-api:us-east-1:${data.aws_caller_identity.current.account_id}:${aws_api_gateway_rest_api.api.id}/*/*"
}

resource "aws_api_gateway_integration" "health_ping_get" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.health_ping.id
  http_method             = aws_api_gateway_method.health_ping_get.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api_lambda.invoke_arn
}

resource "aws_api_gateway_integration" "health_ping_options" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.health_ping.id
  http_method             = aws_api_gateway_method.health_ping_options.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api_lambda.invoke_arn
}

resource "aws_api_gateway_resource" "auth" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "auth"
}

resource "aws_api_gateway_resource" "auth_ir" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.auth.id
  path_part   = "ir"
}

resource "aws_api_gateway_resource" "auth_ir_callback" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.auth_ir.id
  path_part   = "callback"
}

resource "aws_api_gateway_method" "auth_ir_callback_post" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.auth_ir_callback.id
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_method" "auth_ir_callback_options" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.auth_ir_callback.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "auth_ir_callback_post" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.auth_ir_callback.id
  http_method             = aws_api_gateway_method.auth_ir_callback_post.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api_lambda.invoke_arn
}

resource "aws_api_gateway_integration" "auth_ir_callback_options" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.auth_ir_callback.id
  http_method             = aws_api_gateway_method.auth_ir_callback_options.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api_lambda.invoke_arn
}

resource "aws_api_gateway_deployment" "api" {
  depends_on = [
    aws_api_gateway_integration.health_ping_get,
    aws_api_gateway_integration.health_ping_options,
    aws_api_gateway_integration.auth_ir_callback_post,
    aws_api_gateway_integration.auth_ir_callback_options,
  ]
  rest_api_id = aws_api_gateway_rest_api.api.id

  variables = {
    "deployed_at" : timestamp()
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "api" {
  deployment_id = aws_api_gateway_deployment.api.id
  rest_api_id   = aws_api_gateway_rest_api.api.id
  stage_name    = "${local.workspace_prefix}saturdaysspinout-main"
}

resource "aws_api_gateway_base_path_mapping" "test" {
  api_id      = aws_api_gateway_rest_api.api.id
  stage_name  = aws_api_gateway_stage.api.stage_name
  domain_name = aws_api_gateway_domain_name.api.domain_name
}

output "api_url" {
  value = "https://${local.api_domain_name}"
}

output "app_env_vars" {
  value = join(" ", [for k, v in local.app_env_vars : "${k}=${v}"])
}