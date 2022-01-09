package test

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"testing"

	"github.com/apex/log"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/onsi/gomega/gexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: we can currently only test pipe input in dry-run mode,
// as I couldn't find a way to get awsls input via stdIn and user input to confirm deletion from tty
// when running the awsrm binary under test using the exec package.
func TestAcc_Pipe_InputFromAwsls(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if testing.Short() {
		t.Skip("Skipping acceptance test.")
	}

	testVars := Init(t)

	terraformDir := "./test-fixtures/multiple-profiles-and-regions"

	terraformOptions := GetTerraformOptions(terraformDir, testVars, map[string]interface{}{
		"profile1": testVars.AWSProfile1,
		"profile2": testVars.AWSProfile2,
		"region1":  testVars.AWSRegion1,
		"region2":  testVars.AWSRegion2,
	})

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	vpc1 := terraform.Output(t, terraformOptions, "vpc1")
	AssertVpcExists(t, vpc1, testVars.AWSProfile1, testVars.AWSRegion1)

	vpc2 := terraform.Output(t, terraformOptions, "vpc2")
	AssertVpcExists(t, vpc2, testVars.AWSProfile1, testVars.AWSRegion2)

	vpc3 := terraform.Output(t, terraformOptions, "vpc3")
	AssertVpcExists(t, vpc3, testVars.AWSProfile2, testVars.AWSRegion1)

	vpc4 := terraform.Output(t, terraformOptions, "vpc4")
	AssertVpcExists(t, vpc4, testVars.AWSProfile2, testVars.AWSRegion2)

	tests := []struct {
		name                    string
		awslsArgs               []string
		grepArgs                []string
		awsrmArgs               []string
		envs                    map[string]string
		expectResourceIsDeleted bool
		expectedLogs            []string
		unexpectedLogs          []string
		expectedErrCode         int
	}{
		{
			name: "single resource",
			awslsArgs: []string{
				"-p", testVars.AWSProfile1,
				"-r", testVars.AWSRegion1,
				"-a", "tags", "aws_vpc"},
			grepArgs:                []string{"foo"},
			awsrmArgs:               []string{"--dry-run"},
			expectResourceIsDeleted: false,
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED \\(DRY RUN\\)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc1, testVars.AWSProfile1, testVars.AWSRegion1),
			},
			unexpectedLogs: []string{
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES:",
			},
		},
		{
			name: "multiple profiles and regions via awsls flag",
			awslsArgs: []string{
				"-p", fmt.Sprintf("%s,%s", testVars.AWSProfile1, testVars.AWSProfile2),
				"-r", fmt.Sprintf("%s,%s", testVars.AWSRegion1, testVars.AWSRegion2),
				"-a", "tags", "aws_vpc"},
			grepArgs:                []string{"awsrm=test-acc"},
			awsrmArgs:               []string{"--dry-run"},
			expectResourceIsDeleted: false,
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED \\(DRY RUN\\)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 4",
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc1, testVars.AWSProfile1, testVars.AWSRegion1),
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc2, testVars.AWSProfile1, testVars.AWSRegion2),
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc3, testVars.AWSProfile2, testVars.AWSRegion1),
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc4, testVars.AWSProfile2, testVars.AWSRegion2),
			},
			unexpectedLogs: []string{
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES:",
			},
		},
		{
			name: "force flag",
			awslsArgs: []string{
				"-p", testVars.AWSProfile1,
				"-r", testVars.AWSRegion1,
				"-a", "tags", "aws_vpc"},
			grepArgs:                []string{"foo"},
			awsrmArgs:               []string{"--force"},
			expectResourceIsDeleted: true,
			expectedLogs: []string{
				"PROCEEDING WITH DELETION AND SKIPPING CONFIRMATION (FORCE)",
				"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc1, testVars.AWSProfile1, testVars.AWSRegion1),
			},
			unexpectedLogs: []string{
				"Are you sure you want to delete these resources (cannot be undone)? Only YES will be accepted.",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := UnsetAWSEnvs()
			require.NoError(t, err)

			err = SetMultiEnvs(tc.envs)
			require.NoError(t, err)

			logBuffer := runBinaryWithPipeInputFromAwsls(t, tc.awslsArgs, tc.grepArgs, tc.awsrmArgs)

			if tc.expectedErrCode > 0 {
				assert.EqualError(t, err, "exit status 1")
			} else {
				assert.NoError(t, err)
			}

			AssertVpcExists(t, vpc1, testVars.AWSProfile1, testVars.AWSRegion1)
			AssertVpcExists(t, vpc2, testVars.AWSProfile1, testVars.AWSRegion2)
			AssertVpcExists(t, vpc3, testVars.AWSProfile2, testVars.AWSRegion1)
			AssertVpcExists(t, vpc4, testVars.AWSProfile2, testVars.AWSRegion2)

			actualLogs := logBuffer.String()

			for _, expectedLogEntry := range tc.expectedLogs {
				assert.Regexp(t, regexp.MustCompile(expectedLogEntry), actualLogs)
			}

			for _, unexpectedLogEntry := range tc.unexpectedLogs {
				assert.NotContains(t, actualLogs, unexpectedLogEntry)
			}

			fmt.Println(actualLogs)

			err = UnsetAWSEnvs()
			require.NoError(t, err)
		})
	}
}

