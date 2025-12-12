# API Gateway Resources - Path Structure
# =======================================

# /health
resource "aws_api_gateway_resource" "health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "health"
}

# /health/ping
resource "aws_api_gateway_resource" "health_ping" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.health.id
  path_part   = "ping"
}

# /auth
resource "aws_api_gateway_resource" "auth" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "auth"
}

# /auth/ir
resource "aws_api_gateway_resource" "auth_ir" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.auth.id
  path_part   = "ir"
}

# /auth/ir/callback
resource "aws_api_gateway_resource" "auth_ir_callback" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.auth_ir.id
  path_part   = "callback"
}

# /auth/refresh
resource "aws_api_gateway_resource" "auth_refresh" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.auth.id
  path_part   = "refresh"
}

# /doc
resource "aws_api_gateway_resource" "doc" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "doc"
}

# /doc/iracing-api
resource "aws_api_gateway_resource" "doc_iracing_api" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.doc.id
  path_part   = "iracing-api"
}

# /doc/iracing-api/{proxy+}
resource "aws_api_gateway_resource" "doc_iracing_api_proxy" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.doc_iracing_api.id
  path_part   = "{proxy+}"
}

# API Gateway Endpoints
# =====================

module "health_ping_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.health_ping.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "health_ping_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.health_ping.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "auth_ir_callback_post" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.auth_ir_callback.id
  http_method       = "POST"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "auth_ir_callback_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.auth_ir_callback.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "auth_refresh_post" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.auth_refresh.id
  http_method       = "POST"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "auth_refresh_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.auth_refresh.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "doc_iracing_api_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.doc_iracing_api.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "doc_iracing_api_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.doc_iracing_api.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "doc_iracing_api_proxy_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.doc_iracing_api_proxy.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "doc_iracing_api_proxy_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.doc_iracing_api_proxy.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}