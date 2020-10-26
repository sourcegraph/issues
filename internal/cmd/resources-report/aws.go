package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_ext "github.com/aws/aws-sdk-go-v2/aws/external"
	aws_ec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	aws_eks "github.com/aws/aws-sdk-go-v2/service/eks"
)

// for resources that require enumerating over regions, it is not very practical to
// make queries for regions that will either not work or will never have Sourcegraph
// resources - define relevant regions here.
//
// refer to https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.RegionsAndAvailabilityZones.html#Concepts.RegionsAndAvailabilityZones.Availability
var awsRegionPrefixes = []string{
	"us-",
	"eu-",
}

type AWSResourceFetchFunc func(context.Context, chan<- Resource, aws.Config, time.Time) error

// refer to https://docs.aws.amazon.com/sdk-for-go/v2/api/ for how to query resources under /service
var awsResources = map[string]AWSResourceFetchFunc{
	// fetch ec2 resources
	"EC2": func(ctx context.Context, results chan<- Resource, cfg aws.Config, since time.Time) error {
		// ECS not accessible in these regions
		if cfg.Region == "eu-south-1" {
			return nil
		}

		client := aws_ec2.New(cfg)

		instancesPager := aws_ec2.NewDescribeInstancesPaginator(client.DescribeInstancesRequest(&aws_ec2.DescribeInstancesInput{}))
		for instancesPager.Next(ctx) {
			page := instancesPager.CurrentPage()
			for _, reservation := range page.Reservations {
				for _, instance := range reservation.Instances {
					if instance.LaunchTime.After(since) {
						results <- Resource{
							Platform:   PlatformAWS,
							Identifier: *instance.InstanceId,
							Location:   *instance.Placement.AvailabilityZone,
							Owner:      "-",
							Type:       fmt.Sprintf("EC2::Instances::%s", string(instance.InstanceType)),
							Created:    *instance.LaunchTime,
							Meta: map[string]interface{}{
								"tags": ec2TagsToMap(instance.Tags),
							},
						}
					}
				}
			}
		}
		if instancesPager.Err() != nil {
			return fmt.Errorf("instances query failed: %w", instancesPager.Err())
		}

		volumesPager := aws_ec2.NewDescribeVolumesPaginator(client.DescribeVolumesRequest(&aws_ec2.DescribeVolumesInput{}))
		for volumesPager.Next(ctx) {
			page := volumesPager.CurrentPage()
			for _, volume := range page.Volumes {
				if volume.CreateTime.After(since) {
					results <- Resource{
						Platform:   PlatformAWS,
						Identifier: *volume.VolumeId,
						Location:   *volume.AvailabilityZone,
						Owner:      "-",
						Type:       fmt.Sprintf("EC2::Volumes::%s", string(volume.VolumeType)),
						Created:    *volume.CreateTime,
						Meta: map[string]interface{}{
							"tags":        ec2TagsToMap(volume.Tags),
							"attachments": volume.Attachments,
						},
					}
				}
			}
		}
		if volumesPager.Err() != nil {
			return fmt.Errorf("volumes query failed: %w", volumesPager.Err())
		}

		return nil
	},
	// fetch kubernetes clusters
	"EKS": func(ctx context.Context, results chan<- Resource, cfg aws.Config, since time.Time) error {
		// EKS is not available in these regions
		if cfg.Region == "us-west-1" || cfg.Region == "eu-south-1" {
			return nil
		}

		client := aws_eks.New(cfg)
		pager := aws_eks.NewListClustersPaginator(client.ListClustersRequest(&aws_eks.ListClustersInput{}))
		for pager.Next(ctx) {
			page := pager.CurrentPage()
			for _, clusterName := range page.Clusters {
				cluster, err := client.DescribeClusterRequest(&aws_eks.DescribeClusterInput{
					Name: aws.String(clusterName),
				}).Send(ctx)
				if err != nil {
					return fmt.Errorf("failed to fetch details for cluster '%s': %w", clusterName, err)
				}
				if cluster.Cluster.CreatedAt.After(since) {
					results <- Resource{
						Platform:   PlatformAWS,
						Identifier: *cluster.Cluster.Arn,
						Location:   cfg.Region,
						Owner:      "-",
						Type:       "EKS::Cluster",
						Created:    *cluster.Cluster.CreatedAt,
						Meta: map[string]interface{}{
							"tags": cluster.Cluster.Tags, // tags are already a map
						},
					}
				}
			}
		}
		return pager.Err()
	},
}

func collectAWSResources(ctx context.Context, since time.Time, verbose bool, tagsAllowlist map[string]string) ([]Resource, error) {
	logger := log.New(os.Stdout, "aws: ", log.LstdFlags|log.Lmsgprefix)
	if verbose {
		logger.Printf("collecting resources since %s", since)
	}

	cfg, err := aws_ext.LoadDefaultAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to init client: %w", err)
	}
	cfg.Region = "us-east-1" // set an arbitrary region to start

	results := make(chan Resource, resultsBuffer)
	errs := make(chan error)
	wait := &sync.WaitGroup{}

	// iterate over regions based on accessible EC2 regions
	regions, err := aws_ec2.New(cfg).DescribeRegionsRequest(&aws_ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	}).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	for _, region := range regions.Regions {
		if !hasPrefix(*region.RegionName, awsRegionPrefixes) {
			continue // skip this zone
		}
		cfg.Region = *region.RegionName
		if verbose {
			logger.Printf("querying region %s", cfg.Region)
		}

		// query configured resource in region
		for resourceID, fetchResource := range awsResources {
			wait.Add(1)
			go func(resourceID string, fetchResource AWSResourceFetchFunc, cfg aws.Config) {
				if err := fetchResource(ctx, results, cfg, since); err != nil {
					errs <- fmt.Errorf("region %s, resource %s: %w", cfg.Region, resourceID, err)
				}
				wait.Done()
			}(resourceID, fetchResource, cfg.Copy())
		}
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
				// allowlist resource if configured - all AWS tags should be converted to maps
				if tagsAllowlist != nil {
					if tags, ok := resource.Meta["tags"].(map[string]string); ok {
						if hasKeyValue(tags, tagsAllowlist) {
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

func ec2TagsToMap(tags []aws_ec2.Tag) map[string]string {
	m := map[string]string{}
	for _, tag := range tags {
		m[*tag.Key] = *tag.Value
	}
	return m
}
