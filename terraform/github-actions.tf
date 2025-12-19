# GitHub Actions OIDC provider and IAM role for CI/CD deployments
# Only created in the primary (default) workspace

resource "aws_iam_openid_connect_provider" "github" {
  count = local.is_primary_deployment ? 1 : 0

  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["ffffffffffffffffffffffffffffffffffffffff"] # GitHub's OIDC doesn't require thumbprint validation
}

data "aws_iam_policy_document" "github_actions_assume_role" {
  count = local.is_primary_deployment ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github[0].arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:jonsabados/saturdaysspinout:ref:refs/tags/*"]
    }
  }
}

resource "aws_iam_role" "github_actions" {
  count = local.is_primary_deployment ? 1 : 0

  name               = "github-actions-deploy"
  assume_role_policy = data.aws_iam_policy_document.github_actions_assume_role[0].json
}

data "aws_iam_policy_document" "github_actions_permissions" {
  count = local.is_primary_deployment ? 1 : 0

  # Terraform state access
  statement {
    sid    = "TerraformStateAccess"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket",
    ]
    resources = [
      "arn:aws:s3:::8432-6435-3275-saturdaysspinout-tfstate",
      "arn:aws:s3:::8432-6435-3275-saturdaysspinout-tfstate/*",
    ]
  }

  # Lambda management
  statement {
    sid    = "LambdaManagement"
    effect = "Allow"
    actions = [
      "lambda:*",
    ]
    resources = ["*"]
  }

  # API Gateway management
  statement {
    sid    = "APIGatewayManagement"
    effect = "Allow"
    actions = [
      "apigateway:*",
    ]
    resources = ["*"]
  }

  # DynamoDB management
  statement {
    sid    = "DynamoDBManagement"
    effect = "Allow"
    actions = [
      "dynamodb:*",
    ]
    resources = ["*"]
  }

  # S3 management (for frontend buckets)
  statement {
    sid    = "S3Management"
    effect = "Allow"
    actions = [
      "s3:*",
    ]
    resources = ["*"]
  }

  # CloudFront management
  statement {
    sid    = "CloudFrontManagement"
    effect = "Allow"
    actions = [
      "cloudfront:*",
    ]
    resources = ["*"]
  }

  # Route53 management
  statement {
    sid    = "Route53Management"
    effect = "Allow"
    actions = [
      "route53:*",
    ]
    resources = ["*"]
  }

  # ACM certificate management
  statement {
    sid    = "ACMManagement"
    effect = "Allow"
    actions = [
      "acm:*",
    ]
    resources = ["*"]
  }

  # KMS management
  statement {
    sid    = "KMSManagement"
    effect = "Allow"
    actions = [
      "kms:*",
    ]
    resources = ["*"]
  }

  # Secrets Manager management
  statement {
    sid    = "SecretsManagerManagement"
    effect = "Allow"
    actions = [
      "secretsmanager:*",
    ]
    resources = ["*"]
  }

  # SQS management
  statement {
    sid    = "SQSManagement"
    effect = "Allow"
    actions = [
      "sqs:*",
    ]
    resources = ["*"]
  }

  # IAM management (for Lambda execution roles)
  statement {
    sid    = "IAMManagement"
    effect = "Allow"
    actions = [
      "iam:*",
    ]
    resources = ["*"]
  }

  # CloudWatch Logs management
  statement {
    sid    = "CloudWatchLogsManagement"
    effect = "Allow"
    actions = [
      "logs:*",
    ]
    resources = ["*"]
  }

  # X-Ray for tracing
  statement {
    sid    = "XRayManagement"
    effect = "Allow"
    actions = [
      "xray:*",
    ]
    resources = ["*"]
  }

  # CloudWatch for dashboards/alarms
  statement {
    sid    = "CloudWatchManagement"
    effect = "Allow"
    actions = [
      "cloudwatch:*",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "github_actions" {
  count = local.is_primary_deployment ? 1 : 0

  name   = "github-actions-deploy-policy"
  role   = aws_iam_role.github_actions[0].id
  policy = data.aws_iam_policy_document.github_actions_permissions[0].json
}

output "github_actions_role_arn" {
  value       = local.is_primary_deployment ? aws_iam_role.github_actions[0].arn : null
  description = "ARN of the IAM role for GitHub Actions to assume"
}