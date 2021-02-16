package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	awsls "github.com/jckuester/awsls/aws"
	awslsRes "github.com/jckuester/awsls/resource"
	"github.com/jckuester/awsrm/pkg/resource"
	"github.com/jckuester/awstools-lib/aws"
	"github.com/jckuester/awstools-lib/terraform"
	"golang.org/x/net/context"
)

func handleInputFromArgs(ctx context.Context, args []string, profile, region string, dryRun bool) int {
	log.Debug("input via args")

	rType := resource.PrefixResourceType(args[0])
	if !awslsRes.IsSupportedType(rType) {
		fmt.Fprint(os.Stderr, color.RedString("\nError: no resource type found: %s\n", rType))
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

	clients, err := aws.NewClientPool(profiles, regions)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	var resources []awsls.Resource
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

	var clientKeys []aws.ClientKey
	for k := range clients {
		clientKeys = append(clientKeys, k)
	}

	providers, err := terraform.NewProviderPool(ctx, clientKeys, "v3.16.0", "~/.awsrm", 1*time.Minute)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		}
		return 1
	}
	defer func() {
		for _, p := range providers {
			_ = p.Close()
		}
	}()

	resourcesCh := make(chan resource.UpdatedResources, 1)
	go func() { resourcesCh <- resource.Update(resources, providers) }()
	select {
	case <-ctx.Done():
		return 1
	case result := <-resourcesCh:
		resources = result.Resources

		for _, err := range result.Errors {
			fmt.Fprint(os.Stderr, color.RedString("Error %s: %s\n", rType, err))
		}
	}

	doneDelete := make(chan bool, 1)
	go func() { resource.Delete(resources, os.Stdin, dryRun, doneDelete) }()
	select {
	case <-ctx.Done():
		return 0
	case <-doneDelete:
	}

	return 0
}
