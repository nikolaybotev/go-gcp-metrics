package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetInstanceIdOrHostname() (string, error) {
	// URL to access the instance metadata
	url := "http://169.254.169.254/latest/meta-data/instance-id"

	// Make the HTTP GET request to fetch the instance ID
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// Read the response body
			instanceID, err := io.ReadAll(resp.Body)
			if err == nil {
				return string(instanceID), nil
			}
		}
	}

	// If fetching the instance ID fails, get the hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %v", err)
	}
	return hostname, nil
}
