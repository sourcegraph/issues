package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	gcp_crm "google.golang.org/api/cloudresourcemanager/v1"
	gcp_cp "google.golang.org/api/compute/v1"
	gcp_ct "google.golang.org/api/container/v1"
)

// for resources that require enumerating over regions, it is not very partical to
// make queries for regions that will either not work or will never have Sourcegraph
// resources - define relevant regions here.
//
// https://cloud.google.com/compute/docs/regions-zones#locations
var gcpLocationPrefixes = []string{
	"us-",
	"europe-",
}

type GCPResourceFetchFunc func(context.Context, chan<- Resource, string, time.Time) error

var gcpResources = map[string]GCPResourceFetchFunc{
	// fetch disks and instances
	"compute": func(ctx context.Context, results chan<- Resource, project string, since time.Time) error {
		client, err := gcp_cp.NewService(ctx)
		if err != nil {
			return errors.Errorf("failed to init client: %w", err)
		}

		// compute APIs require us to specify zones, so we must iterate over all zones
		if err := client.Zones.List(project).Pages(ctx, func(zones *gcp_cp.ZoneList) error {
			for _, zone := range zones.Items {
				if !hasPrefix(zone.Name, gcpLocationPrefixes) {
					continue // skip this zone
				}

				if err := client.Instances.List(project, zone.Name).
					Pages(ctx, func(instances *gcp_cp.InstanceList) error {
						for _, instance := range instances.Items {
							created, err := time.Parse(time.RFC3339, instance.CreationTimestamp)
							if err != nil {
								return errors.Errorf("could not parse create time for instance %s: %w", instance.Name, err)
							}
							machineTypeSegments := strings.Split(instance.MachineType, "/")
							machineType := machineTypeSegments[len(machineTypeSegments)-1]
							if created.After(since) {
								results <- Resource{
									Platform:   PlatformGCP,
									Identifier: instance.Name,
									Location:   zone.Name,
									Owner:      project,
									Type:       fmt.Sprintf("%s::%s", instance.Kind, machineType),
									Created:    created,
									Meta: map[string]interface{}{
										"labels": instance.Labels,
									},
								}
							}
						}
						return nil
					}); err != nil {
					return errors.Errorf("instances: %w", err)
				}

				if err := client.Disks.List(project, zone.Name).
					Pages(ctx, func(disks *gcp_cp.DiskList) error {
						for _, disk := range disks.Items {
							created, err := time.Parse(time.RFC3339, disk.CreationTimestamp)
							if err != nil {
								return errors.Errorf("could not parse create time for disk %s: %w", disk.Name, err)
							}
							diskTypeSegments := strings.Split(disk.Type, "/")
							diskType := diskTypeSegments[len(diskTypeSegments)-1]
							if created.After(since) {
								results <- Resource{
									Platform:   PlatformGCP,
									Identifier: disk.Name,
									Location:   zone.Name,
									Owner:      project,
									Type:       fmt.Sprintf("%s::%s::%dGB", disk.Kind, diskType, disk.SizeGb),
									Created:    created,
									Meta: map[string]interface{}{
										"labels": disk.Labels,
									},
								}
							}
						}
						return nil
					}); err != nil {
					return errors.Errorf("disks: %w", err)
				}
			}
			return nil
		}); err != nil {
			return err
		}

		return nil
	},
	// fetch clusters
	"containers": func(ctx context.Context, results chan<- Resource, project string, since time.Time) error {
		client, err := gcp_ct.NewService(ctx)
		if err != nil {
			return errors.Errorf("failed to init client: %w", err)
		}

		// cluster api allows us to query all locations at once
		parent := fmt.Sprintf("projects/%s/locations/-", project)
		list, err := client.Projects.Locations.Clusters.List(parent).Context(ctx).Do()
		if err != nil {
			return errors.Errorf("clusters: %w", err)
		}
		for _, cluster := range list.Clusters {
			created, err := time.Parse(time.RFC3339, cluster.CreateTime)
			if err != nil {
				return errors.Errorf("could not parse create time for cluster %s: %w", cluster.Name, err)
			}
			if created.After(since) {
				results <- Resource{
					Platform:   PlatformGCP,
					Identifier: cluster.Name,
					Type:       "container#cluster",
					Owner:      project,
					Location:   cluster.Zone,
					Created:    created,
					Meta: map[string]interface{}{
						"labels": cluster.ResourceLabels,
					},
				}
			}
		}
		return nil
	},
}

func collectGCPResources(ctx context.Context, since time.Time, verbose bool, labelsAllowlist map[string]string) ([]Resource, error) {
	logger := log.New(os.Stdout, "gcp: ", log.LstdFlags|log.Lmsgprefix)
	if verbose {
		logger.Printf("collecting resources since %s", since)
	}

	crm, err := gcp_crm.NewService(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to init cloud resources client: %w", err)
	}

	results := make(chan Resource, resultsBuffer)
	errs := make(chan error)
	wait := &sync.WaitGroup{}

	// aggregate resources for each GCP project
	if err := crm.Projects.List().Pages(ctx, func(page *gcp_crm.ListProjectsResponse) error {
		for _, project := range page.Projects {
			if verbose {
				logger.Printf("querying project %s", project.Name)
			}
			for resourceID, fetchResource := range gcpResources {
				wait.Add(1)
				go func(resourceID string, fetchResource GCPResourceFetchFunc, project string) {
					if err := fetchResource(ctx, results, project, since); err != nil {
						errs <- errors.Errorf("project %s, resource %s: %w", project, resourceID, err)
					}
					wait.Done()
				}(resourceID, fetchResource, project.ProjectId)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// collect results until done
	go func() {
		wait.Wait()
		close(results)
	}()
	var resources []Resource
	for {
		select {
		case r, ok := <-results:
			if ok {
				resource := r
				// allowlist resource if configured - all GCP labels are maps
				if labelsAllowlist != nil {
					if labels, ok := resource.Meta["labels"].(map[string]string); ok {
						if hasKeyValue(labels, labelsAllowlist) {
							resource.Allowed = true
						}
					}
				}
				resources = append(resources, resource)
			} else {
				return resources, nil
			}
		case err := <-errs:
			return nil, err
		}
	}
}
