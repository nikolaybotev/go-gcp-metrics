package cloud_metadata

import "os"

func GetInstanceName() string {
	instanceID, err := GetAWSEC2InstanceID()
	if err == nil && instanceID != "" {
		return instanceID
	}

	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}

	return "unknown"
}
