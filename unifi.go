package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"
	"time"
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

// With exponential wait this would be 64s
const maxRetries = 6

func fetchDeviceDetails(siteID string, deviceID string, retryCount int) netip.Addr {
	path := fmt.Sprintf("v1/sites/%s/devices/%s", siteID, deviceID)
	device := requestApiData[Device](path)

	if device.State == "ONLINE" {
		ip, err := netip.ParseAddr(device.IPAddress)

		if err == nil && !ip.IsPrivate() {
			return ip
		}
	} else if retryCount <= maxRetries {
		time.Sleep(time.Duration(1<<retryCount) * time.Second)

		return fetchDeviceDetails(siteID, deviceID, retryCount+1)
	}

	return netip.IPv4Unspecified()
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
