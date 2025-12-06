locals {
  is_primary_deployment = terraform.workspace == "default"

  workspace_prefix = terraform.workspace == "default" ? "" : "${terraform.workspace}-"
}