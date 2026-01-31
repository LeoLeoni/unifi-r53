package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"log"
)

func main() {
	// Using the SDK's default configuration, load additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Using the Config value, create the DynamoDB client
	svc := route53.NewFromConfig(cfg)
	res, err := svc.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{})

	if err != nil {
		log.Fatalf("Failed to fetch hosted zones, %v", err)
	}
	for _, id := range *res.HostedZoneId {
		fmt.Println(id)
	}
}
