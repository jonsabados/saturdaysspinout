# txt record to verify domain ownership
resource "aws_route53_record" "google_workspace_verification" {
  zone_id = data.aws_route53_zone.route53_zone.id
  name    = ""
  type    = "TXT"
  ttl     = 300
  records = [var.workspaces_ownership_txt_value]
}

resource "aws_route53_record" "google_workspace_mx" {
  zone_id = data.aws_route53_zone.route53_zone.id
  name    = var.route53_domain
  type    = "MX"
  ttl     = 300

  records = [
    "1 SMTP.GOOGLE.COM",
  ]
}