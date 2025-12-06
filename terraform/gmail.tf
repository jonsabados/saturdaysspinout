# txt record to verify domain ownership
resource "aws_route53_record" "google_workspace_verification" {
  count = local.is_primary_deployment ? 1 : 0

  zone_id = data.aws_route53_zone.route53_zone.id
  name    = ""
  type    = "TXT"
  ttl     = 300
  records = [var.workspaces_ownership_txt_value]
}

resource "aws_route53_record" "google_workspace_mx" {
  count = local.is_primary_deployment ? 1 : 0

  zone_id = data.aws_route53_zone.route53_zone.id
  name    = var.route53_domain
  type    = "MX"
  ttl     = 300

  records = [
    "1 SMTP.GOOGLE.COM",
  ]
}