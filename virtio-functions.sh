#!/bin/bash
# virtio-functions.sh - VirtIO Function Library
# Source this file to access inject_virtio_drivers() function
#
# Usage: 
#   source virtio-functions.sh
#   inject_virtio_drivers "/dev/vdc" "job-20250122-143022"

# Function wrapper for the main script
inject_virtio_drivers() {
    local block_device="$1"
    local job_id="$2"
    
    # Call the main script
    sudo /opt/migratekit/bin/inject-virtio-drivers.sh "$block_device" "$job_id"
    return $?
}

# Export the function for use in subshells
export -f inject_virtio_drivers

echo "VirtIO driver injection function loaded."
echo "Usage: inject_virtio_drivers <block_device> <job_id>"
echo "Example: inject_virtio_drivers \"/dev/vdc\" \"job-20250122-143022\""
