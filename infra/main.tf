
provider "aws" {
  region = "us-east-1"
}

locals {
  name           = "TF Provider"
  domain_name    = "cjscloud.city"
  subdomain_name = "tfp.${local.domain_name}"
  environment    = "default"
}

# S3 bucket
resource "aws_s3_bucket" "provider" {
  bucket = local.subdomain_name

  tags = {
    Name        = local.name
    Environment = local.environment
  }
}

resource "aws_s3_bucket_public_access_block" "provider" {
  bucket = aws_s3_bucket.provider.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "provider" {
  bucket = aws_s3_bucket.provider.id
  policy = data.aws_iam_policy_document.provider.json
}

data "aws_iam_policy_document" "provider" {
  statement {
    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "s3:GetObject",
    ]

    resources = [
      "${aws_s3_bucket.provider.arn}/*",
    ]
  }
}

# ACM Certificate
resource "aws_acm_certificate" "provider" {
  domain_name       = local.subdomain_name
  validation_method = "DNS"

  tags = {
    Name        = local.name
    Environment = local.environment
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Route53 Records
data "aws_route53_zone" "provider" {
  name         = local.domain_name
  private_zone = false
}

resource "aws_route53_record" "provider" {
  for_each = {
    for dvo in aws_acm_certificate.provider.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = data.aws_route53_zone.provider.zone_id
}

resource "aws_route53_record" "www" {
  zone_id = data.aws_route53_zone.provider.zone_id
  name    = local.subdomain_name
  type    = "CNAME"
  ttl     = 300
  records = [aws_cloudfront_distribution.provider.domain_name]
}

# Cert DNS validation, depended on by CloudFront
resource "aws_acm_certificate_validation" "provider" {
  certificate_arn         = aws_acm_certificate.provider.arn
  validation_record_fqdns = [for record in aws_route53_record.provider : record.fqdn]
}

# Cloudfront
resource "aws_cloudfront_distribution" "provider" {
  origin {
    domain_name = aws_s3_bucket.provider.bucket_regional_domain_name
    origin_id   = aws_s3_bucket.provider.bucket_domain_name
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "Terraform Provider Registry"
  default_root_object = "index.html"

  aliases = [local.subdomain_name]

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = aws_s3_bucket.provider.bucket_domain_name

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "allow-all"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  price_class = "PriceClass_100"

  restrictions {
    geo_restriction {
      # restriction_type = "whitelist"
      # locations        = ["NZ"]
      restriction_type = "none"
    }
  }

  tags = {
    Name        = local.name
    Environment = local.environment
  }

  viewer_certificate {
    cloudfront_default_certificate = false
    acm_certificate_arn            = aws_acm_certificate.provider.arn
    ssl_support_method             = "sni-only"
  }
}
