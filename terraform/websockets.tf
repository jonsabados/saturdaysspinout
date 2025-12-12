locals {
  ws_host_name   = "${local.workspace_prefix}ws"
  ws_domain_name = "${local.ws_host_name}.${data.aws_route53_zone.route53_zone.name}"
}

resource "aws_apigatewayv2_api" "websockets" {
  name                       = "${local.workspace_prefix}SaturdaysSpinoutWS"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"

  tags = {
    Workspace = terraform.workspace
  }
}

resource "aws_apigatewayv2_integration" "ws_lambda" {
  api_id             = aws_apigatewayv2_api.websockets.id
  integration_type   = "AWS_PROXY"
  integration_uri    = aws_lambda_function.ws_lambda.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_route" "ws_connect" {
  api_id    = aws_apigatewayv2_api.websockets.id
  route_key = "$connect"
  target    = "integrations/${aws_apigatewayv2_integration.ws_lambda.id}"
}

resource "aws_apigatewayv2_route" "ws_disconnect" {
  api_id    = aws_apigatewayv2_api.websockets.id
  route_key = "$disconnect"
  target    = "integrations/${aws_apigatewayv2_integration.ws_lambda.id}"
}

resource "aws_apigatewayv2_route" "ws_default" {
  api_id    = aws_apigatewayv2_api.websockets.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.ws_lambda.id}"
}

resource "aws_apigatewayv2_route" "ws_auth" {
  api_id    = aws_apigatewayv2_api.websockets.id
  route_key = "auth"
  target    = "integrations/${aws_apigatewayv2_integration.ws_lambda.id}"
}

resource "aws_apigatewayv2_route" "ws_ping" {
  api_id    = aws_apigatewayv2_api.websockets.id
  route_key = "pingRequest"
  target    = "integrations/${aws_apigatewayv2_integration.ws_lambda.id}"
}

resource "aws_apigatewayv2_stage" "ws" {
  api_id      = aws_apigatewayv2_api.websockets.id
  name        = "${local.workspace_prefix}saturdaysspinout-ws"
  auto_deploy = true

  default_route_settings {
    throttling_burst_limit = 100
    throttling_rate_limit  = 100
  }

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.ws_api_logs.arn
    format = jsonencode({
      requestId         = "$context.requestId"
      ip                = "$context.identity.sourceIp"
      connectionId      = "$context.connectionId"
      routeKey          = "$context.routeKey"
      status            = "$context.status"
      errorMessage      = "$context.error.message"
      integrationError  = "$context.integration.error"
      integrationStatus = "$context.integration.status"
    })
  }
}

resource "aws_cloudwatch_log_group" "ws_api_logs" {
  name              = "/aws/apigateway/${local.workspace_prefix}SaturdaysSpinoutWS"
  retention_in_days = 7
}

module "ws_cert" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~> 4.0"

  domain_name = local.ws_domain_name
  zone_id     = data.aws_route53_zone.route53_zone.id

  validation_method   = "DNS"
  wait_for_validation = true
}

resource "aws_apigatewayv2_domain_name" "ws" {
  domain_name = local.ws_domain_name

  domain_name_configuration {
    certificate_arn = module.ws_cert.acm_certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

resource "aws_apigatewayv2_api_mapping" "ws" {
  api_id      = aws_apigatewayv2_api.websockets.id
  domain_name = aws_apigatewayv2_domain_name.ws.id
  stage       = aws_apigatewayv2_stage.ws.id
}

resource "aws_route53_record" "ws" {
  name    = local.ws_host_name
  type    = "A"
  zone_id = data.aws_route53_zone.route53_zone.id

  alias {
    name                   = aws_apigatewayv2_domain_name.ws.domain_name_configuration[0].target_domain_name
    zone_id                = aws_apigatewayv2_domain_name.ws.domain_name_configuration[0].hosted_zone_id
    evaluate_target_health = false
  }
}

output "ws_url" {
  value = "wss://${local.ws_domain_name}"
}

output "ws_management_endpoint" {
  value = "https://${aws_apigatewayv2_api.websockets.id}.execute-api.us-east-1.amazonaws.com/${aws_apigatewayv2_stage.ws.name}"
}
