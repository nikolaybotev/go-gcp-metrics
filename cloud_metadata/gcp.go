package cloud_metadata

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	gcpMetadataBaseURL = "http://metadata.google.internal"
)

// GetGCPInstanceID returns the GCE instance ID from the metadata server
func GetGCPInstanceID() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/instance/id")
}

// GetGCPInstanceName returns the GCE instance name from the metadata server
func GetGCPInstanceName() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/instance/name")
}

// GetGCPProjectID returns the GCP project ID from the metadata server
func GetGCPProjectID() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/project/project-id")
}

// GetGCPZone returns the GCE zone from the metadata server
func GetGCPZone() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/instance/zone")
}

// GetGCPRegion returns the GCE region from the metadata server
func GetGCPRegion() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/instance/region")
}

// GetGCPInstanceGroupName returns the managed instance group name from the metadata server
func GetGCPInstanceGroupName() (string, error) {
	return GetGCPMetadata("/computeMetadata/v1/instance/attributes/created-by")
}

// GetGCPMetadata fetches a metadata value from the given path using the GCP metadata server
func GetGCPMetadata(path string) (string, error) {
	client := http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	req, err := http.NewRequest("GET", gcpMetadataBaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", path, err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

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
