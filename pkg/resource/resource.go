package resource

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	awslsRes "github.com/jckuester/awsls/resource"

	"github.com/jckuester/terradozer/pkg/provider"

	"github.com/jckuester/awstools-lib/aws"

	"github.com/jckuester/awstools-lib/terraform"

	"github.com/apex/log"
	awsls "github.com/jckuester/awsls/aws"
	"github.com/jckuester/awsrm/internal"
	terradozerRes "github.com/jckuester/terradozer/pkg/resource"
)

type UpdatedResources struct {
	Resources []awsls.Resource
	Errors    []error
}

func Update(resources []awsls.Resource, providers map[aws.ClientKey]provider.TerraformProvider) UpdatedResources {
	withUpdatedState, errs := terraform.UpdateStates(resources, providers, 10)

	// TODO introduce exists flag in resource type
	var resourcesAlreadyDeleted []awsls.Resource
	var resourcesToDelete []awsls.Resource

	for _, r := range withUpdatedState {
		if r.State().IsNull() {
			resourcesAlreadyDeleted = append(resourcesAlreadyDeleted, r)
		} else {
			resourcesToDelete = append(resourcesToDelete, r)
		}
	}

	if len(resourcesAlreadyDeleted) != 0 {
		internal.LogTitle("the following resources don't exist")
	}

	for _, r := range resourcesAlreadyDeleted {
		log.WithFields(log.Fields{
			"id":      r.ID,
			"profile": r.Profile,
			"region":  r.Region,
		}).Info(internal.Pad(r.Type))
	}

	return UpdatedResources{resources, errs}
}

func Delete(resources []awsls.Resource, confirmDevice io.Reader, dryRun bool, done chan bool) {
	if len(resources) == 0 {
		internal.LogTitle("no resources found to delete")
		return
	}

	// always show the resources that would be affected before deleting anything
	if len(resources) != 0 {
		internal.LogTitle("showing resources that would be deleted (dry run)")
	}
	for _, r := range resources {
		if r.State() != nil {
			log.WithFields(log.Fields{
				"id":      r.ID,
				"profile": r.Profile,
				"region":  r.Region,
			}).Warn(internal.Pad(r.Type))
		}
	}

	internal.LogTitle(fmt.Sprintf("total number of resources that would be deleted: %d", len(resources)))

	if !dryRun && len(resources) > 0 {
		if !internal.UserConfirmedDeletion(confirmDevice) {
			done <- true
		}

		internal.LogTitle("Starting to delete resources")

		numDeletedResources := terradozerRes.DestroyResources(convertToDestroyable(resources), 5)

		internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))
	}

	done <- true
}

// Read reads resources from stdIn (when input is coming from pipe), where a line must be of the following format:
// 	<resource_type> <resource_id> <profile> <region>\n
func Read(r io.Reader) ([]awsls.Resource, error) {
	var result []awsls.Resource

	scanner := bufio.NewScanner(bufio.NewReader(r))
	for scanner.Scan() {
		line := scanner.Text()

		// ignore empty lines and header lines of awsls beginning with "TYPE ID..."
		if line == "\n" || line == "" || strings.HasPrefix(line, "TYPE") {
			continue
		}

		rAttrs := strings.Fields(line)
		if len(rAttrs) < 4 {
			return nil, fmt.Errorf("input must be of form: <resource_type> <resource_id> <profile> <region>")
		}

		rType := PrefixResourceType(rAttrs[0])
		if !awslsRes.IsSupportedType(rType) {
			return nil, fmt.Errorf("no resource type found: %s\n", rType)
		}

		profile := rAttrs[2]

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

func convertToDestroyable(resources []awsls.Resource) []terradozerRes.DestroyableResource {
	var result []terradozerRes.DestroyableResource

	for _, r := range resources {
		result = append(result, r.UpdatableResource.(terradozerRes.DestroyableResource))
	}

	return result
}
