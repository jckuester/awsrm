package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/apex/log"

	"github.com/gruntwork-io/terratest/modules/terraform"

	"github.com/onsi/gomega/gexec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	packagePath = "github.com/jckuester/awsrm"
)

func TestAcc_Args_UserConfirmation(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if testing.Short() {
		t.Skip("Skipping acceptance test.")
	}

	tests := []struct {
		name                    string
		userInput               string
		expectResourceIsDeleted bool
		expectedLogs            []string
		unexpectedLogs          []string
	}{
		{
			name:                    "confirmed with YES",
			userInput:               "YES\n",
			expectResourceIsDeleted: true,
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED (DRY RUN)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
				"Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.",
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES: 1",
			},
		},
		{
			name:                    "confirmed with yes",
			userInput:               "yes\n",
			expectResourceIsDeleted: true,
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED (DRY RUN)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
				"Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.",
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES: 1",
			},
		},
		{
			name:      "confirmed with no",
			userInput: "no\n",
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED (DRY RUN)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
				"Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.",
			},
			unexpectedLogs: []string{
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES:",
			},
		},
		//{
		//	name:      "dry run",
		//	expectedLogs: []string{
		//		"SHOWING RESOURCES THAT WOULD BE DELETED (DRY RUN)",
		//		"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
		//	},
		//	unexpectedLogs: []string{
		//		"STARTING TO DELETE RESOURCES",
		//		"TOTAL NUMBER OF DELETED RESOURCES:",
		//		"Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.",
		//	},
		//},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testVars := Init(t)

			terraformDir := "./test-fixtures/vpc"

			terraformOptions := GetTerraformOptions(terraformDir, testVars)

			defer terraform.Destroy(t, terraformOptions)

			terraform.InitAndApply(t, terraformOptions)

			actualVpcID1 := terraform.Output(t, terraformOptions, "vpc_id1")
			AssertVpcExists(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)

			actualVpcID2 := terraform.Output(t, terraformOptions, "vpc_id2")
			AssertVpcExists(t, actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1)

			logBuffer := runBinary(t, tc.userInput,
				"-p", testVars.AWSProfile1,
				"-r", testVars.AWSRegion1,
				"aws_vpc", actualVpcID1)

			if tc.expectResourceIsDeleted {
				AssertVpcDeleted(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)
			} else {
				AssertVpcExists(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)
			}

			AssertVpcExists(t, actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1)
			actualLogs := logBuffer.String()

			for _, expectedLogEntry := range tc.expectedLogs {
				assert.Contains(t, actualLogs, expectedLogEntry)
			}

			for _, unexpectedLogEntry := range tc.unexpectedLogs {
				assert.NotContains(t, actualLogs, unexpectedLogEntry)
			}

			fmt.Println(actualLogs)
		})
	}
}

func TestAcc_Args_MultipleResourceIDs(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if testing.Short() {
		t.Skip("Skipping acceptance test.")
	}

	testVars := Init(t)

	terraformDir := "./test-fixtures/vpc"

	terraformOptions := GetTerraformOptions(terraformDir, testVars)

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	actualVpcID1 := terraform.Output(t, terraformOptions, "vpc_id1")
	AssertVpcExists(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)

	actualVpcID2 := terraform.Output(t, terraformOptions, "vpc_id2")
	AssertVpcExists(t, actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1)

	actualVpcID3 := terraform.Output(t, terraformOptions, "vpc_id3")
	AssertVpcExists(t, actualVpcID3, testVars.AWSProfile1, testVars.AWSRegion1)

	logBuffer := runBinary(t, "yes\n",
		"-p", testVars.AWSProfile1,
		"-r", testVars.AWSRegion1,
		"aws_vpc", actualVpcID1, actualVpcID2, actualVpcID3)

	AssertVpcDeleted(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)
	AssertVpcDeleted(t, actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1)
	AssertVpcDeleted(t, actualVpcID3, testVars.AWSProfile1, testVars.AWSRegion1)

	actualLogs := logBuffer.String()

	expectedLogs := []string{
		"TOTAL NUMBER OF DELETED RESOURCES: 3",
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1),
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1),
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			actualVpcID2, testVars.AWSProfile1, testVars.AWSRegion1),
	}

	for _, expectedLogEntry := range expectedLogs {
		assert.Regexp(t, regexp.MustCompile(expectedLogEntry), actualLogs)
	}

	fmt.Println(actualLogs)
}

func TestAcc_Args_NonExistingResourceID(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if testing.Short() {
		t.Skip("Skipping acceptance test.")
	}

	testVars := Init(t)

	terraformDir := "./test-fixtures/vpc"

	terraformOptions := GetTerraformOptions(terraformDir, testVars)

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	actualVpcID1 := terraform.Output(t, terraformOptions, "vpc_id1")
	AssertVpcExists(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)

	logBuffer := runBinary(t, "yes\n",
		"-p", testVars.AWSProfile1,
		"-r", testVars.AWSRegion1,
		"aws_vpc", "nonExistingID", actualVpcID1)

	AssertVpcDeleted(t, actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1)

	actualLogs := logBuffer.String()

	expectedLogs := []string{
		"TOTAL NUMBER OF DELETED RESOURCES: 1",
		"THE FOLLOWING RESOURCES DON'T EXIST",
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1),
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			"nonExistingID", testVars.AWSProfile1, testVars.AWSRegion1),
	}

	for _, expectedLogEntry := range expectedLogs {
		assert.Regexp(t, regexp.MustCompile(expectedLogEntry), actualLogs)
	}

	fmt.Println(actualLogs)
}

func runBinary(t *testing.T, userInput string, args ...string) *bytes.Buffer {
	defer gexec.CleanupBuildArtifacts()

	compiledPath, err := gexec.Build(packagePath)
	require.NoError(t, err)

	// if we don't provide user input via file to Stdin,
	// the exec package delivers input via pipe (which is not what we want)
	stdinFile, err := ioutil.TempFile("", "stdinFile")
	require.NoError(t, err)

	defer os.Remove(stdinFile.Name())

	stdIn, err := os.Create(stdinFile.Name())
	require.NoError(t, err)

	_, err = stdIn.Write([]byte(userInput))
	require.NoError(t, err)

	err = stdIn.Close()
	require.NoError(t, err)

	stdIn, err = os.OpenFile(stdinFile.Name(), os.O_RDONLY, os.ModeAppend)
	require.NoError(t, err)

	logBuffer := &bytes.Buffer{}

	p := exec.Command(compiledPath, args...)
	p.Stdin = stdIn
	p.Stdout = logBuffer
	p.Stderr = logBuffer

	err = p.Run()
	assert.NoError(t, err)

	return logBuffer
}
