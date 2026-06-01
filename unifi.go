package main

import (
	"cmp"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"
)

var (
	host = cmp.Or(os.Getenv("UNIFI_HOST"), "192.168.1.1")
	key  = os.Getenv("UNIFI_API_KEY")
)

type UniFiListResponse[T any] struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
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

func requestApiData[T any](path string) T {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://%s/proxy/network/integration/%s", host, path)
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

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("UniFi api error status code %s", path)
	}

	var data T
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Can not unmarshal JSON: %v", err)
	}
	return data
}

// fetchIP returns the first non-private IPv4 reported by any device on the
// first site. Fatal on miss — the external scheduler retries on the next tick.
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
		if err == nil && !ip.IsPrivate() && !ip.IsUnspecified() {
			return ip
		}
	}
	// Just fail fast in this case. This is going to be a cron type thing so retry doesn't matter
	log.Fatalf("No public IP among %d devices on site %s", len(devices.Data), siteID)
	return netip.IPv4Unspecified()
}
