variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "state_bucket" {
  type        = string
  description = "The name of the state bucket, used to grant GitHub Actions access to terraform state"
}

variable "route53_domain" {
  type        = string
  description = "The name of the domain, registered in route53, that will be used to deploy the application"
  default     = "saturdaysspinout.com"
}

variable "workspaces_ownership_txt_value" {
  type        = string
  description = "The value for domain ownership verification for google workspaces"
  default     = "google-site-verification=SR6l5mVpX3Slij5lAcNQ-rdKdTtoj_esr_T2z83mFDM"
}