locals {
  website_base_domain     = trimsuffix(data.aws_route53_zone.route53_zone.name, ".")
  website_domain_name     = terraform.workspace == "default" ? local.website_base_domain : "${terraform.workspace}.${local.website_base_domain}"
  website_www_domain_name = "${local.workspace_prefix}www.${local.website_base_domain}"
}

resource "aws_s3_bucket" "website" {
  bucket = "${local.workspace_prefix}saturdaysspinout-website-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_public_access_block" "website" {
  bucket = aws_s3_bucket.website.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

module "website_cert" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~> 4.0"

  domain_name               = local.website_domain_name
  subject_alternative_names = [local.website_www_domain_name]
  zone_id                   = data.aws_route53_zone.route53_zone.id

  validation_method   = "DNS"
  wait_for_validation = true
}

resource "aws_cloudfront_origin_access_control" "website" {
  name                              = "${local.workspace_prefix}saturdaysspinout-website"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_function" "www_redirect" {
  name    = "${local.workspace_prefix}www-redirect"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-EOF
    function handler(event) {
      var request = event.request;
      var host = request.headers.host.value;
      var uri = request.uri;
      var targetHost = '${local.website_domain_name}';

      // Redirect www to non-www
      if (host !== targetHost) {
        return {
          statusCode: 301,
          statusDescription: 'Moved Permanently',
          headers: {
            location: { value: 'https://' + targetHost + uri }
          }
        };
      }

      // Rewrite directory requests to index.html
      // e.g., /es/ -> /es/index.html
      if (uri.endsWith('/')) {
        request.uri = uri + 'index.html';
      } else if (!uri.includes('.')) {
        // Handle /es -> /es/ redirect for consistency
        return {
          statusCode: 301,
          statusDescription: 'Moved Permanently',
          headers: {
            location: { value: 'https://' + targetHost + uri + '/' }
          }
        };
      }

      return request;
    }
  EOF
}

resource "aws_cloudfront_distribution" "website" {
  enabled             = true
  default_root_object = "index.html"
  aliases             = [local.website_domain_name, local.website_www_domain_name]

  origin {
    domain_name              = aws_s3_bucket.website.bucket_regional_domain_name
    origin_id                = "s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.website.id
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "s3"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = data.aws_cloudfront_cache_policy.caching_optimized.id

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.www_redirect.arn
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = module.website_cert.acm_certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}

data "aws_iam_policy_document" "website_bucket_policy" {
  statement {
    sid       = "AllowCloudFrontAccess"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.website.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.website.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "website" {
  bucket = aws_s3_bucket.website.id
  policy = data.aws_iam_policy_document.website_bucket_policy.json
}

resource "aws_route53_record" "website" {
  zone_id = data.aws_route53_zone.route53_zone.id
  name    = local.website_domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.website.domain_name
    zone_id                = aws_cloudfront_distribution.website.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "website_www" {
  zone_id = data.aws_route53_zone.route53_zone.id
  name    = local.website_www_domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.website.domain_name
    zone_id                = aws_cloudfront_distribution.website.hosted_zone_id
    evaluate_target_health = false
  }
}

output "website_bucket_name" {
  value = aws_s3_bucket.website.bucket
}

output "website_url" {
  value = "https://${local.website_domain_name}"
}