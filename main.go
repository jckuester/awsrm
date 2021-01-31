package main

import (
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/jckuester/awsrm/internal"
	flag "github.com/spf13/pflag"
)

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	var logDebug bool
	var version bool
	var profile string
	var region string
	var dryRun bool

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		printHelp(flags)
	}

	flags.BoolVar(&logDebug, "debug", false, "Enable debug logging")
	flags.BoolVar(&dryRun, "dry-run", false, "Don't deleteResources anything, just show what would be deleted")
	flags.StringVarP(&profile, "profile", "p", "", "The AWS profile for the account to deleteResources resources in")
	flags.StringVarP(&region, "region", "r", "", "The region to deleteResources resources in")
	flags.BoolVar(&version, "version", false, "Show application version")

	_ = flags.Parse(os.Args[1:])
	args := flags.Args()

	fmt.Println()
	defer fmt.Println()

	// discard TRACE logs of GRPCProvider
	stdlog.SetOutput(ioutil.Discard)

	log.SetHandler(cli.Default)

	if logDebug {
		log.SetLevel(log.DebugLevel)
	}

	if version {
		fmt.Println(internal.BuildVersionString())
		return 0
	}

	if isInputFromPipe() {
		return handleInputFromPipe(dryRun)
	}

	if len(args) < 2 {
		printHelp(flags)
		return 1
	}

	return handleInputFromArgs(args, profile, region, dryRun)
}

func printHelp(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\n"+strings.TrimSpace(help)+"\n")
	fs.PrintDefaults()
}

const help = `
awsrm - A remove command for AWS resources.

USAGE:
  $ awsrm [flags] <type> <id> [<id>...]

The resource type and ID(s) are required arguments to
delete some resource(s). If no profile and/or region for an AWS account is given,
credentials are used by the usual precedence of the
AWS CLI: environment variables, AWS credentials file, etc.

Resources in multiple accounts and regions can be filtered and deleted by piping
the output of awsls, for example, through grep to awsrm:

  $ awsls [profile/region flags] vpc -a tags | grep Name=foo | awsrm

For supported resource types and a full help text,
see the README in the GitHub repository
https://github.com/jckuester/awsrm and
https://github.com/jckuester/awsls.

FLAGS:
`
