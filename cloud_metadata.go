package gcpmetrics

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func GetInstanceName() string {
	instanceID, err := getAWSEC2InstanceID()
	if err == nil && instanceID != "" {
		return instanceID
	}

	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}

	return "unknown"
}

func getAWSEC2InstanceID() (string, error) {
	// Create a new HTTP client
	client := http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	// Create IMDSv2 token request
	tokenReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %v", err)
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "30")

	// Send IMDSv2 token request
	tokenResp, err := client.Do(tokenReq)
	var token string
	if err == nil {
		defer tokenResp.Body.Close()
		if tokenResp.StatusCode == http.StatusOK {
			tokenBytes, err := io.ReadAll(tokenResp.Body)
			if err == nil {
				token = string(tokenBytes)
			}
		}
	}

	// Create instance ID request
	req, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/instance-id", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create instance-id request: %v", err)
	}
	if token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", token)
	}

	// Send instance ID request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS EC2 instance ID: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get AWS EC2 instance ID with HTTP status %d", resp.StatusCode)
	}
	instanceID, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read instance ID response: %v", err)
	}

	return string(instanceID), nil
}
