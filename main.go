package main

import (
	"context"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"os/signal"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/jckuester/awsrm/internal"
	flag "github.com/spf13/pflag"
)

const terraformAwsProviderVersion = "v3.42.0"

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
	flags.BoolVar(&dryRun, "dry-run", false, "Don't delete anything, just show what would be deleted")
	flags.StringVarP(&profile, "profile", "p", "", "The AWS profile for the account to delete resources in")
	flags.StringVarP(&region, "region", "r", "", "The region to delete resources in")
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

	ctx := context.Background()

	// trap Ctrl+C and call cancel on the context
	// to close running Terraform AWS provider plugins properly
	ctx, cancel := context.WithCancel(ctx)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, ignoreSignals...)
	signal.Notify(signalCh, forwardSignals...)
	defer func() {
		signal.Stop(signalCh)
		cancel()
	}()
	go func() {
		select {
		case <-signalCh:
			fmt.Fprint(os.Stderr, color.RedString("\nAborting...\n"))
			cancel()
		case <-ctx.Done():
		}
	}()

	if isInputFromPipe() {
		return handleInputFromPipe(ctx, dryRun)
	}

	if len(args) < 2 {
		printHelp(flags)
		return 1
	}

	return handleInputFromArgs(ctx, args, profile, region, dryRun)
}

func printHelp(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\n"+strings.TrimSpace(help)+"\n")
	fs.PrintDefaults()
}

const help = `
awsrm - A remove command for AWS resources.

USAGE:
  $ awsrm [flags] <resource_type> <id> [<id>...]

The resource type and ID(s) are required arguments to delete resource(s).
If no profile and/or region for an AWS account is given, credentials are
used by the usual precedence of the AWS CLI: environment variables, AWS credentials file, etc.

Resources in multiple accounts and regions can be filtered and deleted by piping
the output of awsls through grep to awsrm:

  $ awsls [profile/region flags] vpc -a tags | grep Name=foo | awsrm

For supported resource types and a full help text, see the README in the GitHub repository
https://github.com/jckuester/awsrm and https://github.com/jckuester/awsls.

FLAGS:
`