func TestAcc_Pipe_UnsupportedResourceType(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if testing.Short() {
		t.Skip("Skipping acceptance test.")
	}

	logBuffer, err := runBinaryWithInputFromPipe(t,
		[]string{"--dry-run"},
		"aws_unsupported someId myaccount us-west-2")
	assert.Error(t, err)

	actualLogs := logBuffer.String()

	assert.Contains(t, actualLogs, "Error: no resource type found: aws_unsupported")

	fmt.Println(actualLogs)
}

func TestAcc_Pipe_ResourceTypeWithoutAWSPrefix(t *testing.T) {
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

	logBuffer, err := runBinaryWithInputFromPipe(t,
		[]string{"--dry-run"},
		fmt.Sprintf("vpc %s %s %s", actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1))
	assert.NoError(t, err)

	actualLogs := logBuffer.String()

	expectedLogs := []string{
		"TOTAL NUMBER OF RESOURCES THAT WOULD BE DELETED: 1",
		fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
			actualVpcID1, testVars.AWSProfile1, testVars.AWSRegion1),
	}

	for _, expectedLogEntry := range expectedLogs {
		assert.Regexp(t, regexp.MustCompile(expectedLogEntry), actualLogs)
	}

	fmt.Println(actualLogs)
}

func runBinaryWithInputFromPipe(t *testing.T, awsrmArgs []string, inputFromPipe string) (*bytes.Buffer, error) {
	defer gexec.CleanupBuildArtifacts()

	compiledPathAwsrm, err := gexec.Build(packagePath)
	require.NoError(t, err)

	logBuffer := &bytes.Buffer{}

	awsrm := exec.Command(compiledPathAwsrm, awsrmArgs...)
	require.NoError(t, err)
	awsrm.Stdin = bytes.NewBufferString(inputFromPipe)
	awsrm.Stdout = logBuffer
	awsrm.Stderr = logBuffer

	err = awsrm.Run()

	return logBuffer, err
}

func runBinaryWithPipeInputFromAwsls(t *testing.T, awslsArgs, grepArgs, awsrmArgs []string) *bytes.Buffer {
	defer gexec.CleanupBuildArtifacts()

	compiledPath, err := gexec.Build("github.com/jckuester/awsls")
	require.NoError(t, err)

	logBuffer := &bytes.Buffer{}

	var awslsOut bytes.Buffer

	awsls := exec.Command(compiledPath, awslsArgs...)
	awsls.Stdout = &awslsOut
	awsls.Stderr = logBuffer

	err = awsls.Run()
	assert.NoError(t, err)

	var egrepOut bytes.Buffer

	egrep := exec.Command("grep", grepArgs...)
	egrep.Stdin = bytes.NewReader(awslsOut.Bytes())
	egrep.Stdout = &egrepOut
	egrep.Stderr = logBuffer

	err = egrep.Run()
	assert.NoError(t, err)

	compiledPathAwsrm, err := gexec.Build(packagePath)
	require.NoError(t, err)

	awsrm := exec.Command(compiledPathAwsrm, awsrmArgs...)
	require.NoError(t, err)
	awsrm.Stdin = bytes.NewReader(egrepOut.Bytes())
	awsrm.Stdout = logBuffer
	awsrm.Stderr = logBuffer

	err = awsrm.Run()
	assert.NoError(t, err)

	return logBuffer
}
