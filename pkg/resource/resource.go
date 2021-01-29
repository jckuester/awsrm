package resource

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/apex/log"
	awsls "github.com/jckuester/awsls/aws"
	"github.com/jckuester/awsls/resource"
	"github.com/jckuester/awsls/util"
	"github.com/jckuester/awsrm/internal"
	terradozerRes "github.com/jckuester/terradozer/pkg/resource"
)

func Delete(clientKeys []util.AWSClientKey, resources []awsls.Resource, confirmDevice io.Reader,
	dryRun bool) error {
	providers, err := util.NewProviderPool(clientKeys)
	if err != nil {
		return err
	}

	resourcesWithUpdatedState := resource.GetStates(resources, providers)

	internal.LogTitle("showing resources that would be deleted (dry run)")

	// always show the resources that would be affected before deleting anything
	for _, r := range resourcesWithUpdatedState {
		log.WithFields(log.Fields{
			"id":      r.ID,
			"profile": r.Profile,
			"region":  r.Region,
		}).Warn(internal.Pad(r.Type))
	}

	if len(resourcesWithUpdatedState) == 0 {
		internal.LogTitle("no resources found to delete")
		return nil
	}

	internal.LogTitle(fmt.Sprintf("total number of resources that would be deleted: %d",
		len(resourcesWithUpdatedState)))

	if !dryRun {
		if !internal.UserConfirmedDeletion(confirmDevice, false) {
			return nil
		}

		internal.LogTitle("Starting to deleteResources resources")

		numDeletedResources := terradozerRes.DestroyResources(
			convertToDestroyable(resourcesWithUpdatedState), 5)

		internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))
	}

	return nil
}

func Read(r io.Reader) ([]awsls.Resource, error) {
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

func convertToDestroyable(resources []awsls.Resource) []terradozerRes.DestroyableResource {
	var result []terradozerRes.DestroyableResource

	for _, r := range resources {
		result = append(result, r.UpdatableResource.(terradozerRes.DestroyableResource))
	}

	return result
}
