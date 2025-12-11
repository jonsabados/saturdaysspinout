variable "rest_api_id" {
  type        = string
  description = "The ID of the REST API"
}

variable "resource_id" {
  type        = string
  description = "The API Gateway resource ID to attach the method to"
}

variable "http_method" {
  type        = string
  description = "The HTTP method (GET, POST, PUT, DELETE, OPTIONS, etc.)"
}

variable "lambda_invoke_arn" {
  type        = string
  description = "The invoke ARN of the target Lambda function"
}