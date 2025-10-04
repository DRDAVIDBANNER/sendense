package main

import (
	"fmt"
	"strings"
)

func constructByIDPath(volumeID string) string {
	cleanUUID := strings.ReplaceAll(volumeID, "-", "")
	shortID := cleanUUID[:20]
	return fmt.Sprintf("/dev/disk/by-id/virtio-%s", shortID)
}

func main() {
	fmt.Println("🧪 Testing by-id path construction")
	
	// pgtest1 volumes
	vol1 := "3106013a-e175-423e-a090-20cf5551389b"
	vol2 := "b3bb9310-1b59-4f62-97e8-cefffdfe3804"
	
	path1 := constructByIDPath(vol1)
	path2 := constructByIDPath(vol2)
	
	fmt.Printf("Volume 1: %s\n", vol1)
	fmt.Printf("by-id:    %s\n", path1)
	fmt.Printf("Expected: /dev/disk/by-id/virtio-3106013ae175423ea090\n")
	fmt.Printf("Match:    %t\n\n", path1 == "/dev/disk/by-id/virtio-3106013ae175423ea090")
	
	fmt.Printf("Volume 2: %s\n", vol2)
	fmt.Printf("by-id:    %s\n", path2)
	fmt.Printf("Expected: /dev/disk/by-id/virtio-b3bb93101b594f6297e8\n")
	fmt.Printf("Match:    %t\n", path2 == "/dev/disk/by-id/virtio-b3bb93101b594f6297e8")
}
