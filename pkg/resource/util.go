package resource

import "strings"

// PrefixResourceType prefixes a resource type
// with "aws_" to be a valid Terraform resource type for the AWS provider.
func PrefixResourceType(rType string) string {
	if !strings.HasPrefix(rType, "aws_") {
		return "aws_" + rType
	}
	return rType
}
