package cloud_metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	imdsBaseURL   = "http://169.254.169.254"
	imdsTokenPath = "/latest/api/token"
)

// GetAWSEC2InstanceID returns the EC2 instance ID from IMDSv2
func GetAWSEC2InstanceID() (string, error) {
	return getAWSMetadata("/latest/meta-data/instance-id")
}

// GetAWSAutoScalingGroupName returns the Auto Scaling Group name from IMDSv2
// Note: Requires Instance Metadata Tags to be enabled on the instance
func GetAWSAutoScalingGroupName() (string, error) {
	return getAWSMetadata("/latest/meta-data/tags/instance/aws:autoscaling:groupName")
}

// GetAWSRegion returns the AWS region from IMDSv2
func GetAWSRegion() (string, error) {
	return getAWSMetadata("/latest/meta-data/placement/region")
}

// GetAWSAvailabilityZone returns the AWS availability zone from IMDSv2
func GetAWSAvailabilityZone() (string, error) {
	return getAWSMetadata("/latest/meta-data/placement/availability-zone")
}

// GetAWSAccountID returns the AWS account ID from IMDSv2
func GetAWSAccountID() (string, error) {
	data, err := getAWSMetadata("/latest/dynamic/instance-identity/document")
	if err != nil {
		return "", err
	}

	var doc struct {
		AccountId string `json:"accountId"`
	}
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		return "", fmt.Errorf("failed to parse instance identity document: %v", err)
	}

	return doc.AccountId, nil
}

// getIMDSv2Token obtains an IMDSv2 session token
func getIMDSv2Token(client *http.Client) string {
	tokenReq, err := http.NewRequest("PUT", imdsBaseURL+imdsTokenPath, nil)
	if err != nil {
		return ""
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "30")

	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return ""
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return ""
	}
	tokenBytes, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return ""
	}
	return string(tokenBytes)
}

// getAWSMetadata fetches a metadata value from the given path using IMDSv2
func getAWSMetadata(path string) (string, error) {
	client := http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	token := getIMDSv2Token(&client)

	req, err := http.NewRequest("GET", imdsBaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", path, err)
	}
	if token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get %s with HTTP status %d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response for %s: %v", path, err)
	}

	return string(body), nil
}
