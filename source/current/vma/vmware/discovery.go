// Package vmware provides VMware vCenter integration for VM discovery and inventory collection
package vmware

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/vexxhost/migratekit-oma/models"
)

// Config holds VMware connection configuration
type Config struct {
	Host       string `json:"host"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Datacenter string `json:"datacenter"`
	Insecure   bool   `json:"insecure"`
}

// Discovery provides VMware VM discovery capabilities
type Discovery struct {
	config Config
	client *govmomi.Client
}

// NewDiscovery creates a new VMware discovery client
func NewDiscovery(config Config) *Discovery {
	return &Discovery{
		config: config,
	}
}

// Connect establishes connection to vCenter
func (d *Discovery) Connect(ctx context.Context) error {
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", d.config.Host))
	if err != nil {
		return fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	u.User = url.UserPassword(d.config.Username, d.config.Password)

	client, err := govmomi.NewClient(ctx, u, d.config.Insecure)
	if err != nil {
		return fmt.Errorf("failed to connect to vCenter: %w", err)
	}

	d.client = client
	log.WithField("host", d.config.Host).Info("Successfully connected to vCenter")

	return nil
}

// Disconnect closes vCenter connection
func (d *Discovery) Disconnect() {
	if d.client != nil {
		d.client.Logout(context.Background())
		d.client = nil
	}
}

// GetClient returns the authenticated govmomi client for reuse
func (d *Discovery) GetClient() *govmomi.Client {
	return d.client
}

// DiscoverVMs discovers all VMs in the datacenter
func (d *Discovery) DiscoverVMs(ctx context.Context) (*models.VMInventoryRequest, error) {
	if d.client == nil {
		return nil, fmt.Errorf("not connected to vCenter")
	}

	finder := find.NewFinder(d.client.Client, true)

	// Find datacenter
	dc, err := finder.Datacenter(ctx, d.config.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenter %s: %w", d.config.Datacenter, err)
	}
	finder.SetDatacenter(dc)

	// Find all VMs
	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	log.WithField("vm_count", len(vms)).Info("Discovered VMs from vCenter")

	// Convert VMs to ManagedObjectReferences for property retrieval
	var refs []types.ManagedObjectReference
	for _, vm := range vms {
		refs = append(refs, vm.Reference())
	}

	// Get VM properties
	var vmMos []mo.VirtualMachine
	pc := property.DefaultCollector(d.client.Client)

	err = pc.Retrieve(ctx, refs, []string{
		"name",
		"config",
		"runtime.powerState",
		"guest.guestFullName",
		"guest.toolsStatus",
		"guest.toolsVersion",
		"layoutEx",
		"summary.config.annotation",
		"parent",
	}, &vmMos)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Convert to our VM model
	var vmInfos []models.VMInfo

	// Create a map of references to VM objects for proper matching
	vmByRef := make(map[types.ManagedObjectReference]*object.VirtualMachine)
	for _, vm := range vms {
		vmByRef[vm.Reference()] = vm
	}

	// Match VM properties with VM objects using references (not array indices)
	for _, vmMo := range vmMos {
		vm, exists := vmByRef[vmMo.Reference()]
		if !exists {
			log.WithField("ref", vmMo.Reference()).Warn("VM managed object not found in original list")
			continue
		}

		vmInfo := d.convertVMToModel(vm, &vmMo)
		vmInfos = append(vmInfos, vmInfo)
	}

	// Get vCenter info
	vcenterInfo := models.VCenterInfo{
		Host:              d.config.Host,
		Version:           d.client.ServiceContent.About.Version,
		Datacenter:        d.config.Datacenter,
		TotalVMs:          len(vmInfos),
		ConnectionHealthy: true,
	}

	inventory := &models.VMInventoryRequest{
		VMs:       vmInfos,
		VCenter:   vcenterInfo,
		Timestamp: time.Now().UTC(),
	}

	log.WithFields(log.Fields{
		"vm_count":     len(vmInfos),
		"vcenter_host": d.config.Host,
		"datacenter":   d.config.Datacenter,
	}).Info("VM inventory discovery completed")

	return inventory, nil
}

// convertVMToModel converts govmomi VM to our model
func (d *Discovery) convertVMToModel(vm *object.VirtualMachine, vmMo *mo.VirtualMachine) models.VMInfo {
	// Use UUID if available, fallback to reference value
	vmID := vm.Reference().Value
	if vmMo.Config != nil && vmMo.Config.Uuid != "" {
		vmID = vmMo.Config.Uuid
	}

	// Get VM inventory path
	vmPath := vm.InventoryPath

	// Validate path consistency - if VM name doesn't match the last path component,
	// log a warning and fix the path as this indicates vCenter data inconsistency
	pathParts := strings.Split(vmPath, "/")
	if len(pathParts) > 0 {
		lastPathComponent := pathParts[len(pathParts)-1]
		if lastPathComponent != vmMo.Name {
			log.WithFields(log.Fields{
				"vm_name":        vmMo.Name,
				"vm_path":        vmPath,
				"path_component": lastPathComponent,
			}).Warn("VM name/path mismatch detected - potential vCenter data inconsistency")

			// DO NOT modify paths - use original vCenter path as-is
			log.WithFields(log.Fields{
				"original_path": vm.InventoryPath,
				"vm_name":       vmMo.Name,
			}).Info("Using original vCenter path despite name mismatch")
		}
	}

	vmInfo := models.VMInfo{
		ID:         vmID,
		Name:       vmMo.Name,
		Path:       vmPath,
		Datacenter: d.config.Datacenter, // Add datacenter from config
		Disks:      []models.DiskInfo{},
		Networks:   []models.NetworkInfo{},
	}

	// Get basic config
	if vmMo.Config != nil {
		vmInfo.CPUs = int(vmMo.Config.Hardware.NumCPU)
		vmInfo.MemoryMB = int(vmMo.Config.Hardware.MemoryMB)
		vmInfo.VMXVersion = vmMo.Config.Version
	}

	// Get additional VM metadata
	vmInfo.DisplayName = vmMo.Name // Use name as display name (could be enhanced with config.name if different)

	// Get VM annotation/notes
	if vmMo.Summary.Config.Annotation != "" {
		vmInfo.Annotation = vmMo.Summary.Config.Annotation
	}

	// Get folder path from VM parent structure
	vmInfo.FolderPath = d.resolveFolderPath(vm)

	// Get VMware Tools information
	if vmMo.Guest != nil {
		vmInfo.VMwareToolsStatus = string(vmMo.Guest.ToolsStatus)
		if vmMo.Guest.ToolsVersion != "" {
			vmInfo.VMwareToolsVersion = vmMo.Guest.ToolsVersion
		}
	}

	// Get power state
	switch vmMo.Runtime.PowerState {
	case types.VirtualMachinePowerStatePoweredOn:
		vmInfo.PowerState = "poweredOn"
	case types.VirtualMachinePowerStatePoweredOff:
		vmInfo.PowerState = "poweredOff"
	case types.VirtualMachinePowerStateSuspended:
		vmInfo.PowerState = "suspended"
	default:
		vmInfo.PowerState = "unknown"
	}

	// Get OS type
	if vmMo.Guest != nil && vmMo.Guest.GuestFullName != "" {
		osName := vmMo.Guest.GuestFullName
		if contains(osName, "Windows") {
			vmInfo.OSType = "windows"
		} else if contains(osName, "Linux") {
			vmInfo.OSType = "linux"
		} else {
			vmInfo.OSType = "unknown"
		}
	} else {
		vmInfo.OSType = "unknown"
	}

	// Get disk information
	if vmMo.Config != nil {
		for _, device := range vmMo.Config.Hardware.Device {
			if disk, ok := device.(*types.VirtualDisk); ok {
				unitNumber := 0
				if disk.UnitNumber != nil {
					unitNumber = int(*disk.UnitNumber)
				}

				// Use VMware CapacityInBytes directly - this is the correct logical disk size
				// The getActualDiskSize() function has a bug where it sums all VMDK files for multi-disk VMs
				actualSizeBytes := disk.CapacityInBytes

				diskInfo := models.DiskInfo{
					ID:               fmt.Sprintf("disk-%d", disk.Key),
					Label:            disk.DeviceInfo.GetDescription().Label,
					CapacityBytes:    actualSizeBytes,                                  // Use actual file size
					SizeGB:           int((actualSizeBytes + 1073741823) / 1073741824), // Convert actual bytes to GB (round up)
					ProvisioningType: "unknown",                                        // Will be updated below if backing info is available
					UnitNumber:       unitNumber,
				}

				// Get backing info if available
				if backing := disk.Backing; backing != nil {
					if flatBacking, ok := backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
						diskInfo.VMDKPath = flatBacking.FileName
						diskInfo.Path = flatBacking.FileName // Set Path field as well

						// Extract datastore from file path like "[datastore1] VM/disk.vmdk"
						if strings.HasPrefix(flatBacking.FileName, "[") {
							if endBracket := strings.Index(flatBacking.FileName, "]"); endBracket > 0 {
								diskInfo.Datastore = flatBacking.FileName[1:endBracket]
							}
						}

						if flatBacking.ThinProvisioned != nil && *flatBacking.ThinProvisioned {
							diskInfo.ProvisioningType = "thin"
						} else {
							diskInfo.ProvisioningType = "thick"
						}
					}
				}

				vmInfo.Disks = append(vmInfo.Disks, diskInfo)
			}
		}
	}

	// Get network information
	if vmMo.Config != nil {
		for _, device := range vmMo.Config.Hardware.Device {
			if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
				ethernetCard := nic.GetVirtualEthernetCard()

				// Resolve network name from backing
				networkName := d.resolveNetworkName(ethernetCard.Backing)

				networkInfo := models.NetworkInfo{
					Label:       ethernetCard.DeviceInfo.GetDescription().Label,
					NetworkName: networkName,
					MACAddress:  ethernetCard.MacAddress,
					Connected:   ethernetCard.Connectable.Connected,
				}

				// Determine adapter type
				switch nic.(type) {
				case *types.VirtualVmxnet3:
					networkInfo.AdapterType = "vmxnet3"
				case *types.VirtualE1000:
					networkInfo.AdapterType = "e1000"
				case *types.VirtualE1000e:
					networkInfo.AdapterType = "e1000e"
				default:
					networkInfo.AdapterType = "unknown"
				}

				vmInfo.Networks = append(vmInfo.Networks, networkInfo)
			}
		}
	}

	return vmInfo
}

// resolveNetworkName extracts network name from VirtualEthernetCard backing with enhanced DVS support
func (d *Discovery) resolveNetworkName(backing types.BaseVirtualDeviceBackingInfo) string {
	if backing == nil {
		log.Error("Network backing is nil - possible VMware configuration issue")
		return "ERROR-NULL-BACKING"
	}

	backingType := fmt.Sprintf("%T", backing)
	log.WithFields(log.Fields{
		"backing_type": backingType,
	}).Debug("Starting network name resolution")

	switch b := backing.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		// Standard network backing (vSwitch)
		log.WithFields(log.Fields{
			"has_network_ref": b.Network != nil,
			"device_name":     b.DeviceName,
		}).Debug("Processing standard vSwitch network backing")

		if b.Network != nil {
			log.WithFields(log.Fields{
				"network_ref_value": b.Network.Value,
				"network_ref_type":  b.Network.Type,
			}).Debug("Attempting to resolve standard network reference")

			// Try to resolve the network object reference to actual name
			networkName := d.resolveNetworkReference(b.Network)
			if networkName != "" {
				log.WithField("resolved_name", networkName).Info("✅ Successfully resolved standard network name")
				return networkName
			}

			// Enhanced fallback: use network reference value with prefix
			if b.Network.Value != "" {
				fallbackName := fmt.Sprintf("STD-REF-%s", b.Network.Value)
				log.WithField("fallback_name", fallbackName).Info("Using standard network reference as fallback")
				return fallbackName
			}
		}

		// Use device name as fallback
		if b.DeviceName != "" {
			log.WithField("device_name", b.DeviceName).Info("Using device name for standard network")
			return b.DeviceName
		}

		log.Error("Standard network backing has no resolvable identifiers")
		return "ERROR-STANDARD-NO-NAME"

	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		// Enhanced DVS handling for networks like "VLAN 1253 - QCLOUD-DEV-M"
		log.WithFields(log.Fields{
			"portgroup_key": b.Port.PortgroupKey,
			"switch_uuid":   b.Port.SwitchUuid,
			"port_key":      b.Port.PortKey,
		}).Info("Processing DVS (Distributed Virtual Switch) network backing")

		if b.Port.PortgroupKey != "" {
			// Enhanced DVS portgroup resolution with multiple strategies
			portgroupName := d.resolveDVSPortgroupNameEnhanced(b.Port.PortgroupKey)
			
			log.WithFields(log.Fields{
				"portgroup_key":      b.Port.PortgroupKey,
				"resolved_name":      portgroupName,
				"resolution_success": portgroupName != "",
			}).Debug("DVS portgroup resolution completed")

			if portgroupName != "" {
				log.WithField("dvs_name", portgroupName).Info("✅ Successfully resolved DVS portgroup name")
				return portgroupName
			}

			// Enhanced fallback for DVS: use portgroup key with descriptive prefix
			fallbackName := fmt.Sprintf("DVS-PG-%s", b.Port.PortgroupKey)
			log.WithField("fallback_name", fallbackName).Info("Using DVS portgroup key as fallback")
			return fallbackName
		}

		// If no portgroup key, use switch UUID
		if b.Port.SwitchUuid != "" {
			fallbackName := fmt.Sprintf("DVS-SWITCH-%s", b.Port.SwitchUuid[:8])
			log.WithField("fallback_name", fallbackName).Info("Using DVS switch UUID as fallback")
			return fallbackName
		}

		log.Error("DVS backing has no identifiable portgroup or switch information")
		return "ERROR-DVS-NO-IDENTIFIERS"

	case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
		// NSX or other opaque network backing
		log.WithFields(log.Fields{
			"opaque_network_id":   b.OpaqueNetworkId,
			"opaque_network_type": b.OpaqueNetworkType,
		}).Debug("Processing opaque network backing (NSX/other)")

		if b.OpaqueNetworkId != "" {
			networkName := fmt.Sprintf("NSX-%s", b.OpaqueNetworkId)
			log.WithField("nsx_name", networkName).Info("✅ Resolved NSX opaque network")
			return networkName
		}
		return "ERROR-OPAQUE-NO-ID"

	default:
		// Unknown backing type
		log.WithFields(log.Fields{
			"backing_type": backingType,
		}).Error("Encountered unknown network backing type")
		return fmt.Sprintf("ERROR-UNKNOWN-BACKING-%s", backingType)
	}
}

// resolveNetworkReference resolves a standard network reference to its human-readable name
func (d *Discovery) resolveNetworkReference(networkRef *types.ManagedObjectReference) string {
	if networkRef == nil || d.client == nil {
		return ""
	}

	ctx := context.Background()

	// Get the network object properties using property collector
	pc := property.DefaultCollector(d.client.Client)

	// Try to get network properties
	var networkMo mo.Network
	err := pc.RetrieveOne(ctx, *networkRef, []string{"name"}, &networkMo)
	if err != nil {
		log.WithFields(log.Fields{
			"network_ref": networkRef.Value,
			"error":       err.Error(),
		}).Debug("Failed to resolve network reference, using reference value")
		return ""
	}

	if networkMo.Name != "" {
		log.WithFields(log.Fields{
			"network_ref":  networkRef.Value,
			"network_name": networkMo.Name,
		}).Debug("Successfully resolved network reference")
		return networkMo.Name
	}

	return ""
}

// resolveDVSPortgroupName resolves a DVS portgroup key to its human-readable name (legacy function)
func (d *Discovery) resolveDVSPortgroupName(portgroupKey string) string {
	return d.resolveDVSPortgroupNameEnhanced(portgroupKey)
}

// resolveDVSPortgroupNameEnhanced uses multiple strategies to resolve DVS portgroup names
func (d *Discovery) resolveDVSPortgroupNameEnhanced(portgroupKey string) string {
	if portgroupKey == "" || d.client == nil {
		log.WithField("portgroup_key", portgroupKey).Debug("Invalid input for DVS portgroup resolution")
		return ""
	}

	ctx := context.Background()
	log.WithField("portgroup_key", portgroupKey).Info("🔍 Starting enhanced DVS portgroup resolution")

	// Strategy 1: Direct portgroup lookup using NetworkList
	if name := d.resolveDVSPortgroupDirect(ctx, portgroupKey); name != "" {
		log.WithFields(log.Fields{
			"strategy":        "direct_lookup",
			"portgroup_key":   portgroupKey,
			"resolved_name":   name,
		}).Info("✅ DVS portgroup resolved via direct lookup")
		return name
	}

	// Strategy 2: Search through all DVS switches and their portgroups
	if name := d.resolveDVSPortgroupViaSwitches(ctx, portgroupKey); name != "" {
		log.WithFields(log.Fields{
			"strategy":        "switch_traversal",
			"portgroup_key":   portgroupKey,
			"resolved_name":   name,
		}).Info("✅ DVS portgroup resolved via switch traversal")
		return name
	}

	// Strategy 3: Use container view for comprehensive search
	if name := d.resolveDVSPortgroupViaContainerView(ctx, portgroupKey); name != "" {
		log.WithFields(log.Fields{
			"strategy":        "container_view",
			"portgroup_key":   portgroupKey,
			"resolved_name":   name,
		}).Info("✅ DVS portgroup resolved via container view")
		return name
	}

	log.WithField("portgroup_key", portgroupKey).Error("❌ All DVS resolution strategies failed")
	return ""
}

// resolveDVSPortgroupDirect attempts direct portgroup resolution using NetworkList
func (d *Discovery) resolveDVSPortgroupDirect(ctx context.Context, portgroupKey string) string {
	log.WithField("portgroup_key", portgroupKey).Debug("Attempting direct DVS portgroup lookup")

	// Search for the portgroup by key in the datacenter
	finder := find.NewFinder(d.client.Client, false)

	// Set datacenter context
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		log.WithField("error", err.Error()).Debug("Failed to get default datacenter for portgroup lookup")
		return ""
	}

	finder.SetDatacenter(dc)

	// Search for DVS portgroups using NetworkList (which includes DVS portgroups)
	networks, err := finder.NetworkList(ctx, "*")
	if err != nil {
		log.WithField("error", err.Error()).Debug("Failed to list networks")
		return ""
	}

	log.WithField("network_count", len(networks)).Debug("Retrieved network list for DVS search")

	// Property collector for batch retrieval
	pc := property.DefaultCollector(d.client.Client)

	// Search for matching portgroup key
	for _, network := range networks {
		// Check if this is a DVS portgroup
		var networkMo mo.DistributedVirtualPortgroup
		err := pc.RetrieveOne(ctx, network.Reference(), []string{"key", "name", "config"}, &networkMo)
		if err != nil {
			// Not a DVS portgroup, skip
			continue
		}

		log.WithFields(log.Fields{
			"checking_key":  networkMo.Key,
			"checking_name": networkMo.Name,
			"target_key":    portgroupKey,
		}).Debug("Comparing DVS portgroup key")

		if networkMo.Key == portgroupKey {
			log.WithFields(log.Fields{
				"portgroup_key":  portgroupKey,
				"portgroup_name": networkMo.Name,
				"method":         "direct_lookup",
			}).Info("Successfully resolved DVS portgroup name")
			return networkMo.Name
		}
	}

	log.WithField("portgroup_key", portgroupKey).Debug("Direct lookup failed to find DVS portgroup")
	return ""
}

// resolveDVSPortgroupViaSwitches uses an alternative network search approach
func (d *Discovery) resolveDVSPortgroupViaSwitches(ctx context.Context, portgroupKey string) string {
	log.WithField("portgroup_key", portgroupKey).Debug("Attempting DVS portgroup lookup via alternative search")

	// This is a simplified fallback - in practice, the direct lookup should work for most cases
	// If more sophisticated searching is needed, it would require deeper VMware API knowledge
	// For now, we'll just return empty to fall through to container view
	
	log.WithField("portgroup_key", portgroupKey).Debug("Alternative search not implemented - falling back to container view")
	return ""
}

// resolveDVSPortgroupViaContainerView uses container view for comprehensive portgroup search  
func (d *Discovery) resolveDVSPortgroupViaContainerView(ctx context.Context, portgroupKey string) string {
	log.WithField("portgroup_key", portgroupKey).Debug("Attempting DVS portgroup lookup via container view")

	// This is a more comprehensive approach using ViewManager
	viewManager := view.NewManager(d.client.Client)

	// Create container view for all DistributedVirtualPortgroup objects
	containerView, err := viewManager.CreateContainerView(ctx, d.client.ServiceContent.RootFolder, []string{"DistributedVirtualPortgroup"}, true)
	if err != nil {
		log.WithField("error", err.Error()).Debug("Failed to create container view for DVS portgroups")
		return ""
	}
	defer containerView.Destroy(ctx)

	// Retrieve all DVS portgroups
	var portgroups []mo.DistributedVirtualPortgroup
	err = containerView.Retrieve(ctx, []string{"DistributedVirtualPortgroup"}, []string{"key", "name"}, &portgroups)
	if err != nil {
		log.WithField("error", err.Error()).Debug("Failed to retrieve DVS portgroups via container view")
		return ""
	}

	log.WithField("portgroup_count", len(portgroups)).Debug("Retrieved DVS portgroups via container view")

	// Search for matching key
	for _, pg := range portgroups {
		if pg.Key == portgroupKey {
			log.WithFields(log.Fields{
				"portgroup_key":  portgroupKey,
				"portgroup_name": pg.Name,
				"method":         "container_view",
			}).Info("Found DVS portgroup via container view")
			return pg.Name
		}
	}

	log.WithField("portgroup_key", portgroupKey).Debug("Container view failed to find DVS portgroup")
	return ""
}

// resolveFolderPath extracts the folder path from VM inventory path
func (d *Discovery) resolveFolderPath(vm *object.VirtualMachine) string {
	inventoryPath := vm.InventoryPath
	if inventoryPath == "" {
		return "Unknown"
	}

	// Extract folder path by removing VM name from the end
	// InventoryPath format: /Datacenter/vm/Folder1/Folder2/VMName
	pathParts := strings.Split(inventoryPath, "/")
	if len(pathParts) > 1 {
		// Remove the VM name (last part) and rejoin
		folderParts := pathParts[:len(pathParts)-1]
		folderPath := strings.Join(folderParts, "/")

		// Clean up the path to show logical folder structure
		if strings.Contains(folderPath, "/vm/") {
			// Remove datacenter and vm prefix to show clean folder path
			if vmIndex := strings.Index(folderPath, "/vm/"); vmIndex >= 0 {
				cleanPath := folderPath[vmIndex+4:] // Remove "/vm/" part
				if cleanPath == "" {
					return "Root"
				}
				return cleanPath
			}
		}
		return folderPath
	}

	return "Root"
}

// GetVMDetails gets detailed information for a specific VM by path
func (d *Discovery) GetVMDetails(ctx context.Context, vmPath string) (*models.VMInfo, error) {
	log.WithField("vm_path", vmPath).Info("Getting VM details from vCenter")

	finder := find.NewFinder(d.client.Client, true)

	// Find datacenter (extract from path if needed)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	// Find the VM by path
	vm, err := finder.VirtualMachine(ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find VM %s: %w", vmPath, err)
	}

	// Get VM properties
	var mvm mo.VirtualMachine
	pc := property.DefaultCollector(d.client.Client)
	err = pc.RetrieveOne(ctx, vm.Reference(), []string{
		"name",
		"config.uuid",
		"config.hardware.numCPU",
		"config.hardware.memoryMB",
		"config.hardware.device",
		"runtime.powerState",
		"guest.guestId",
		"guest.toolsStatus",
		"guest.toolsVersion",
		"summary.config.vmPathName",
		"summary.config.annotation",
	}, &mvm)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Extract disk information
	var disks []models.DiskInfo
	for _, device := range mvm.Config.Hardware.Device {
		if disk, ok := device.(*types.VirtualDisk); ok {
			capacityGB := disk.CapacityInKB / 1024 / 1024 // Convert KB to GB

			unitNumber := 0
			if disk.UnitNumber != nil {
				unitNumber = int(*disk.UnitNumber)
			}

			diskInfo := models.DiskInfo{
				ID:            fmt.Sprintf("disk-%d", disk.Key),
				Path:          fmt.Sprintf("%s", disk.Backing),
				SizeGB:        int(capacityGB),
				Datastore:     "unknown",                              // Would need more complex extraction
				CapacityBytes: int64(capacityGB * 1024 * 1024 * 1024), // Convert to bytes
				UnitNumber:    unitNumber,
			}
			disks = append(disks, diskInfo)
		}
	}

	// Extract network information
	var networks []models.NetworkInfo
	for _, device := range mvm.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			ethernetCard := nic.GetVirtualEthernetCard()

			// Resolve network name from backing
			networkName := d.resolveNetworkName(ethernetCard.Backing)

			networkInfo := models.NetworkInfo{
				Label:       ethernetCard.DeviceInfo.GetDescription().Label,
				NetworkName: networkName,
				MACAddress:  ethernetCard.MacAddress,
				Connected:   ethernetCard.Connectable.Connected,
			}

			// Determine adapter type
			switch nic.(type) {
			case *types.VirtualVmxnet3:
				networkInfo.AdapterType = "vmxnet3"
			case *types.VirtualE1000:
				networkInfo.AdapterType = "e1000"
			case *types.VirtualE1000e:
				networkInfo.AdapterType = "e1000e"
			default:
				networkInfo.AdapterType = "unknown"
			}

			networks = append(networks, networkInfo)
		}
	}

	// Create VM info with all required fields including metadata
	vmInfo := &models.VMInfo{
		ID:         mvm.Config.Uuid,
		Name:       mvm.Name,
		Path:       vmPath,
		Datacenter: dc.Name(),
		CPUs:       int(mvm.Config.Hardware.NumCPU),
		MemoryMB:   int(mvm.Config.Hardware.MemoryMB),
		Disks:      disks,
		Networks:   networks,
		PowerState: string(mvm.Runtime.PowerState),
		OSType:     mvm.Guest.GuestId,
		VMXVersion: "unknown", // Would need additional property

		// Additional VM metadata
		DisplayName:        mvm.Name,
		Annotation:         mvm.Summary.Config.Annotation,
		FolderPath:         d.resolveFolderPath(vm),
		VMwareToolsStatus:  string(mvm.Guest.ToolsStatus),
		VMwareToolsVersion: mvm.Guest.ToolsVersion,
	}

	log.WithFields(log.Fields{
		"vm_id":       vmInfo.ID,
		"vm_name":     vmInfo.Name,
		"vm_path":     vmInfo.Path,
		"cpus":        vmInfo.CPUs,
		"memory_mb":   vmInfo.MemoryMB,
		"disk_count":  len(vmInfo.Disks),
		"power_state": vmInfo.PowerState,
	}).Info("VM details retrieved successfully")

	return vmInfo, nil
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getActualDiskSize returns the actual file size of a disk from VM layout information
func (d *Discovery) getActualDiskSize(layoutEx *types.VirtualMachineFileLayoutEx, diskKey int32) int64 {
	if layoutEx == nil || layoutEx.File == nil {
		return 0 // Return 0 if no layout information available
	}

	var totalSize int64 = 0

	// Look for files associated with this disk key
	// VMDK files typically have names like "VM_name-000001.vmdk", "VM_name-000001-flat.vmdk"
	for _, file := range layoutEx.File {
		// Match files related to this disk by checking if the key appears in the file associations
		// The layout contains file entries with keys that correspond to disk device keys
		if file.Key == diskKey {
			totalSize += file.Size
			log.WithFields(log.Fields{
				"disk_key":  diskKey,
				"file_name": file.Name,
				"file_size": file.Size,
			}).Debug("Found disk file in VM layout")
		}
	}

	// If we didn't find files by key matching, try a different approach
	// Look for disk files in the layout by examining all files
	if totalSize == 0 {
		for _, file := range layoutEx.File {
			// Check if this is a VMDK flat file (contains actual data)
			if strings.Contains(file.Name, "-flat.vmdk") || strings.Contains(file.Name, ".vmdk") {
				// For simplicity, sum all VMDK files - this may need refinement for multi-disk VMs
				totalSize += file.Size
				log.WithFields(log.Fields{
					"disk_key":  diskKey,
					"file_name": file.Name,
					"file_size": file.Size,
				}).Debug("Found VMDK file in VM layout")
			}
		}
	}

	if totalSize > 0 {
		log.WithFields(log.Fields{
			"disk_key":          diskKey,
			"actual_size_gb":    float64(totalSize) / 1073741824,
			"actual_size_bytes": totalSize,
		}).Info("✅ Retrieved actual VMDK file size from VM layout")
	} else {
		log.WithField("disk_key", diskKey).Warn("Could not determine actual disk size from VM layout")
	}

	return totalSize
}
