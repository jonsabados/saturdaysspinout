resource "aws_cloudwatch_dashboard" "system_health" {
  dashboard_name = "${local.workspace_prefix}SaturdaysSpinoutSystemHealth"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6
        properties = {
          title  = "iRacing API Rate Limit Remaining"
          region = "us-east-1"
          metrics = [
            ["${local.workspace_prefix}SaturdaysSpinout", "iracing_ratelimit_remaining", { stat = "Minimum", period = 60 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 6
        height = 6
        properties = {
          title  = "Driver Sessions Ingested"
          region = "us-east-1"
          metrics = [
            ["${local.workspace_prefix}SaturdaysSpinout", "driver_sessions_ingested", { stat = "Sum", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 18
        y      = 0
        width  = 6
        height = 6
        properties = {
          title  = "Journal Entries Saved"
          region = "us-east-1"
          metrics = [
            ["${local.workspace_prefix}SaturdaysSpinout", "journal_entries_created", { stat = "Sum", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6
        properties = {
          title  = "Website Traffic (Requests)"
          region = "us-east-1"
          metrics = [
            ["AWS/CloudFront", "Requests", "DistributionId", aws_cloudfront_distribution.website.id, "Region", "Global", { stat = "Sum", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 6
        width  = 12
        height = 6
        properties = {
          title  = "App Traffic (Requests)"
          region = "us-east-1"
          metrics = [
            ["AWS/CloudFront", "Requests", "DistributionId", aws_cloudfront_distribution.frontend.id, "Region", "Global", { stat = "Sum", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 12
        width  = 12
        height = 6
        properties = {
          title  = "Website Error Rate (%)"
          region = "us-east-1"
          metrics = [
            ["AWS/CloudFront", "TotalErrorRate", "DistributionId", aws_cloudfront_distribution.website.id, "Region", "Global", { stat = "Average", period = 300 }],
            ["AWS/CloudFront", "4xxErrorRate", "DistributionId", aws_cloudfront_distribution.website.id, "Region", "Global", { stat = "Average", period = 300 }],
            ["AWS/CloudFront", "5xxErrorRate", "DistributionId", aws_cloudfront_distribution.website.id, "Region", "Global", { stat = "Average", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
              max = 100
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 12
        width  = 12
        height = 6
        properties = {
          title  = "App Error Rate (%)"
          region = "us-east-1"
          metrics = [
            ["AWS/CloudFront", "TotalErrorRate", "DistributionId", aws_cloudfront_distribution.frontend.id, "Region", "Global", { stat = "Average", period = 300 }],
            ["AWS/CloudFront", "4xxErrorRate", "DistributionId", aws_cloudfront_distribution.frontend.id, "Region", "Global", { stat = "Average", period = 300 }],
            ["AWS/CloudFront", "5xxErrorRate", "DistributionId", aws_cloudfront_distribution.frontend.id, "Region", "Global", { stat = "Average", period = 300 }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
              max = 100
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 18
        width  = 12
        height = 6
        properties = {
          title  = "REST API Errors"
          region = "us-east-1"
          metrics = [
            ["AWS/ApiGateway", "4XXError", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.api.stage_name, { stat = "Sum", period = 300, label = "4XX Errors" }],
            ["AWS/ApiGateway", "5XXError", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.api.stage_name, { stat = "Sum", period = 300, label = "5XX Errors" }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 18
        width  = 12
        height = 6
        properties = {
          title  = "REST API Latency (ms)"
          region = "us-east-1"
          metrics = [
            ["AWS/ApiGateway", "Latency", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.api.stage_name, { stat = "p50", period = 300, label = "p50" }],
            ["AWS/ApiGateway", "Latency", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.api.stage_name, { stat = "p90", period = 300, label = "p90" }],
            ["AWS/ApiGateway", "Latency", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.api.stage_name, { stat = "p99", period = 300, label = "p99" }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 24
        width  = 12
        height = 6
        properties = {
          title  = "WebSocket API Errors"
          region = "us-east-1"
          metrics = [
            ["AWS/ApiGateway", "ClientError", "ApiId", aws_apigatewayv2_api.websockets.id, "Stage", aws_apigatewayv2_stage.ws.name, { stat = "Sum", period = 300, label = "Client Errors" }],
            ["AWS/ApiGateway", "ExecutionError", "ApiId", aws_apigatewayv2_api.websockets.id, "Stage", aws_apigatewayv2_stage.ws.name, { stat = "Sum", period = 300, label = "Execution Errors" }],
            ["AWS/ApiGateway", "IntegrationError", "ApiId", aws_apigatewayv2_api.websockets.id, "Stage", aws_apigatewayv2_stage.ws.name, { stat = "Sum", period = 300, label = "Integration Errors" }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 24
        width  = 12
        height = 6
        properties = {
          title  = "WebSocket Connections"
          region = "us-east-1"
          metrics = [
            ["AWS/ApiGateway", "ConnectCount", "ApiId", aws_apigatewayv2_api.websockets.id, "Stage", aws_apigatewayv2_stage.ws.name, { stat = "Sum", period = 300, label = "New Connections" }],
            ["AWS/ApiGateway", "MessageCount", "ApiId", aws_apigatewayv2_api.websockets.id, "Stage", aws_apigatewayv2_stage.ws.name, { stat = "Sum", period = 300, label = "Messages" }]
          ]
          view    = "timeSeries"
          stacked = false
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      }
    ]
  })
}

output "system_health_dashboard_url" {
  value = "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#dashboards:name=${aws_cloudwatch_dashboard.system_health.dashboard_name}"
}
