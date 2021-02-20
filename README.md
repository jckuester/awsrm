# awsrm

A remove command for AWS resources

[![Release](https://img.shields.io/github/release/jckuester/awsrm.svg?style=for-the-badge)](https://github.com/jckuester/awsrm/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Travis](https://img.shields.io/travis/com/jckuester/awsrm/master.svg?style=for-the-badge)](https://travis-ci.com/jckuester/awsrm)

This command line tool follows
the [Unix Philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well)
of `doing only one thing and doing it well`:

It simplifies deleting over [250 supported AWS resource types](#supported-resources)
across multiple accounts and regions.

Like other Unix-like tools, `awsrm` reveals its full power when combining it (via pipes) with other tools, such
as [`awsls`](https://github.com/jckuester/awsls) for listing AWS resources and `grep` for filtering by resource
attributes.

## Example

### Delete resources by tags (or other attributes)

To delete, for example, all EC2 instances with tag `Name=foo`, run

      awsls instance -a tags | grep Name=foo | awsrm

To filter on multiple attributes, display them with `awsls` via the `-a (--attributes) <comma-separated list>` flag.
Every attribute in the Terraform documentation
([here](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/instance#attributes-reference) are
the attributes for `aws_instance`) can be used:

![](https://raw.githubusercontent.com/jckuester/awsrm/master/.github/img/awsrm-grep.gif)

1. List resources via [`awsls`](https://github.com/jckuester/awsls) with the attributes you want to filter on
   (here: `-a tags`)
2. Use standard tools like `grep` to filter resources
3. Pipe result to `awsrm` (nothing is deleted until you confirm)

**Note**: `awsls` output passes on profile and region information, so that `awsrm` knows for each resource in what
account and region to delete it.

Depending on the type of resource, deletion can take some time. For your convenience, this GIF runs faster than EC2
instances are actually terminated; the shell prompt shows the real execution times in seconds.

### Delete across multiple accounts and regions

List all instances in the AWS accounts associated with profile `myaccount1` and `myaccount2` in both regions `us-west-2`
and `us-east-1` and pipe the result to `awsrm`:

    awsls -p myaccount1,myaccount2 -r us-west-2,us-east-1 instance | awsrm

![](https://raw.githubusercontent.com/jckuester/awsrm/master/.github/img/awsrm-multi-profile-region.gif)

### Delete by IDs

Delete specific resources by ID, for example, some VPCs and IAM roles:

    awsrm iam_role db-cluster elb nginx
    awsrm vpc vpc-1234 vpc-3456 vpc-7689

![](https://raw.githubusercontent.com/jckuester/awsrm/master/.github/img/awsrm-args.gif)

1. List resources via [`awsls`](https://github.com/jckuester/awsls) to find out what resources to delete.
2. Use `awsrm` to delete the resources by resource type and ID(s)

## Usage

Input via arguments:

	awsrm [flags] <resource_type> <id> [<id>...]

Input via pipe:

    awsls [flags] <resource_type> | awsrm
    echo "<resource_type> <id> <profile> <region>" | awsrm

To see options available run `awsrm --help`.

## Installation

### Binary Releases

You can download a specific version on the [releases page](https://github.com/jckuester/awsrm/releases) or use the
following way to install to `./bin/`:

```bash
curl -sSfL https://raw.githubusercontent.com/jckuester/awrm/master/install.sh | sh -s v0.1.0
```

### Homebrew

Homebrew users can install by:

```bash
brew install jckuester/tap/awrm
```

For more information on Homebrew taps please see the [tap documentation](https://docs.brew.sh/Taps).

## Supported resources

This tool can not only delete any resource type that
is [supported by awsls](https://github.com/jckuester/awsls#supported-resources), but any resource
type [covered by the Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs).

Note: the prefix `aws_` for resource types is optional. This means, for example, `awsrm aws_instance <id>` and
`awsrm instance <id>` are both valid commands.

## Disclaimer

You are using this tool at your own risk! I will not take responsibility if you delete any critical resources in your
production environments.
