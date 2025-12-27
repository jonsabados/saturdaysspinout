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
          title  = "Sessions Ingested"
          region = "us-east-1"
          metrics = [
            ["${local.workspace_prefix}SaturdaysSpinout", "sessions_ingested", { stat = "Sum", period = 300 }]
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
          title  = "Laps Ingested"
          region = "us-east-1"
          metrics = [
            ["${local.workspace_prefix}SaturdaysSpinout", "laps_ingested", { stat = "Sum", period = 300 }]
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
