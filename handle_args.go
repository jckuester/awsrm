package main

import (
	"fmt"
	"os"

	"github.com/jckuester/awsrm/pkg/resource"

	"github.com/apex/log"

	"github.com/fatih/color"
	awsls "github.com/jckuester/awsls/aws"
	"github.com/jckuester/awsls/util"
)

func handleInputFromArgs(args []string, profile, region string, dryRun bool) int {
	log.Debug("input via args")

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

	var resources []awsls.Resource
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

	var clientKeys []util.AWSClientKey
	for k := range clients {
		clientKeys = append(clientKeys, k)
	}

	err = resource.Delete(clientKeys, resources, os.Stdin, dryRun)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	return 0
}
