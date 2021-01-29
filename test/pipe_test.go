package test

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"

	"github.com/apex/log"

	"github.com/onsi/gomega/gexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcc_DryRun(t *testing.T) {
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

	tests := []struct {
		name            string
		awslsArgs       []string
		grepArgs        []string
		awsrmArgs       []string
		envs            map[string]string
		expectedLogs    []string
		unexpectedLogs  []string
		expectedErrCode int
	}{
		{
			name: "single resource via pipe",
			awslsArgs: []string{
				"-p", fmt.Sprintf("%s", testVars.AWSProfile1),
				"-r", fmt.Sprintf("%s", testVars.AWSRegion1),
				"-a", "tags", "aws_vpc"},
			grepArgs:  []string{"foo"},
			awsrmArgs: []string{"--dry-run"},
			expectedLogs: []string{
				"SHOWING RESOURCES THAT WOULD BE DELETED (DRY RUN)",
				fmt.Sprintf("aws_vpc\\s+id=%s\\s+profile=%s\\s+region=%s",
					vpc1, testVars.AWSProfile1, testVars.AWSRegion1),
			},
			unexpectedLogs: []string{
				"STARTING TO DELETE RESOURCES",
				"TOTAL NUMBER OF DELETED RESOURCES:",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := UnsetAWSEnvs()
			require.NoError(t, err)

			err = SetMultiEnvs(tc.envs)
			require.NoError(t, err)

			logBuffer := runBinaryWithPipedOutputFromAwsls(t, tc.awslsArgs, tc.grepArgs, tc.awsrmArgs)

			if tc.expectedErrCode > 0 {
				assert.EqualError(t, err, "exit status 1")
			} else {
				assert.NoError(t, err)
			}

			AssertVpcExists(t, vpc1, testVars.AWSProfile1, testVars.AWSRegion1)
			AssertVpcExists(t, vpc2, testVars.AWSProfile1, testVars.AWSRegion2)

			actualLogs := logBuffer.String()

			for _, unexpectedLogEntry := range tc.unexpectedLogs {
				assert.NotContains(t, actualLogs, unexpectedLogEntry)
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

func runBinaryWithPipedOutputFromAwsls(t *testing.T, awslsArgs, grepArgs, awsrmArgs []string) *bytes.Buffer {
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
