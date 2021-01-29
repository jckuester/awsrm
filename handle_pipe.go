package main

import (
	"fmt"
	"os"

	"github.com/jckuester/awsrm/pkg/resource"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/jckuester/awsls/util"
)

func isInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()
	log.Debugf("%v\n", fileInfo.Mode())

	return fileInfo.Mode()&os.ModeNamedPipe != 0
}

func handleInputFromPipe(dryRun bool) int {
	log.Debug("input via pipe")

	resources, err := resource.Read(os.Stdin)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	var clientKeys []util.AWSClientKey
	for _, r := range resources {
		clientKeys = append(clientKeys, util.AWSClientKey{Profile: r.Profile, Region: r.Region})
	}

	err = os.Stdin.Close()
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	confirmDevice, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("can't open /dev/tty: %s", err)
	}

	err = resource.Delete(clientKeys, resources, confirmDevice, dryRun)
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString("\nError: %s\n", err))
		return 1
	}

	return 0
}
