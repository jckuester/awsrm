# awsrm

A remove command for AWS resources

[![Release](https://img.shields.io/github/release/jckuester/awsrm.svg?style=for-the-badge)](https://github.com/jckuester/awsrm/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Travis](https://img.shields.io/travis/com/jckuester/awsrm/master.svg?style=for-the-badge)](https://travis-ci.com/jckuester/awsrm)

This command line tool follows the [Unix Philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well)
and `does only one thing, but (hopefully) does it well`:

Simplify deleting over [250 supported AWS resource types](https://github.com/jckuester/awsls#supported-resources)
across multiple accounts and regions.

Like other Unix-like tools, `awsrm` unveils its real power when combining it via pipes with the output of other tools,
such as [`awsls`](https://github.com/jckuester/awsls) for listing AWS resources and `grep` for filtering them.

## Example


### Delete across multiple accounts and regions

    awsls -p myaccount1,myaccount2 -r us-west-2,us-east-1 instance | awsrm

// Add gif

### Filter resources by tags (or other attributes)

1. List resources via [`awsls`](https://github.com/jckuester/awsls) with the attributes you want to filter on
   (here: `-a tags`)
2. Use standard tools like `grep` to filter resources
3. Pipe result into `awsrm` (nothing is deleted until you confirm)

The following example deletes all AWS instances with tag `Name=foo` in the AWS accounts associated with
 profile `myaccount1` and `myaccount2` in both regions `us-west-2` and `us-east-1`:

    awsls instance -a tags | grep Name=foo | awsrm

![](img/pipe-iam-role.gif)

## Usage

Input via arguments:

	awsrm [flags] <resource_type> <id> [<id>...]

Input via pipe:

    awsls [flags] <resource_type> | awsrm

To see options available run `awsrm --help`.

## Installation

### Binary Releases

You can download a specific version on the [releases page](https://github.com/jckuester/awsrm/releases) or
use the following way to install to `./bin/`:

```bash
curl -sSfL https://raw.githubusercontent.com/jckuester/awrm/master/install.sh | sh -s v0.1.0
```

### Homebrew

Homebrew users can install by:

```bash
brew install jckuester/tap/awls
```

For more information on Homebrew taps please see the [tap documentation](https://docs.brew.sh/Taps).

## Disclaimer

You are using this tool at your own risk! I will not take responsibility if you delete any critical resources in your
production environments.
