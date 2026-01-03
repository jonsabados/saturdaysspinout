variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "route53_domain" {
  type        = string
  description = "The name of the domain, registered in route53, that will be used to deploy the application"
  default     = "saturdaysspinout.com"
}

variable "race_ingestion_processor_concurrency" {
  description = "Reserved concurrent executions for the race ingestion processor Lambda"
  type        = number
  default     = 15
}