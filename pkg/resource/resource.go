package resource

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/apex/log"
	awsls "github.com/jckuester/awsls/aws"
	"github.com/jckuester/awsls/resource"
	awslsRes "github.com/jckuester/awsls/resource"
	"github.com/jckuester/awsls/util"
	"github.com/jckuester/awsrm/internal"
	terradozerRes "github.com/jckuester/terradozer/pkg/resource"
)

func Delete(clientKeys []util.AWSClientKey, resources []awsls.Resource, confirmDevice io.Reader, dryRun bool) error {
	for _, r := range resources {
		if !awslsRes.IsSupportedType(r.Type) {
			return fmt.Errorf("no resource type found: %s\n", r.Type)
		}
	}

	providers, err := util.NewProviderPool(clientKeys)
	if err != nil {
		return err
	}

	resourcesWithUpdatedState := resource.GetStates(resources, providers)

	var resourcesAlreadyDeleted []awsls.Resource
	var resourcesToDelete []awsls.Resource

	for _, r := range resourcesWithUpdatedState {
		if r.State().IsNull() {
			resourcesAlreadyDeleted = append(resourcesAlreadyDeleted, r)
		} else {
			resourcesToDelete = append(resourcesToDelete, r)
		}
	}

	if len(resourcesToDelete) != 0 {
		internal.LogTitle("showing resources that would be deleted (dry run)")
	}

	// always show the resources that would be affected before deleting anything
	for _, r := range resourcesToDelete {
		if r.State() != nil {
			log.WithFields(log.Fields{
				"id":      r.ID,
				"profile": r.Profile,
				"region":  r.Region,
			}).Warn(internal.Pad(r.Type))
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

	if len(resourcesWithUpdatedState) == 0 {
		internal.LogTitle("no resources found to delete")
		return nil
	}

	internal.LogTitle(fmt.Sprintf("total number of resources that would be deleted: %d",
		len(resourcesToDelete)))

	if !dryRun && len(resourcesToDelete) > 0 {
		if !internal.UserConfirmedDeletion(confirmDevice) {
			return nil
		}

		internal.LogTitle("Starting to delete resources")

		numDeletedResources := terradozerRes.DestroyResources(
			convertToDestroyable(resourcesToDelete), 5)

		internal.LogTitle(fmt.Sprintf("total number of deleted resources: %d", numDeletedResources))
	}

	return nil
}

// Read resources from stdIn via pipe
// A line must be of the following form:
// 	<resource_type> <resource_id> <profile> <region>\n
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
