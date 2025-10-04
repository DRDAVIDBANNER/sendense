#!/bin/bash
# inject-virtio-drivers.sh - VirtIO Driver Injection for Windows VMs
# 
# Description: Injects VirtIO drivers into Windows block devices using virt-v2v-in-place
# Author: MigrateKit OSSEA Project
# Created: 2025-01-22
# Version: 1.0.0
#
# Usage: inject_virtio_drivers <block_device> <job_id>
# Example: inject_virtio_drivers "/dev/vdc" "job-20250122-143022"

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Global configuration
readonly SCRIPT_NAME="$(basename "$0")"
readonly LOG_DIR="/var/log/migratekit"
readonly VIRTIO_WIN_ISO="/usr/share/virtio-win/virtio-win.iso"

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE" >&2
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Script failed with exit code $exit_code"
        log_error "Check log file for details: $LOG_FILE"
    fi
    exit $exit_code
}

# Set trap for cleanup
trap cleanup EXIT

# Input validation function
validate_inputs() {
    local block_device="$1"
    local job_id="$2"
    
    # Validate block device
    if [[ -z "$block_device" ]]; then
        log_error "Block device parameter is required"
        return 1
    fi
    
    if [[ ! -b "$block_device" ]]; then
        log_error "Block device '$block_device' does not exist or is not a block device"
        return 1
    fi
    
    # Validate job ID
    if [[ -z "$job_id" ]]; then
        log_error "Job ID parameter is required"
        return 1
    fi
    
    # Job ID should be alphanumeric with hyphens and underscores
    if [[ ! "$job_id" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log_error "Job ID '$job_id' contains invalid characters. Use only alphanumeric, hyphens, and underscores"
        return 1
    fi
    
    log_info "Input validation passed"
    return 0
}

# Prerequisites check function
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        return 1
    fi
    
    # Check if virt-v2v-in-place is available
    if ! command -v virt-v2v-in-place &> /dev/null; then
        log_error "virt-v2v-in-place command not found. Install with: sudo apt install virt-v2v"
        return 1
    fi
    
    # Check if VirtIO drivers ISO exists
    if [[ ! -f "$VIRTIO_WIN_ISO" ]]; then
        log_error "VirtIO drivers ISO not found at $VIRTIO_WIN_ISO"
        log_error "Download with: sudo wget -O $VIRTIO_WIN_ISO https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso"
        return 1
    fi
    
    # Check ISO file size (should be > 400MB)
    local iso_size
    iso_size=$(stat -c%s "$VIRTIO_WIN_ISO" 2>/dev/null || echo "0")
    if [[ $iso_size -lt 419430400 ]]; then  # 400MB in bytes
        log_error "VirtIO ISO file appears to be incomplete (size: $iso_size bytes)"
        return 1
    fi
    
    log_success "All prerequisites met"
    return 0
}

# Block device safety checks
check_block_device_safety() {
    local block_device="$1"
    
    log_info "Performing safety checks on block device $block_device"
    
    # Check if device is mounted
    if mount | grep -q "^$block_device"; then
        log_error "Block device $block_device is currently mounted. Unmount before proceeding"
        log_info "Mounted filesystems:"
        mount | grep "^$block_device" | tee -a "$LOG_FILE"
        return 1
    fi
    
    # Check device size
    local device_size
    device_size=$(blockdev --getsize64 "$block_device" 2>/dev/null || echo "0")
    if [[ $device_size -eq 0 ]]; then
        log_error "Could not determine size of block device $block_device"
        return 1
    fi
    
    # Convert to GB for display
    local size_gb=$((device_size / 1073741824))
    log_info "Block device size: $size_gb GB ($device_size bytes)"
    
    # Warn if device is very large (>500GB) 
    if [[ $device_size -gt 537109504000 ]]; then
        log_warning "Large block device detected ($size_gb GB). Driver injection may take significant time"
    fi
    
    # Basic filesystem detection (non-destructive)
    log_info "Attempting to identify filesystem types..."
    if command -v file &> /dev/null; then
        file -s "$block_device" | tee -a "$LOG_FILE" || true
    fi
    
    log_success "Block device safety checks passed"
    return 0
}

# Main VirtIO injection function
inject_virtio_drivers() {
    local block_device="$1"
    local job_id="$2"
    
    log_info "🚀 Starting VirtIO driver injection for job $job_id"
    log_info "📀 Target block device: $block_device"
    log_info "🔧 VirtIO drivers: $VIRTIO_WIN_ISO"
    log_info "📁 Log file: $LOG_FILE"
    
    # Set VIRTIO_WIN environment variable for virt-v2v
    export VIRTIO_WIN="$VIRTIO_WIN_ISO"
    log_info "Set VIRTIO_WIN environment variable: $VIRTIO_WIN"
    
    # Create a timestamp for operation tracking
    local start_time=$(date +%s)
    local start_timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    log_info "⏱️  Operation started at: $start_timestamp"
    log_info "🔄 Executing virt-v2v-in-place with MigrateKit's exact command format..."
    log_info "🛡️  Using MigrateKit's direct backend (no libvirt/qemu dependencies)"
    
    # Execute virt-v2v-in-place with comprehensive logging
    local virt_v2v_output
    local exit_code
    
    # Run virt-v2v-in-place and capture output
    # Use MigrateKit's exact libguestfs settings for CloudStack appliance environment
    set +e  # Temporarily disable exit on error to handle return code
    
    # Set libguestfs backend exactly as MigrateKit does (NO libvirt/qemu dependencies)
    export LIBGUESTFS_BACKEND="direct"
    
    virt_v2v_output=$(virt-v2v-in-place \
        -v \
        -x \
        -i disk \
        "$block_device" 2>&1)
    exit_code=$?
    set -e  # Re-enable exit on error
    
    # Log the complete output
    echo "=== VIRT-V2V-IN-PLACE OUTPUT ===" >> "$LOG_FILE"
    echo "$virt_v2v_output" >> "$LOG_FILE"
    echo "=== END VIRT-V2V-IN-PLACE OUTPUT ===" >> "$LOG_FILE"
    
    # Calculate operation duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local end_timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Check exit status
    if [[ $exit_code -ne 0 ]]; then
        log_error "❌ virt-v2v-in-place failed with exit code $exit_code"
        log_error "⏱️  Operation failed after $duration seconds"
        log_error "📋 Check log file for detailed error information: $LOG_FILE"
        log_error "🔍 Last 10 lines of virt-v2v output:"
        echo "$virt_v2v_output" | tail -10 | while read -r line; do
            log_error "   $line"
        done
        return 1
    fi
    
    log_success "✅ VirtIO driver injection completed successfully!"
    log_success "⏱️  Operation completed in $duration seconds"
    log_success "📁 Complete log available at: $LOG_FILE"
    log_success "🎯 Job $job_id ready for failover with VirtIO drivers"
    
    # Log operation summary
    log_info "📊 Operation Summary:"
    log_info "   - Job ID: $job_id"
    log_info "   - Block Device: $block_device"
    log_info "   - Start Time: $start_timestamp"
    log_info "   - End Time: $end_timestamp"
    log_info "   - Duration: $duration seconds"
    log_info "   - VirtIO ISO: $(basename "$VIRTIO_WIN_ISO")"
    log_info "   - Exit Code: $exit_code (success)"
    
    return 0
}

# Usage function
usage() {
    cat << EOF
Usage: $SCRIPT_NAME <block_device> <job_id>

Description:
    Injects VirtIO drivers into Windows block devices using virt-v2v-in-place.
    This prepares Windows VMs for optimal performance in KVM/CloudStack environments.

Parameters:
    block_device    Path to the Windows block device (e.g., /dev/vdc)
    job_id         Unique identifier for this operation (alphanumeric, hyphens, underscores)

Examples:
    $SCRIPT_NAME /dev/vdc job-20250122-143022
    $SCRIPT_NAME /dev/vdb migration-vm-001
    
Requirements:
    - Must run as root (use sudo)
    - virt-v2v-in-place must be installed
    - VirtIO drivers ISO must be available at $VIRTIO_WIN_ISO
    - Target block device must not be mounted

Logs:
    Operation logs are written to: $LOG_DIR/virtv2v-<job_id>.log

For more information, see the MigrateKit OSSEA documentation.
EOF
}

# Main execution function
main() {
    # Check for help flag
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        usage
        exit 0
    fi
    
    # Check parameter count
    if [[ $# -ne 2 ]]; then
        log_error "Invalid number of parameters"
        echo ""
        usage
        exit 1
    fi
    
    local block_device="$1"
    local job_id="$2"
    
    # Set up log file
    LOG_FILE="$LOG_DIR/virtv2v-${job_id}.log"
    
    # Ensure log directory exists
    mkdir -p "$LOG_DIR"
    
    # Start logging
    log_info "=================================================="
    log_info "🚀 VirtIO Driver Injection Starting"
    log_info "📝 Script: $SCRIPT_NAME"
    log_info "🔗 Job ID: $job_id"
    log_info "📀 Block Device: $block_device"
    log_info "=================================================="
    
    # Execute all checks and operations
    validate_inputs "$block_device" "$job_id" || exit 1
    check_prerequisites || exit 1
    check_block_device_safety "$block_device" || exit 1
    inject_virtio_drivers "$block_device" "$job_id" || exit 1
    
    log_success "=================================================="
    log_success "🎉 VirtIO Driver Injection Completed Successfully"
    log_success "🔗 Job ID: $job_id"
    log_success "📁 Log File: $LOG_FILE"
    log_success "=================================================="
    
    exit 0
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
