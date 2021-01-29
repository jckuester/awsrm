variable "profile1" {
  description = "The named profile for 1st AWS account that will be deployed to"
  default = "myaccount1"
}

variable "profile2" {
  description = "The named profile for 2nd AWS account that will be deployed to"
  default = "myaccount2"

}

variable "region1" {
  description = "The 1st AWS region to deploy to"
  default = "us-west-2"
}

variable "region2" {
  description = "The 2nd AWS region to deploy to"
  default = "us-east-1"
}
