package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func fetchHostedZones(client *route53.Client) []types.HostedZone {
	res, err := client.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{})
	if err != nil {
		return []types.HostedZone{}
	}
	return res.HostedZones
}

func updateRecords(
	client *route53.Client,
	zone types.HostedZone,
	ip netip.Addr,
) (*route53.ChangeResourceRecordSetsOutput, error) {
	request := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zone.Id,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: zone.Name,
						Type: types.RRTypeA,
						TTL:  aws.Int64(300),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(ip.String())},
						},
					},
				},
			},
			Comment: aws.String("Auto updated from reverse DNS bot"),
		},
	}
	return client.ChangeResourceRecordSets(context.TODO(), request)
}

func main() {
	publicIP := fetchIP()
	fmt.Println("Public IP: ", publicIP)

	// Using the SDK's default configuration, load additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("Failed to load AWS SDK config: %v", err)
	}
	client := route53.NewFromConfig(cfg)

	hostedZones := fetchHostedZones(client)
	// TODO: args with a list of hostedZones to update so we don't just update all of them
	for _, zone := range hostedZones {
		// TODO: something with the response
		updateRecords(client, zone, publicIP)
	}
}
