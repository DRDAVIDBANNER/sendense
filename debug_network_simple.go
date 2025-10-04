package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func main() {
	// vCenter connection
	u, _ := url.Parse("https://quad-vcenter-01.quadris.local/sdk")
	u.User = url.UserPassword("administrator@vsphere.local", "EmyGVoBFesGQc47-")

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatal(err)
	}

	finder := find.NewFinder(client.Client, true)
	dc, _ := finder.Datacenter(ctx, "DatabanxDC")
	finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(ctx, "QCDEV-AUVIK01")
	if err != nil {
		log.Fatal(err)
	}

	var mvm mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)
	err = pc.RetrieveOne(ctx, vm.Reference(), []string{"config.hardware.device"}, &mvm)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== QCDEV-AUVIK01 Network Analysis ===")

	for _, device := range mvm.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			card := nic.GetVirtualEthernetCard()
			fmt.Printf("\nNetwork Adapter: %s\n", card.DeviceInfo.GetDescription().Label)
			fmt.Printf("MAC: %s\n", card.MacAddress)
			fmt.Printf("Connected: %v\n", card.Connectable.Connected)

			backing := card.Backing
			if backing == nil {
				fmt.Println("❌ No backing found")
				continue
			}

			fmt.Printf("Backing Type: %T\n", backing)

			switch b := backing.(type) {
			case *types.VirtualEthernetCardNetworkBackingInfo:
				fmt.Println("📡 Standard Network Backing")
				fmt.Printf("Device Name: '%s'\n", b.DeviceName)
				if b.Network != nil {
					fmt.Printf("Network Reference: %s\n", b.Network.Value)
					fmt.Printf("Network Type: %s\n", b.Network.Type)

					// Try to resolve network name
					var netMo mo.Network
					err := pc.RetrieveOne(ctx, *b.Network, []string{"name", "summary"}, &netMo)
					if err != nil {
						fmt.Printf("❌ Failed to resolve network: %v\n", err)
					} else {
						fmt.Printf("✅ Resolved Name: '%s'\n", netMo.Name)
						if netMo.Summary != nil {
							fmt.Printf("Network Summary: %+v\n", netMo.Summary)
						}
					}
				}

			case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
				fmt.Println("📡 DVS Backing")
				fmt.Printf("Portgroup Key: %s\n", b.Port.PortgroupKey)
				fmt.Printf("Switch UUID: %s\n", b.Port.SwitchUuid)

			default:
				fmt.Printf("❓ Unknown backing: %T\n", backing)
			}
		}
	}
}
