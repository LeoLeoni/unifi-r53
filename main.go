package main

import (
	"context"
	"log"
	"net/netip"
	"os"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func fetchHostedZones(client *route53.Client) []types.HostedZone {
	res, err := client.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{})
	if err != nil {
		log.Fatalf("Failed to list hosted zones: %v", err)
	}
	return res.HostedZones
}

func updateRecords(client *route53.Client, zone types.HostedZone, ip netip.Addr) error {
	_, err := client.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
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
	})
	return err
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <zone>...", os.Args[0])
	}
	// NOTE: Must end in trailing dot i.e. mydomain.com.
	zones := os.Args[1:]

	publicIP := fetchIP()
	log.Printf("Public IP: %s", publicIP)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("Failed to load AWS SDK config: %v", err)
	}
	client := route53.NewFromConfig(cfg)

	for _, zone := range fetchHostedZones(client) {
		// ignore hostedzones on the acct that weren't passed as args
		if !slices.Contains(zones, *zone.Name) {
			continue
		}
		if err := updateRecords(client, zone, publicIP); err != nil {
			log.Fatalf("Failed to update %s: %v", *zone.Name, err)
		}
		log.Printf("Upserted %s to %s", *zone.Name, publicIP)
	}
}
