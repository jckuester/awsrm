package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	awsls "github.com/jckuester/awsls/aws"
	"github.com/jckuester/awsls/resource"
	"github.com/jckuester/awsls/util"
	"github.com/jckuester/awsrm/internal"
	terradozerRes "github.com/jckuester/terradozer/pkg/resource"
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

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		printHelp(flags)
	}

	flags.BoolVar(&logDebug, "debug", false, "Enable debug logging")
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

	var resources []awsls.Resource
	var err error
	var confirmDevice *os.File
	var clientKeys []util.AWSClientKey

	if isInputFromPipe() {
		log.Debug("input via pipe")

		resources, err = readResources(os.Stdin)
		if err != nil {
			fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
			return 1
		}

		err = os.Stdin.Close()
		if err != nil {
			fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
			return 1
		}

		confirmDevice, err = os.Open("/dev/tty")
		if err != nil {
			log.Fatalf("can't open /dev/tty: %s", err)
		}
	} else {
		log.Debug("input via args")

		if len(args) < 2 {
			printHelp(flags)
			return 1
		}

		var profiles []string
		var regions []string

		if profile != "" {
			profiles = []string{profile}
		} else {
			env, ok := os.LookupEnv("AWS_PROFILE")
			if ok {
				profiles = []string{env}
			}
		}

		if region != "" {
			regions = []string{region}
		}

		clients, err := util.NewAWSClientPool(profiles, regions)
		if err != nil {
			fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
			return 1
		}

		rType := args[0]
		for _, client := range clients {
			for _, id := range args[1:] {
				resources = append(resources, awsls.Resource{
					Type:    rType,
					ID:      id,
					Profile: client.Profile,
					Region:  client.Region,
				})
			}
		}

		for k := range clients {
			clientKeys = append(clientKeys, k)
		}

		confirmDevice = os.Stdin
	}

	providers, err := util.NewProviderPool(clientKeys)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	resourcesWithUpdatedState := resource.GetStates(resources, providers)

	internal.LogTitle("showing resources that would be deleted (dry run)")

	// always show the resources that would be affected before deleting anything
	for _, r := range resourcesWithUpdatedState {
		log.WithFields(log.Fields{
			"id":      r.ID,
			"profile": r.Profile,
			"region":  r.Region,
		}).Warn(internal.Pad(r.Type))
	}

	if len(resourcesWithUpdatedState) == 0 {
		internal.LogTitle("all resources have already been deleted")
		return 0
	}

	internal.LogTitle(fmt.Sprintf("total number of resources that would be deleted: %d",
		len(resourcesWithUpdatedState)))

	if !internal.UserConfirmedDeletion(confirmDevice, false) {
		return 0
	}

	internal.LogTitle("Starting to delete resources")

	numDeletedResources := terradozerRes.DestroyResources(
		convertToDestroyableResources(resourcesWithUpdatedState), 5)

	internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))

	return 0
}

func isInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()
	log.Debugf("%v\n", fileInfo.Mode())

	return fileInfo.Mode()&os.ModeNamedPipe != 0
}

func readResources(r io.Reader) ([]awsls.Resource, error) {
	var result []awsls.Resource

	scanner := bufio.NewScanner(bufio.NewReader(r))
	for scanner.Scan() {
		line := scanner.Text()
		rAttrs := strings.Fields(line)
		if len(rAttrs) < 4 {
			return nil, fmt.Errorf("input must be of form: <resource_type> <resource_id> <profile> <region>")
		}

		rType := rAttrs[0]
		profile := rAttrs[2]

		if !resource.IsType(rType) {
			return nil, fmt.Errorf("is not a Terraform resource type: %s", rType)
		}

		if profile == `N\A` {
			profile = ""
		}

		result = append(result, awsls.Resource{
			Type:    rType,
			ID:      rAttrs[1],
			Profile: profile,
			Region:  rAttrs[3],
		})
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func convertToDestroyableResources(resources []awsls.Resource) []terradozerRes.DestroyableResource {
	var result []terradozerRes.DestroyableResource

	for _, r := range resources {
		result = append(result, r.UpdatableResource.(terradozerRes.DestroyableResource))
	}

	return result
}

func printHelp(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\n"+strings.TrimSpace(help)+"\n")
	fs.PrintDefaults()
}

const help = `
awsrm - A remove command for AWS resources.

USAGE:
  $ awsrm [flags] <type> <id> [<id>...]

The resource type and some ID(s) are required arguments to
delete resource(s). If no profile and/or region for an AWS account is given,
credentials will be searched for by the usual precedence of the
AWS CLI: environment variables, AWS credentials file, etc.

Resources in multiple accounts and regions can be filtered and deleted by piping
the output of awsls, for example, through grep to awsrm:

  $ awsls [profile/region flags] vpc -a tags | egrep 'Name=foo' | awsrm

For supported resource types and a full help text,
see the README in the GitHub repository
https://github.com/jckuester/awsrm and
https://github.com/jckuester/awsls.

FLAGS:
`
