package resource

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/apex/log"
	"github.com/jckuester/awsrm/internal"
	"github.com/jckuester/awstools-lib/aws"
	"github.com/jckuester/awstools-lib/terraform"
	"github.com/jckuester/terradozer/pkg/provider"
	terradozerRes "github.com/jckuester/terradozer/pkg/resource"
)

type UpdatedResources struct {
	Resources []terraform.Resource
	Errors    []error
}

// Update fetches the Terraform state for the given resources. A state is needed to delete resources
// via the Delete() function, which calls the Terraform AWS provider for deletion.
func Update(resources []terraform.Resource, providers map[aws.ClientKey]provider.TerraformProvider) UpdatedResources {
	withUpdatedState, errs := terraform.UpdateStates(resources, providers, 10, false)

	var resourcesAlreadyDeleted []terraform.Resource
	var resourcesToDelete []terraform.Resource

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

	return UpdatedResources{resourcesToDelete, errs}
}

// Delete deletes the given resources via the Terraform AWS Provider.
func Delete(resources []terraform.Resource, confirmDevice io.Reader, force bool, dryRun bool, done chan bool) {
	if len(resources) == 0 {
		internal.LogTitle("no resources found to delete")
		done <- true
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
		if !force {
			if !internal.UserConfirmedDeletion(confirmDevice) {
				done <- true
				return
			}
		} else {
			internal.LogTitle("Proceeding with deletion and skipping confirmation (Force)")
		}

		internal.LogTitle("Starting to delete resources")

		numDeletedResources := terradozerRes.DestroyResources(convertToDestroyable(resources), 5)

		internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))
	}

	done <- true
}

// Read reads resources from stdIn (when input is coming from pipe), where a line must be of the following format:
// 	<resource_type> <resource_id> <profile> <region>\n
func Read(r io.Reader) ([]terraform.Resource, error) {
	var result []terraform.Resource

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
		if !terraform.IsType(rType) {
			return nil, fmt.Errorf("no resource type found: %s\n", rType)
		}

		profile := rAttrs[2]

		if profile == `N\A` {
			profile = ""
		}

		result = append(result, terraform.Resource{
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

func convertToDestroyable(resources []terraform.Resource) []terradozerRes.DestroyableResource {
	var result []terradozerRes.DestroyableResource

	for _, r := range resources {
		result = append(result, r.UpdatableResource.(terradozerRes.DestroyableResource))
	}

	return result
}
