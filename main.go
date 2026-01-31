package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

const HOST = "192.168.1.1"

type UniFiListResponse[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T
}
type Site struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}
type Device struct {
	ID                string   `json:"id"`
	MacAddress        string   `json:"macAddress"`
	IPAddress         string   `json:"ipAddress"`
	Name              string   `json:"name"`
	Model             string   `json:"model"`
	State             string   `json:"state"`
	Supported         bool     `json:"supported"`
	FirmwareVersion   string   `json:"firmwareVersion"`
	FirmwareUpdatable bool     `json:"firmwareUpdatable"`
	Features          []string `json:"features"`
	Interfaces        []string `json:"interfaces"`
}

func fetchHostedZones(client *route53.Client) []types.HostedZone {
	res, err := client.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{})
	if err != nil {
		return []types.HostedZone{}
	}
	return res.HostedZones
}

func requestApiData[T any](path string) T {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	key := os.Getenv("UNIFI_API_KEY")
	url := fmt.Sprintf("https://%s/proxy/network/integration/%s", HOST, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Unable to form request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-KEY", key)
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalf("UniFi api response failed %v", err)
	}

	defer resp.Body.Close()

	var data T
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Can not unmarshal JSON: %v", err)
	}
	return data
}

// For a given site returns the first matching IP address
func fetchIP() netip.Addr {
	sites := requestApiData[UniFiListResponse[Site]]("v1/sites")
	siteID := sites.Data[0].ID

	devices := requestApiData[UniFiListResponse[Device]](fmt.Sprintf("v1/sites/%s/devices", siteID))
	// TODO: If this program runs while the Device.State is still OFFLINE, IPAddress
	// won't have been set yet which is VERY likely after a power outage. The best
	// approach would be to find just the ID of the Cloud Gateway, recursively retry
	// fetching details for just that device until "ONLINE", THEN grab the Public IP
	for _, device := range devices.Data {
		ip, err := netip.ParseAddr(device.IPAddress)
		if err == nil && !ip.IsPrivate() {
			return ip
		}
	}
	return netip.IPv4Unspecified()
}

func main() {
	// Using the SDK's default configuration, load additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := route53.NewFromConfig(cfg)

	publicIP := fetchIP()
	fmt.Println("Public IP: ", publicIP)
	_ = fetchHostedZones(client)
}
