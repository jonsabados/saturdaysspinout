# AWS Account Prep

One-time account-level AWS configuration that must be applied before deploying the main infrastructure.

## What's in here?

| File | Purpose |
|------|---------|
| `api-gateway.tf` | IAM role and account settings allowing API Gateway to write CloudWatch logs |
| `github-actions.tf` | GitHub Actions OIDC provider and IAM role for CI/CD deployments |
| `gmail.tf` | Google Workspace DNS records (MX, domain verification) |

These resources are account-level singletons - they only need to exist once per AWS account, regardless of how many Terraform workspaces you use for the main infrastructure.

## Usage

```bash
cd aws_account_prep

# Initialize with your state bucket
terraform init -backend-config="bucket=your-state-bucket-name"

# Apply (you'll be prompted for state_bucket variable)
terraform apply -var="state_bucket=your-state-bucket-name"
```

Or create an `aws_account_prep/terraform.tfvars`:

```hcl
state_bucket = "your-state-bucket-name"
```

Then just:

```bash
terraform init -backend-config="bucket=your-state-bucket-name"
terraform apply
```

## When to run this

- Once when setting up a new AWS account
- When adding new account-level resources
- After changes to GitHub Actions IAM permissions

This is a manual process - it's not part of the CI/CD pipeline since it manages the infrastructure that CI/CD depends on.