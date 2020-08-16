# awsrm

A remove command for AWS resources

[![Release](https://img.shields.io/github/release/jckuester/awsrm.svg?style=for-the-badge)](https://github.com/jckuester/awsrm/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Travis](https://img.shields.io/travis/jckuester/awsrm/master.svg?style=for-the-badge)](https://travis-ci.org/jckuester/awsrm)

**Work in progress**

The [Unix Philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well) suggests
writing programs which `do one thing and do it well`, rather than adding more features to existing programs.

`awsrm` does one thing: deleting AWS resources via the command line.

Like other Unix-like tools, `awsrm` shows its real power when being combined via pipes with other tools,
such as [`awsls`](https://github.com/jckuester/awsls) for listing resources or `grep` for filtering resources.

## Example

![](img/pipe-iam-role.gif)

1. List resources via [`awsls`](https://github.com/jckuester/awsls) with the attributes you want to filter on
   (here: `-a tags`)
2. Use standard tools, such as grep, to filter the resources to delete
3. Pipe result into `awsrm` (nothing is deleted until you confirm)

The following example deletes all AWS instances with tag `Name=foo` in the AWS accounts associated with
 profile `myaccount1` and `myaccount2` in both regions `us-west-2` and `us-east-1`:

    awsls -p myaccount1 -r us-west-2 instance -a tags | grep Name=foo | awsrm
    
## Install

Until the first release:
    
    go build
    cp awsrm ~/bin   