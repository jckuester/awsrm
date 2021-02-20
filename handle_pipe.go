package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/jckuester/awsrm/pkg/resource"
	"github.com/jckuester/awstools-lib/aws"
	"github.com/jckuester/awstools-lib/terraform"
)

func isInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()
	return fileInfo.Mode()&os.ModeNamedPipe != 0
}

func handleInputFromPipe(ctx context.Context, dryRun bool) int {
	log.Debug("input via pipe")

	resources, err := resource.Read(os.Stdin)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	var clientKeys []aws.ClientKey
	for _, r := range resources {
		clientKeys = append(clientKeys, aws.ClientKey{Profile: r.Profile, Region: r.Region})
	}

	err = os.Stdin.Close()
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
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
			fmt.Fprint(os.Stderr, color.RedString("Error %s: %s\n", err))
		}
	}

	confirmDevice, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("can't open /dev/tty: %s", err)
	}

	doneDelete := make(chan bool, 1)
	go func() {
		resource.Delete(resources, confirmDevice, dryRun, doneDelete)
	}()
	select {
	case <-ctx.Done():
		return 0
	case <-doneDelete:
	}

	return 0
}
