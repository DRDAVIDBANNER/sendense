package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/apache/cloudstack-go/cloudstack"
	"github.com/vexxhost/migratekit/internal/sha/database"
)

func main() {
	// Create database connection
	dbConfig := &database.MariaDBConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "oma_user",
		Password: "oma_password",
		Database: "migratekit_oma",
	}

	db, err := database.NewMariaDBConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get OSSEA configuration
	osseaRepo := database.NewOSSEAConfigRepository(db)
	configs, err := osseaRepo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get OSSEA configs: %v", err)
	}

	config := configs[0]
	cs := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, false)

	// Get Linux device information
	linuxDevices, err := getLinuxDeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get Linux device info: %v", err)
	}

	log.Printf("=== Linux Physical Devices ===")
	for _, dev := range linuxDevices {
		log.Printf("Device: %s, Size: %d bytes (%.2f GB), Virtio: %s",
			dev.Device, dev.SizeBytes, float64(dev.SizeBytes)/(1024*1024*1024), dev.VirtioController)
	}

	// Test CloudStack volumes
	testVolumes := []string{
		"e915ef05-ddf5-48d5-8352-a01300609717", // PGWINTESTBIOS-OLD
		"dd0e1f2f-1062-4c83-b011-cca29d21748b", // PGWINTESTBIOS-NEW
		"00ff0e64-8619-433e-a4df-1ecaaf804010", // PhilB Test
	}

	log.Printf("\n=== CloudStack → Linux Device Correlation Test ===")

	for _, volumeID := range testVolumes {
		log.Printf("\n--- Testing Volume: %s ---", volumeID)

		// Get CloudStack volume info
		params := cs.Volume.NewListVolumesParams()
		params.SetId(volumeID)
		response, err := cs.Volume.ListVolumes(params)
		if err != nil {
			log.Printf("❌ Error querying CloudStack: %v", err)
			continue
		}

		if response.Count == 0 || len(response.Volumes) == 0 {
			log.Printf("❌ Volume not found in CloudStack")
			continue
		}

		csVol := response.Volumes[0]
		log.Printf("CloudStack: %s, Size: %d bytes (%.2f GB), Deviceid: %d",
			csVol.Name, csVol.Size, float64(csVol.Size)/(1024*1024*1024), csVol.Deviceid)

		// Find matching Linux device by size (with tolerance)
		matched := false
		for _, linuxDev := range linuxDevices {
			sizeDiff := int64(linuxDev.SizeBytes) - csVol.Size
			if sizeDiff < 0 {
				sizeDiff = -sizeDiff
			}

			// Allow 5% tolerance for size differences
			tolerance := float64(csVol.Size) * 0.05

			if float64(sizeDiff) <= tolerance {
				log.Printf("✅ MATCH FOUND:")
				log.Printf("  CloudStack Size: %d bytes", csVol.Size)
				log.Printf("  Linux Size: %d bytes", linuxDev.SizeBytes)
				log.Printf("  Size Difference: %d bytes (%.2f%%)", sizeDiff,
					float64(sizeDiff)/float64(csVol.Size)*100)
				log.Printf("  Linux Device: %s", linuxDev.Device)
				log.Printf("  Virtio Controller: %s", linuxDev.VirtioController)
				matched = true
				break
			}
		}

		if !matched {
			log.Printf("❌ NO MATCHING LINUX DEVICE FOUND")
			log.Printf("  CloudStack Size: %d bytes (%.2f GB)", csVol.Size, float64(csVol.Size)/(1024*1024*1024))
		}
	}
}

type LinuxDeviceInfo struct {
	Device           string
	SizeBytes        int64
	VirtioController string
}

func getLinuxDeviceInfo() ([]LinuxDeviceInfo, error) {
	// Get list of vd* devices
	cmd := exec.Command("lsblk", "-b", "-n", "-o", "NAME,SIZE", "/dev/vda", "/dev/vdc", "/dev/vdd")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get device sizes: %w", err)
	}

	devices := []LinuxDeviceInfo{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "vd") && len(fields[0]) == 3 {
			sizeBytes, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				continue
			}

			// Get virtio controller for this device
			virtioCmd := exec.Command("readlink", fmt.Sprintf("/sys/block/%s/device", fields[0]))
			virtioOutput, err := virtioCmd.Output()
			if err != nil {
				continue
			}

			virtioPath := strings.TrimSpace(string(virtioOutput))
			virtioNum := strings.TrimPrefix(virtioPath, "../../../virtio")

			devices = append(devices, LinuxDeviceInfo{
				Device:           "/dev/" + fields[0],
				SizeBytes:        sizeBytes,
				VirtioController: "virtio" + virtioNum,
			})
		}
	}

	return devices, nil
}
