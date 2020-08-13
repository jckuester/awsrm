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
	var dryRun bool
	var force bool
	var logDebug bool
	var version bool

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		printHelp(flags)
	}

	flags.BoolVar(&dryRun, "dry-run", false, "Show what would be destroyed")
	flags.BoolVar(&force, "force", false, "Destroy without asking for confirmation")
	flags.BoolVar(&logDebug, "debug", false, "Enable debug logging")
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

	if isInputFromPipe() {
		log.Debug("input is from pipe")

		readResources, err := readResources(os.Stdin)
		if err != nil {
			log.Fatal(err.Error())
		}

		resources = readResources

		err = os.Stdin.Close()
		if err != nil {
			fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))

			return 1
		}
	} else {
		if len(args) < 3 {
			fmt.Fprint(os.Stderr, color.RedString("Error: not enough arguments given\n"))
			printHelp(flags)

			return 1
		}

		resources = append(resources, awsls.Resource{
			Type:    args[0],
			ID:      args[1],
			Profile: args[2],
			Region:  args[3],
		})
	}

	var clientKeys []util.AWSClientKey
	for _, r := range resources {
		clientKeys = append(clientKeys, util.AWSClientKey{
			Profile: r.Profile,
			Region:  r.Region,
		})
	}

	providers, err := util.NewProviderPool(clientKeys)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))

		return 1
	}

	resourcesWithUpdatedState := resource.GetStates(resources, providers)

	if !force {
		internal.LogTitle("showing resources that would be deleted (dry run)")

		// always show the resources that would be affected before deleting anything
		for _, r := range resourcesWithUpdatedState {
			log.WithField("id", r.ID).Warn(internal.Pad(r.Type))
		}

		if len(resourcesWithUpdatedState) == 0 {
			internal.LogTitle("all resources have already been deleted")
			return 0
		}

		internal.LogTitle(fmt.Sprintf("total number of resources that would be deleted: %d",
			len(resourcesWithUpdatedState)))
	}

	if !dryRun {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			log.Fatalf("can't open /dev/tty: %s", err)
		}

		if !internal.UserConfirmedDeletion(tty, force) {
			return 0
		}

		internal.LogTitle("Starting to delete resources")

		numDeletedResources := terradozerRes.DestroyResources(
			convertToDestroyableResources(resourcesWithUpdatedState), 5)

		internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))
	}

	return 0
}

func isInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()
	return fileInfo.Mode()&os.ModeCharDevice == 0
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
awsrm - Remove AWS resources via the CLI.

USAGE:
  $ awsrm [flags] <resource_type> <resource_id> <profile> <region>

FLAGS:
`
