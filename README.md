# awsrm

A remove command for AWS resources

[![Release](https://img.shields.io/github/release/jckuester/awsrm.svg?style=for-the-badge)](https://github.com/jckuester/awsrm/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Travis](https://img.shields.io/travis/jckuester/awsrm/master.svg?style=for-the-badge)](https://travis-ci.org/jckuester/awsrm)

**Work in progress (but already usable)**

The [Unix Philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well) suggests
writing programs which `do one thing and do it well`, rather than adding more features to existing programs.

`awsrm` does one thing: deleting AWS resources via the command line.

Like other Unix-like tools, `awsrm` shows its real power when being combined via pipes with other tools,
such as [`awsls`](https://github.com/jckuester/awsls) for listing or `grep` for filtering resources.

## Example

    awsls "aws_instance" -a tags | grep Name=foo | awsrm
    
## Install

Until the first release:
    
    go build
    cp awsrm ~/bin   