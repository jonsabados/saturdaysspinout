terraform {
  backend "s3" {
    region = "us-east-1"
    bucket = "8432-6435-3275-saturdaysspinout-tfstate"
    key    = "saturdaysspinout"
  }
}