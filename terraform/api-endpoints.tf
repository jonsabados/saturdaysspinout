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

# /ingestion
resource "aws_api_gateway_resource" "ingestion" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "ingestion"
}

# /ingestion/race
resource "aws_api_gateway_resource" "ingestion_race" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.ingestion.id
  path_part   = "race"
}

# /developer
resource "aws_api_gateway_resource" "developer" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "developer"
}

# /developer/iracing-api
resource "aws_api_gateway_resource" "developer_iracing_api" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.developer.id
  path_part   = "iracing-api"
}

# /developer/iracing-api/{proxy+}
resource "aws_api_gateway_resource" "developer_iracing_api_proxy" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.developer_iracing_api.id
  path_part   = "{proxy+}"
}

# /developer/iracing-token
resource "aws_api_gateway_resource" "developer_iracing_token" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.developer.id
  path_part   = "iracing-token"
}

# /driver
resource "aws_api_gateway_resource" "driver" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "driver"
}

# /driver/{driver_id}
resource "aws_api_gateway_resource" "driver_id" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.driver.id
  path_part   = "{driver_id}"
}

# /driver/{driver_id}/races
resource "aws_api_gateway_resource" "driver_races" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.driver_id.id
  path_part   = "races"
}

# /driver/{driver_id}/races/{driver_race_id}
resource "aws_api_gateway_resource" "driver_race" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.driver_races.id
  path_part   = "{driver_race_id}"
}

# /tracks
resource "aws_api_gateway_resource" "tracks" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "tracks"
}

# /cars
resource "aws_api_gateway_resource" "cars" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "cars"
}

# /session
resource "aws_api_gateway_resource" "session" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "session"
}

# /session/{subsession_id}
resource "aws_api_gateway_resource" "session_id" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.session.id
  path_part   = "{subsession_id}"
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

module "ingestion_race_post" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.ingestion_race.id
  http_method       = "POST"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "ingestion_race_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.ingestion_race.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_api_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_api.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_api_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_api.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_api_proxy_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_api_proxy.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_api_proxy_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_api_proxy.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_token_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_token.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "developer_iracing_token_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.developer_iracing_token.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_id.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_id.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_races_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_races.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_races_delete" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_races.id
  http_method       = "DELETE"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_races_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_races.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_race_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_race.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "driver_race_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.driver_race.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "tracks_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.tracks.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "tracks_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.tracks.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "cars_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.cars.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "cars_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.cars.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "session_get" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.session_id.id
  http_method       = "GET"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}

module "session_options" {
  source            = "./api_endpoint"
  rest_api_id       = aws_api_gateway_rest_api.api.id
  resource_id       = aws_api_gateway_resource.session_id.id
  http_method       = "OPTIONS"
  lambda_invoke_arn = aws_lambda_function.api_lambda.invoke_arn
}