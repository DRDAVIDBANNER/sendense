package device

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// getDeviceSize reads the device size from /sys/block/*/size
func getDeviceSize(deviceName string) (int64, error) {
	sizePath := fmt.Sprintf("/sys/block/%s/size", deviceName)
	data, err := ioutil.ReadFile(sizePath)
	if err != nil {
		return 0, err
	}

	// Size is in 512-byte sectors
	sectors, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return sectors * 512, nil
}

// getVirtioController reads virtio controller information
func getVirtioController(deviceName string) (string, error) {
	// Try to read the device symlink to get virtio controller info
	deviceLink := fmt.Sprintf("/sys/block/%s/device", deviceName)
	target, err := os.Readlink(deviceLink)
	if err != nil {
		return "", err
	}

	// Extract virtio controller identifier from the symlink target
	// Example: ../../../devices/pci0000:00/0000:00:06.0/virtio2/block/vdb
	parts := strings.Split(target, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "virtio") {
			return part, nil
		}
	}

	return "", fmt.Errorf("no virtio controller found in device path: %s", target)
}
