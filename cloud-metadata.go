package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func GetInstanceName() (string, error) {
	// URL to access the AWS instance ID
	url := "http://169.254.169.254/latest/meta-data/instance-id"

	// Make the HTTP GET request to fetch the instance ID
	client := http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Get(url)
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

	// If fetching the AWS instance ID fails, get the hostname (matches the instance name on Google Cloud)
	hostname, err := os.Hostname()
	if err != nil {
		return "undefined", fmt.Errorf("failed to get hostname: %v", err)
	}
	return hostname, nil
}
